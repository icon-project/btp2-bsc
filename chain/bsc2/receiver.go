package bsc2

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	lru "github.com/hashicorp/golang-lru"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/icon-project/btp2/common/db"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/mta"
	btp "github.com/icon-project/btp2/common/types"
)

const (
	CacheSize          = 1024
	CheckpointInterval = 256
	DBName             = string("db")
	BucketName         = db.BucketID("accumulator")
	AccStateKey        = string("accumulator")
)

var EmptyHash = common.Hash{}

type RecvConfig struct {
	ChainID     uint64 `json:"chain_id"`
	Epoch       uint64 `json:"epoch"`
	StartNumber uint64 `json:"start_height"`
	Endpoint    string `json:"endpoint"`
	DBType      string `json:"db_type"`
	DBPath      string `json:"db_path"`
	SrcAddress  btp.BtpAddress
	DstAddress  btp.BtpAddress
}

type receiver struct {
	chainId     *big.Int
	epoch       uint64
	startnumber uint64

	client      *Client
	accumulator *mta.ExtAccumulator
	database    db.Database
	snapshots   *Snapshots
	heads       *lru.ARCCache
	msgs        *lru.ARCCache
	mutex       sync.Mutex
	peer        *bstatus
	local       *bstatus
	cond        *sync.Cond
	log         log.Logger
}

func newReceiver(config RecvConfig, log log.Logger) *receiver {
	o := &receiver{
		chainId:     new(big.Int).SetUint64(config.ChainID),
		epoch:       config.Epoch,
		startnumber: config.StartNumber,
		client:      NewClient(config.Endpoint, config.SrcAddress, config.DstAddress, log),
		local:       &bstatus{},
		peer:        &bstatus{},
		cond:        sync.NewCond(&sync.Mutex{}),
		log:         log,
	}

	if cache, err := lru.NewARC(CacheSize); err != nil {
		o.log.Panicf("fail to create lru head cache - size(%d) err(%s)", CacheSize, err.Error())
	} else {
		o.heads = cache
	}

	if cache, err := lru.NewARC(CacheSize); err != nil {
		o.log.Panicf("fail to create lru msg cache - size(%d) err(%s)", CacheSize, err.Error())
	} else {
		o.msgs = cache
	}

	o.log.Infof("create bsc receiver - chainid(%d), epoch(%d), startnumber(%d)\n",
		o.chainId.Uint64(), o.epoch, o.startnumber)

	o.log.Infof("open database - path(%s) type(%s)\n", config.DBPath, string(db.GoLevelDBBackend))
	if database, err := db.Open(config.DBPath, string(db.GoLevelDBBackend), DBName); err != nil {
		o.log.Panicf("fail to open database - err(%s)\n", err.Error())
	} else {
		o.database = database
		if bucket, err := database.GetBucket(BucketName); err != nil {
			o.log.Panicf("fail to get bucket - err(%s)\n", err.Error())
		} else {
			o.accumulator = mta.NewExtAccumulator([]byte(AccStateKey), bucket, int64(o.startnumber))
			if ok, err := bucket.Has([]byte(AccStateKey)); ok {
				if err = o.accumulator.Recover(); err != nil {
					o.log.Panicf("fail to recover accumulator - err(%s)", err.Error())
				}
			} else if err != nil {
				o.log.Panicf("fail to access bucket - err(%s)", err.Error())
			}
			o.log.Infof("Current accumulator height(%d) offset(%d)\n",
				o.accumulator.Height(), o.accumulator.Offset())
		}
	}
	o.snapshots = newSnapshots(o.chainId, o.client.Client, CacheSize, o.database, o.log)
	return o
}

func (o *receiver) Start(peer *btp.BMCLinkStatus) (<-chan interface{}, error) {
	och := make(chan interface{})
	o.local = newStatus(peer)
	o.peer = newStatus(peer)
	o.log.Infof("peer bmc status - block(%d:%s), sequence(%d)",
		o.peer.number, o.peer.hash, o.peer.sequence)
	o.log.Infoln("prepare receiver...")
	if err := o.prepare(); err != nil {
		o.log.Panicf("fail to prepare receiver - err(%s)", err)
	}

	// synchronize local status until finality of peer
	o.log.Infoln("synchronize receiver status...")
	o.log.Infof("Sync recv status - target(%d:%s)", o.peer.number, o.peer.hash)
	if err := o.synchronize(new(big.Int).SetUint64(o.peer.number)); err != nil {
		o.log.Panicf("fail to synchronize status - err(%s)", err)
	}
	o.log.Infof("Done - sync recv status")

	// check node connection
	if latest, err := o.client.BlockNumber(context.Background()); err != nil {
		o.log.Panicf("fail to connect to node - err(%s)", err)
	} else {
		o.log.Infof("latest block number on bsc network - number(%d)", latest)
	}

	// fetch missing messages
	if msgs, err := o.client.MessagesAfterSequence(&bind.FilterOpts{
		Start:   o.startnumber,
		End:     &o.peer.number,
		Context: context.Background(),
	}, o.peer.sequence); err != nil {
		o.log.Panicf("fail to fetch missing messages - err(%s)", err)
	} else {
		for _, msg := range msgs {
			o.log.Infof("unreached message - block(%d:%s) sequence(%d)", msg.Raw.BlockNumber, msg.Raw.BlockHash.Hex(), msg.Seq.Uint64())
			o.msgs.Add(msg.Seq.Uint64(), msg)
		}

		if len(msgs) > 0 {
			o.local.sequence = msgs[len(msgs)-1].Seq.Uint64()
			go func() {
				o.log.Debugf("notification for unreached messages - status(%d:%s:%d)",
					o.local.number, o.local.hash, o.local.cache)
				och <- o.local.clone()
			}()
		}
	}

	go func() {
		for {
			if err := recoverable(o.loop(o.peer.hash, och)); err != ErrRecoverable {
				o.log.Panicf("Fault Receiver Loop(%s)", err.Error())
			}
		}
	}()
	return och, nil
}

func (o *receiver) recoverable(err error) bool {

	counter := make(map[error]uint64)
	go func() {
		ticker := time.NewTicker(time.Second * 60)
		for t := range ticker.C {
			o.log.Println("Tick at", t)
			for _, v := range counter {
				if v > 10 {
					o.log.Panicf("Too many same errors(%v)", counter)
				} else {
					o.log.Infoln("TICKER~~")
				}
			}
		}
	}()

	if err == ErrInconsistentBlock {
		o.log.Warnln("RecoverInconsistentChain")
		return true
	} else if err == ErrRecoverable {
		o.log.Warnln("RecoverableErr")
		return true
	}
	return false
}

func (o *receiver) applyAndCache(snap *Snapshot, head *types.Header) (*Snapshot, error) {
	next, err := snap.apply(head, o.chainId)
	if err != nil {
		o.log.Warnf("fail to apply snapshot - err(%s)", err)
		return nil, err
	}
	if err := o.snapshots.add(next); err != nil {
		o.log.Warnf("fail to add snapshot - err(%s)", err)
		return nil, err
	}

	o.heads.Add(head.Number.Uint64(), head)
	return next, nil
}

func (o *receiver) calcLatestFinality(calc *BlockFinalityCalculator, feed common.Hash) (common.Hash, error) {
	if finalities, err := calc.feed(feed); err != nil {
		o.log.Errorf("fail to calculate finality - err(%s)", err)
		return EmptyHash, err
	} else if len(finalities) <= 0 {
		return EmptyHash, nil
	} else {
		return finalities[len(finalities)-1], nil
	}
}

var ErrInconsistentCount = 0

func (o *receiver) loop(hash common.Hash, och chan<- interface{}) error {
	o.log.Tracef("StartReceiverLoop")
	headCh := make(chan *types.Header)
	calc := newBlockFinalityCalculator(hash, make([]common.Hash, 0), o.snapshots, o.log)

	snap, err := o.snapshots.get(hash)
	if err != nil {
		o.log.Panicf("NoSnapshot(%s)", hash)
	}
	o.log.Infof("InitialFinality - S(%d:%s)", snap.Number, snap.Hash)
	sub := o.client.WatchHeader(context.Background(), new(big.Int).SetUint64(snap.Number+1), headCh)
	FeedCounter := uint64(0)
	for {
		select {
		case head := <-headCh:
			FeedCounter++
			o.waitIfFullBuf(head.Number.Uint64())
			var err error
			if snap, err = o.applyAndCache(snap, head); err != nil {
				o.log.Warnf("Fail to apply and cache head - err(%s)", err.Error())
				sub.Unsubscribe()
				return err
			}

			fnzs, err := calc.feed(snap.Hash)
			if err != nil {
				o.log.Errorf("Fail to calc finality - feed(%s) err(%s)", snap.Hash, err.Error())
				sub.Unsubscribe()
				return err
			} else if len(fnzs) <= 0 {
				break
			}
			final := fnzs[len(fnzs)-1]
			var number uint64
			var hash common.Hash
			if snap, err := o.snapshots.get(final); err != nil {
				o.log.Panicln(err.Error())
			} else {
				// latest finalized block number
				o.local.cache = snap.Number

				// To prevent system fault caused by forked block stores past finalized block number
				// that has been finalized from latest finalized block
				number = snap.Attestation.SourceNumber
				hash = snap.Attestation.SourceHash

				if number <= o.local.number {
					break
				}
			}

			if sequence, err := o.queryAndCacheMessages(hash, o.local.hash); err != nil {
				o.log.Warnf("Fail to fetch and cache messages - err(%s)", err.Error())
				sub.Unsubscribe()
				return ErrRecoverable
			} else {
				o.updateStatus(number, hash, sequence)
				if sequence != 0 || FeedCounter > 50 {
					FeedCounter = 0
					time.Sleep(time.Second * 2)
					o.log.Tracef("Notify(%d:%d:%s:%d)",
						o.local.number, o.local.cache, o.local.hash, o.local.sequence)
					och <- &bstatus{
						number:   o.local.number,
						sequence: o.local.sequence,
					}
				}
			}
		case err := <-sub.Err():
			o.log.Warnf("Fail to watching new heads - err(%s)", err)
			return err
		}
	}
	return nil
}

func (o *receiver) updateStatus(number uint64, hash common.Hash, sequence uint64) {
	o.local.number = number
	o.local.hash = hash
	if sequence != 0 {
		o.local.sequence = sequence
	}
}

func (o *receiver) queryAndCacheMessages(child, ancestor common.Hash) (uint64, error) {
	var sequence uint64
	for child != ancestor {
		snap, _ := o.snapshots.get(child)
		o.log.Debugf("Query Message BlockHash(%d:%s)", snap.Number, snap.Hash)
		ms, err := o.client.MessagesByBlockHash(context.Background(), child)
		if err != nil {
			return sequence, err
		}

		for _, m := range ms {
			seq := m.Seq.Uint64()
			o.log.Infof("NewMessage - M(%d:%d:%.8s)",
				seq, m.Raw.BlockNumber, m.Raw.BlockHash.Hex())
			o.msgs.Add(seq, m)
			if seq != 0 {
				sequence = seq
			}
		}

		if snap, err := o.snapshots.get(child); err != nil {
			return sequence, err
		} else {
			child = snap.ParentHash
		}
	}
	return sequence, nil
}

func (o *receiver) Stop() {
	// TODO
}

func (o *receiver) GetStatus() (link.ReceiveStatus, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	status := &bstatus{
		number:   o.local.number,
		sequence: o.local.sequence,
	}
	return status, nil
}

func (o *receiver) GetHeightForSeq(sequence int64) int64 {
	o.log.Tracef("++GetHeightForSequence(%d)", sequence)
	defer o.log.Tracef("--GetHeightForSequence(%d)", sequence)
	if msg, ok := o.msgs.Get(uint64(sequence)); ok {
		m := msg.(*BTPMessageCenterMessage)
		o.log.Tracef("message(%d) block(%d:%s)", sequence, m.Raw.BlockNumber, m.Raw.BlockHash)
		return int64(m.Raw.BlockNumber)
	}

	o.log.Warnf("no cache message")
	o.mutex.Lock()
	defer o.mutex.Unlock()
	end := o.local.number
	if msg, err := o.client.MessageBySequence(&bind.FilterOpts{
		Start:   o.startnumber,
		End:     &end,
		Context: context.Background(),
	}, uint64(sequence)); err != nil {
		o.log.Errorf("fail to fetch message - err(%s) start(%d) sequence(%d)", err.Error(), o.startnumber, sequence)
		return 0
	} else if msg == nil {
		o.log.Warnf("no such message(%d)", sequence)
		return 0
	} else {
		o.log.Tracef("found message(%d:%s)", msg.Raw.BlockNumber, msg.Raw.BlockHash)
		return int64(msg.Raw.BlockNumber)
	}
}

func (o *receiver) selectFork(from common.Hash, blocks *BlockTree) []common.Hash {
	feeds := make([]common.Hash, 0)
	for {
		children := blocks.ChildrenOf(from)
		if len(children) <= 0 {
			break
		}

		var hash common.Hash
		for _, child := range children {
			snap, err := o.snapshots.get(child)
			if err == nil {
				hash = snap.Hash
				break
			}
		}

		if hash != EmptyHash {
			from = hash
			feeds = append(feeds, hash)
		} else {
			break
		}
	}
	return feeds
}

func (o *receiver) hasMessages(finalities []common.Hash) (bool, error) {
	for _, finality := range finalities {
		msgs, err := o.client.MessagesByBlockHash(context.Background(), finality)
		if err != nil {
			o.log.Errorf("fail to fetch messages for block(%s) - err(%s)", finality, err)
			return false, err
		}

		if len(msgs) > 0 {
			o.log.Infof("block(%d:%s) contains messages", msgs[0].Raw.BlockNumber, msgs[0].Raw.BlockHash)
			return true, nil
		}
	}
	return false, nil
}

func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	peer := newStatus(status)
	o.log.Tracef("Build block updates - P(%d:%.8s) L(%d:%d:%.8s)",
		peer.number, peer.blocks.Root().Hex(), o.local.number, o.local.cache, o.local.hash)
	defer o.log.Tracef("Done - Build block updates")
	if _, err := o.snapshots.get(peer.hash); err != nil {
		o.log.Errorf("No header for finalized block number(%d)", peer.number)
		return nil, errors.New("NoCachedHeader")
	}

	canonical := o.selectFork(peer.hash, peer.blocks)
	o.log.Tracef("Canonical(%v)", canonical)
	calc := newBlockFinalityCalculator(peer.hash, canonical, o.snapshots, o.log)

	heads := make([]*types.Header, 0)
	parent := peer.hash
	if len(canonical) > 0 {
		parent = canonical[len(canonical)-1]
	}
	finalities := uint64(0)
	number := peer.number + uint64(len(canonical)) + 1
	exists := false
	for number <= o.local.cache && !exists {
		var head *types.Header
		if hd, ok := o.heads.Get(number); ok {
			head = hd.(*types.Header)
		} else {
			o.log.Warnf("no header(%d)", number)
			break
		}

		o.log.Tracef("Head(%d:%.8s:%.8s)", head.Number.Uint64(), head.Hash().Hex(), head.ParentHash.Hex())
		hash := head.Hash()
		if parent != head.ParentHash {
			o.log.Panicf("InconsistentHead - P(%.8s) C(%d:%.8s:%.8s)",
				parent.Hex(), head.Number.Uint64(), head.Hash().Hex(), head.ParentHash.Hex())

			return nil, errors.New("InconsistentChain")
		}

		if size := int64(math.Ceil(float64(head.Size()))) * 5; limit >= size {
			limit -= size
		} else {
			break
		}

		heads = append(heads, head)
		if err := peer.blocks.Add(head.ParentHash, hash); err != nil {
			o.log.Panicf(err.Error())
		}
		if fnzs, err := calc.feed(hash); err != nil {
			o.log.Panicf("Fail to calculate finality - feed(%s) err(%s)", hash, err)
		} else if len(fnzs) > 0 {
			finalities += uint64(len(fnzs))
			peer.blocks.Prune(fnzs[len(fnzs)-1])
			for i := 0; i < len(fnzs) && !exists; i++ {
				if msgs, err := o.client.MessagesByBlockHash(context.Background(), fnzs[i]); err != nil {
					o.log.Errorf("fail to fetch message - block(%s) err(%s)", fnzs[i], err)
					return nil, err
				} else if len(msgs) > 0 {
					exists = true
					break
				}
			}
		}

		number++
		parent = hash
	}

	if len(heads) > 0 {
		o.log.Infof("BU{ H(%d:%.8s) ~ H(%d:%.8s) F(%d:%.8s) }",
			heads[0].Number.Uint64(), heads[0].Hash().Hex(),
			heads[len(heads)-1].Number.Uint64(), heads[len(heads)-1].Hash().Hex(),
			peer.number+finalities, peer.blocks.Root().Hex())
		return []link.BlockUpdate{
			&BlockUpdate{
				heads:  heads,
				start:  peer.number + 1,
				height: peer.number + finalities,
				status: &VerifierStatus{
					Offset:    uint64(o.accumulator.Offset()),
					BlockTree: peer.blocks,
				},
			},
		}, nil
	}
	o.log.Tracef("NoAvailableHead")
	return []link.BlockUpdate{}, nil
}

func (o *receiver) BuildBlockProof(status *btp.BMCLinkStatus, target int64) (link.BlockProof, error) {
	peer := newStatus(status)
	o.log.Tracef("Build block proof - P(%d:%.8s:%d)",
		peer.number, peer.hash, peer.sequence)
	defer o.log.Tracef("Done - Build block proof")

	if err := o.syncAcc(peer.hash); err != nil {
		return nil, err
	}
	_, witness, err := o.accumulator.WitnessForAt(target+1, int64(peer.number+1), o.accumulator.Offset())
	if err != nil {
		o.log.Errorf("fail to load witness - err(%s)", err)
		return nil, err
	}

	if head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(target)); err != nil {
		o.log.Errorf("fail to find header(%d) - err(%s)", target, err)
		return nil, err
	} else {
		o.log.Infof("BP{ H(%d:%s) }", head.Number.Uint64(), head.Hash())
		return &BlockProof{
			Header:    head,
			AccHeight: peer.number + 1,
			Witness:   mta.WitnessesToHashes(witness),
		}, nil
	}
}

func (o *receiver) BuildMessageProof(status *btp.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	o.log.Tracef("Build message proof - S(%d: %d: %d)", status.Verifier.Height, status.RxSeq, limit)
	defer o.log.Tracef("Done - Build message proof")
	sequence := uint64(status.RxSeq)
	var msgs []*BTPMessageCenterMessage
	for {
		sequence++
		raw, ok := o.msgs.Get(sequence)
		if !ok {
			o.log.Tracef("no message(%d)", sequence)
			break
		}
		msg := raw.(*BTPMessageCenterMessage)
		if uint64(status.Verifier.Height) < msg.Raw.BlockNumber {
			o.log.Tracef("message in unfianlized block")
			break
		}

		if msgs != nil && msgs[0].Raw.BlockNumber != msg.Raw.BlockNumber {
			break
		} else if msgs == nil {
			msgs = make([]*BTPMessageCenterMessage, 0)
		}
		o.log.Debugf("append message to message proof - number(%d) hash(%s) sequence(%d)",
			msg.Raw.BlockNumber, msg.Raw.BlockHash.Hex(), msg.Seq.Uint64())
		msgs = append(msgs, msg)
	}

	if len(msgs) <= 0 {
		return nil, nil
	}

	hash := msgs[0].Raw.BlockHash
	receipts, err := o.client.ReceiptsByBlockHash(context.Background(), hash)
	if err != nil {
		o.log.Errorf("fail to fetch receipts - hash(%s), err(%+v)", hash.Hex(), err)
		return nil, err
	}

	mpt, err := newMTPWithReceipts(receipts)
	if err != nil {
		o.log.Errorf("fail to create receipts mpt - hash(%s), err(%+v)", hash.Hex(), err)
		return nil, err
	}

	o.log.Debugf("Calc ReceiptsRoot:(%s)", mpt.Hash())
	if h, e := o.client.HeaderByHash(context.Background(), hash); e != nil {
		panic(fmt.Sprintf("fail to fetching header(%s), err(%s)", hash.Hex(), e.Error()))
	} else {
		o.log.Debugf("H=> %d:%s:%s", h.Number.Uint64(), h.Hash().Hex(), h.ReceiptHash.Hex())
	}

	var msgproof *MessageProof
	for _, msg := range msgs {
		key, err := rlp.EncodeToBytes(msg.Raw.TxIndex)
		if err != nil {
			return nil, err
		}

		proof, err := newProofOf(mpt, key)
		if err != nil {
			o.log.Errorf("fail to create merkle proof - mpt(%+v), key(%s), err(%+v)",
				mpt, hex.EncodeToString(key), err)
			return nil, err
		}

		// TODO FIXME not exact size...
		size := int64(0)
		for i := 0; i < len(proof); i++ {
			size += int64(len(proof[i]))
		}
		if msgproof != nil && limit < size {
			break
		}
		limit -= size

		if msgproof == nil {
			msgproof = &MessageProof{
				Hash:     hash,
				Proofs:   make([]BSCReceiptProof, 0),
				sequence: msg.Seq.Uint64(),
			}
		}
		msgproof.Proofs = append(msgproof.Proofs, BSCReceiptProof{
			Key:   key,
			Proof: proof,
		})
	}
	if msgproof != nil {
		o.log.Debugf("MP{ block(%d:%s) nproofs(%d) start sequence(%d)",
			msgs[0].Raw.BlockNumber, msgproof.Hash.Hex(), len(msgproof.Proofs), msgproof.sequence)
	}
	return msgproof, nil
}

func (o *receiver) BuildRelayMessage(parts []link.RelayMessageItem) ([]byte, error) {
	o.log.Tracef("++Recv::BuildRelayMessage - size(%d)\n", len(parts))
	defer o.log.Traceln("--Recv::BuildRelayMessage")
	msg := &BSCRelayMessage{
		TypePrefixedMessages: make([]BSCTypePrefixedMessage, 0),
	}

	for _, part := range parts {
		blob, err := rlp.EncodeToBytes(part)
		if err != nil {
			o.log.Errorf("fail to encode type prefixed message - (%d)\n", typeToUint(part.Type()))
			return nil, err
		}

		o.log.Debugf("pack relay message parts - type(%d)", part.Type())
		msg.TypePrefixedMessages = append(msg.TypePrefixedMessages, BSCTypePrefixedMessage{
			Type:    typeToUint(part.Type()),
			Payload: blob,
		})
	}

	if blob, err := rlp.EncodeToBytes(msg); err != nil {
		o.log.Errorf("fail to encoding relay message - err(%+v)", err)
		return nil, err
	} else {
		return blob, nil
	}
}

func (o *receiver) FinalizedStatus(sch <-chan *btp.BMCLinkStatus) {
	go func() {
		for {
			select {
			case status := <-sch:
				peer := newStatus(status)
				if err := o.syncAcc(peer.hash); err != nil {
					o.log.Errorf("fail to sync accumulator - block(%d:%s) acc(%d)", peer.number, peer.hash, o.accumulator.Height())
				}
				o.purgeAndWakeup(peer)
			}
		}
	}()
}

func (o *receiver) syncAcc(until common.Hash) error {
	end, err := o.client.HeaderByHash(context.Background(), until)
	if err != nil {
		o.log.Errorf("fail to fetch header(%s) - err(%s)", until, err)
		return err
	}

	cursor := o.accumulator.Height()
	for cursor <= end.Number.Int64() {
		if head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(cursor)); err != nil {
			o.log.Errorf("fail to fetch header(%d) - err(%s)", cursor, err)
			return err
		} else {
			o.accumulator.AddHash(head.Hash().Bytes())
		}
		cursor++
	}

	if err := o.accumulator.Flush(); err != nil {
		o.log.Errorf("fail to flush accumulator - (%s)", err)
		return err
	}
	return nil
}

// ensure initial merkle accumulator and snapshot
func (o *receiver) prepare() error {
	offset := big.NewInt(o.accumulator.Offset())
	if offset.Uint64()%o.epoch != 0 {
		o.log.Panicf("accumulator offset must be epoch number - (%d != %d)", offset.Uint64(), o.epoch)
	}

	head, err := o.client.HeaderByNumber(context.Background(), offset)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)", offset.Uint64(), err.Error())
		return err
	}

	number := head.Number.Uint64()
	hash := head.Hash()
	if o.accumulator.Len() <= 0 {
		o.log.Infof("accumulate initial header - number(%d) hash(%s)", number, hash)
		o.accumulator.AddHash(hash.Bytes())
		if err := o.accumulator.Flush(); err != nil {
			o.log.Errorln("fail to flush accumulator - err(%s)", err.Error())
			return err
		}
	}

	ok, err := hasSnapshot(o.database, hash)
	if err != nil {
		o.log.Errorln("fail to check snapshot existent - err(%s)", err.Error())
		return err
	}

	if !ok {
		snap, err := BootSnapshot(o.epoch, head, o.client.Client, o.log)
		if err != nil {
			return err
		}

		if err := snap.store(o.database); err != nil {
			return err
		}

		o.log.Infof("create initial snapshot: number(%d) hash(%s)", snap.Number, snap.Hash)
	}
	return nil
}

func (o *receiver) synchronize(until *big.Int) error {
	// synchronize up to `target` block
	target, err := o.client.HeaderByNumber(context.Background(), until)
	if err != nil {
		return err
	}
	hash := target.Hash()

	// synchronize snapshots
	o.log.Infof("synchronize block snapshots - until(%s)", hash)
	if err := o.snapshots.ensure(hash); err != nil {
		o.log.Errorf("fail to load past snapshots - err(%+v)", err)
		return err
	}

	// synchronize accumulator
	o.log.Infof("synchronize accumulator...")
	if err := o.syncAcc(hash); err != nil {
		return err
	}
	return nil
}

// TODO Add `ToInt()` to `link.MessageItemType`
func typeToUint(_type link.MessageItemType) uint {
	if _type == link.TypeBlockUpdate {
		return 1
	} else if _type == link.TypeBlockProof {
		return 2
	} else if _type == link.TypeMessageProof {
		return 3
	}
	return 0
}

func encodeForDerive(receipts types.Receipts, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	receipts.EncodeIndex(i, buf)
	// It's really unfortunate that we need to do perform this copy.
	// StackTrie holds onto the values until Hash is called, so the values
	// written to it must not alias.
	return common.CopyBytes(buf.Bytes())
}

func newMTPWithReceipts(receipts types.Receipts) (*trie.Trie, error) {
	trie, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	trie.Reset()

	valueBuf := new(bytes.Buffer)
	// StackTrie requires values to be inserted in increasing hash order, which is not the
	// order that `list` provides hashes in. This insertion sequence ensures that the
	// order is correct.
	var indexBuf []byte
	for i := 1; i < receipts.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(receipts, i, valueBuf)
		trie.Update(indexBuf, value)
	}
	if receipts.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(receipts, 0, valueBuf)
		trie.Update(indexBuf, value)
	}
	for i := 0x80; i < receipts.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(receipts, i, valueBuf)
		trie.Update(indexBuf, value)
	}
	return trie, nil
}

func newProofOf(trie *trie.Trie, key []byte) ([][]byte, error) {
	ns := light.NewNodeSet()
	if err := trie.Prove(key, 0, ns); err != nil {
		return nil, err
	}
	proof := make([][]byte, len(ns.NodeList()))
	for i, n := range ns.NodeList() {
		proof[i] = n
	}
	return proof, nil
}

func ToVerifierStatus(blob []byte) *VerifierStatus {
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(blob, vs); err != nil {
		panic(fmt.Sprintf("fail to decode verifier status - blob(%s) err(%+v)\n",
			hex.EncodeToString(blob), err))
	}
	return vs
}

type bstatus struct {
	number   uint64
	hash     common.Hash
	sequence uint64
	blocks   *BlockTree
	cache    uint64
}

func (o *bstatus) clone() *bstatus {
	return &bstatus{
		number:   o.number,
		sequence: o.sequence,
	}
}

func (o *bstatus) update(number uint64, hash common.Hash, sequence uint64) (dirty bool) {
	if number != uint64(0) && o.number != number {
		dirty = true
		o.number = number
		o.hash = hash
	}
	if sequence != uint64(0) && o.sequence != sequence {
		dirty = true
		o.sequence = sequence
	}
	return dirty
}

// Implement ReceiveStatus
func (o *bstatus) Height() int64 {
	return int64(o.number)
}

func (o *bstatus) Seq() int64 {
	return int64(o.sequence)
}

func newStatus(status *btp.BMCLinkStatus) *bstatus {
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		panic(fmt.Sprintf("fail to decode verifier status - blob(%s) err(%s)",
			hex.EncodeToString(status.Verifier.Extra), err.Error()))
	}
	return &bstatus{
		number:   uint64(status.Verifier.Height),
		hash:     vs.BlockTree.Root(),
		sequence: uint64(status.RxSeq),
		blocks:   vs.BlockTree,
	}
}

func (o *receiver) purgeAndWakeup(peer *bstatus) {
	o.cond.L.Lock()
	defer o.cond.L.Unlock()

	o.log.Tracef("Purge Heads(%d ~ %d) Msgs(%d ~ %d)",
		o.peer.number, peer.number, o.peer.sequence, peer.sequence)
	for i := o.peer.number; i <= peer.number; i++ {
		o.heads.Remove(i)
	}
	for i := o.peer.sequence; i <= peer.sequence; i++ {
		o.msgs.Remove(i)
	}

	if o.peer.number*2/3 < peer.number {
		defer func() {
			o.log.Tracef("ResumeWatcher")
			o.cond.Broadcast()
		}()
	}

	o.peer = peer
}

func (o *receiver) waitIfFullBuf(number uint64) {
	o.cond.L.Lock()
	defer func() {
		o.cond.L.Unlock()
	}()
	if o.peer.number+CacheSize < number {
		o.log.Tracef("WaitWatcher...")
		o.cond.Wait()
	}
}
