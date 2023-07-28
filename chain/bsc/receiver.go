package bsc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"golang.org/x/exp/slices"

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
	start       uint64
	chainId     *big.Int
	epoch       uint64
	client      *Client
	accumulator *mta.ExtAccumulator
	cancel      context.CancelFunc
	database    db.Database
	snapshots   *Snapshots
	heads       *lru.ARCCache
	msgs        *lru.ARCCache
	mutex       sync.Mutex
	status      srcStatus
	cond        *sync.Cond
	finality    struct {
		number   uint64
		sequence uint64
	}
	log log.Logger
}

func NewReceiver(config RecvConfig, log log.Logger) *receiver {
	o := &receiver{
		chainId: new(big.Int).SetUint64(config.ChainID),
		epoch:   config.Epoch,
		start:   config.StartNumber,
		client:  NewClient(config.Endpoint, config.SrcAddress, config.DstAddress, log),
		status:  srcStatus{},
		cond:    sync.NewCond(&sync.Mutex{}),
		log:     log,
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
		o.chainId.Uint64(), o.epoch, o.start)

	o.log.Infof("open database - path(%s) type(%s)\n", config.DBPath, string(db.GoLevelDBBackend))
	if database, err := db.Open(config.DBPath, string(db.GoLevelDBBackend), DBName); err != nil {
		o.log.Panicf("fail to open database - err(%s)\n", err.Error())
	} else {
		o.database = database
		if bucket, err := database.GetBucket(BucketName); err != nil {
			o.log.Panicf("fail to get bucket - err(%s)\n", err.Error())
		} else {
			o.accumulator = mta.NewExtAccumulator([]byte(AccStateKey), bucket, int64(o.start))
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

func (o *receiver) Start(status *btp.BMCLinkStatus) (<-chan link.ReceiveStatus, error) {

	finnum := status.Verifier.Height
	finseq := status.RxSeq
	o.log.Infof("current verifier status - height(%d) rx(%d)\n", finnum, finseq)

	och := make(chan link.ReceiveStatus)
	o.status = srcStatus{
		finnum:   uint64(finnum),
		curnum:   uint64(finnum),
		sequence: uint64(finseq),
	}
	o.finality.number = uint64(finnum)
	o.finality.sequence = uint64(finseq)

	// ensure initial mta & snapshot
	if err := o.prepare(); err != nil {
		return nil, err
	}

	// sync mta & snapshots based on peer status
	fn := big.NewInt(finnum)
	if err := o.synchronize(fn); err != nil {
		return nil, err
	}

	o.log.Infof("connect to bsc client...")
	if latest, err := o.client.BlockNumber(context.Background()); err != nil {
		return nil, err
	} else {
		o.log.Infof("latest block number of bsc client - number(%d)", latest)
	}

	o.log.Infof("fetch missing messages...")
	e := uint64(finnum)
	if msgs, err := o.client.MessagesAfterSequence(&bind.FilterOpts{
		Start:   o.start,
		End:     &e,
		Context: context.Background(),
	}, uint64(finseq)); err != nil {
		return nil, err
	} else {
		for _, m := range msgs {
			o.log.Infof("message - sequence(%d) number(%d) hash(%s)",
				m.Seq.Uint64(), m.Raw.BlockNumber, m.Raw.BlockHash.Hex())
			o.msgs.Add(m.Seq.Uint64(), m)
		}

		if len(msgs) > 0 {
			number, sequence := big.NewInt(finnum), msgs[len(msgs)-1].Seq
			go o.updateAndSendReceiverStatus(number, sequence, nil, och)
		}
	}

	blks := ToVerifierStatus(status.Verifier.Extra).BlockTree
	snap, err := o.snapshots.get(blks.Root())
	if err != nil {
		return nil, err
	}

	go o.loop(finnum, finseq, snap, och)
	return och, nil
}

func (o *receiver) loop(height, sequence int64, snap *Snapshot, och chan<- link.ReceiveStatus) {
	// watch new head & messages
	headCh := make(chan *types.Header)
	calc := newBlockFinalityCalculator(o.epoch, snap, o.snapshots, o.log)
	o.log.Tracef("new calculator - addr(%p) number(%d) hash(%s)", calc, snap.Number, snap.Hash.Hex())
	// rewatch with other height
	o.client.WatchHeader(context.Background(), big.NewInt(height+1), headCh)
	nth := uint64(0)
	for {
		select {
		case head := <-headCh:
			o.log.Tracef("receive new head - number(%d)", head.Number.Uint64())
			o.waitIfTooAhead(head.Number.Uint64())
			var err error
			// records block snapshot
			if snap, err = snap.apply(head, o.chainId); err != nil {
				o.log.Panicln(err.Error())
			}
			if err = o.snapshots.add(snap); err != nil {
				o.log.Panicln(err.Error())
			}

			fnzs, err := calc.feed(snap.Hash)
			if err != nil {
				o.log.Panicln(err.Error())
			}

			var finnum, sequence *big.Int
			if len(fnzs) > 0 {
				for _, fnz := range fnzs {
					head, err := o.client.HeaderByHash(context.Background(), fnz)
					if err != nil {
						o.log.Panicln(err)
					}
					finnum = head.Number
					o.heads.Add(head.Number.Uint64(), head)

					msgs, err := o.client.MessagesByBlockHash(context.Background(), fnz)
					if err != nil {
						// TODO
						o.log.Panicln(err)
					}

					if len(msgs) > 0 {
						sequence = msgs[len(msgs)-1].Seq
						for _, msg := range msgs {
							o.log.Debugf("new message - block number(%d) hash(%s), message sequence(%d)", msg.Raw.BlockNumber, msg.Raw.BlockHash, msg.Seq.Uint64())
							o.msgs.Add(sequence.Uint64(), msg)
						}
					}
				}
			}

			if sequence != nil || nth%100 == 0 {
				go o.updateAndSendReceiverStatus(finnum, sequence, new(big.Int).SetUint64(snap.Number), och)
			}
			nth++
		}
	}
}

func (o *receiver) updateAndSendReceiverStatus(finnum, sequence, curnum *big.Int, ch chan<- link.ReceiveStatus) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if curnum != nil && o.status.curnum != curnum.Uint64() {
		o.status.curnum = curnum.Uint64()
	}

	if finnum != nil && o.status.finnum != finnum.Uint64() ||
		sequence != nil && o.status.sequence != sequence.Uint64() {
		if finnum != nil {
			o.status.finnum = finnum.Uint64()
		}
		if sequence != nil {
			o.status.sequence = sequence.Uint64()
		}
		o.log.Tracef("send receiver status - number(%d) sequence(%d) monitor(%d)",
			o.status.finnum, o.status.sequence, o.status.curnum)
		ch <- o.status
	}
}

func (o *receiver) Stop() {
	// TODO dispose resources
	o.cancel()
}

func (o *receiver) GetStatus() (link.ReceiveStatus, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.status, nil
}

func (o *receiver) GetHeightForSeq(sequence int64) int64 {
	o.log.Tracef("++Recv::GetHeightForSequence(%d)", sequence)
	defer o.log.Tracef("--Recv::GetHeightForSequence(%d)", sequence)
	if msg, ok := o.msgs.Get(uint64(sequence)); ok {
		m := msg.(*BTPMessageCenterMessage)
		o.log.Tracef("from cache - number=(%d)", m.Raw.BlockNumber)
		return int64(m.Raw.BlockNumber)
	}

	e := o.status.curnum
	if msg, err := o.client.MessageBySequence(&bind.FilterOpts{
		Start:   o.start,
		End:     &e,
		Context: context.Background(),
	}, uint64(sequence)); err != nil {
		o.log.Errorf("fail to fetch message - err(%s) start(%d) sequence(%d)", err.Error(), o.start, sequence)
		return 0
	} else {
		if msg == nil {
			return 0
		} else {
			// TODO cache message
			o.log.Tracef("from network - number=(%d)", msg.Raw.BlockNumber)
			return int64(msg.Raw.BlockNumber)
		}
	}
}

func (o *receiver) findCanonicalBlock(rootNumber uint64, blks *BlockTree) *Snapshot {
	number := rootNumber + 1
	leaf, err := o.snapshots.get(blks.Root())
	if err != nil {
		o.log.Panicf("fail to fetch root snapshot - number(%d) hash(%s) err(%s)",
			rootNumber, blks.Root().Hex(), err.Error())
	}

	for {
		children := blks.ChildrenOf(leaf.Hash)
		if len(children) <= 0 {
			return leaf
		}

		head, err := o.headerByNumber(context.Background(), number)
		if err != nil {
			o.log.Errorf("no such header - number(%d) err(%s)", number, err.Error())
			return leaf
		}

		hash := head.Hash()
		if ok := slices.ContainsFunc(children, func(child common.Hash) bool {
			return bytes.Equal(child.Bytes(), hash.Bytes())
		}); !ok {
			return leaf
		} else {
			var err error
			if leaf, err = o.snapshots.get(hash); err != nil {
				o.log.Panicf("fail to load snapshot - number(%d) hash(%s) err(%s)",
					number, hash.Hex(), err.Error())
			}
			number++
			continue
		}
	}
}

func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	o.log.Tracef("++Recv:BuildBlockUpdate(%d, %d, %d)", status.Verifier.Height, status.RxSeq, limit)
	defer o.log.Tracef("--Recv:BuildBlockUpdate(%d, %d, %d)", status.Verifier.Height, status.RxSeq, limit)
	finnum := uint64(status.Verifier.Height)
	blks := ToVerifierStatus(status.Verifier.Extra).BlockTree
	// accumulate missing finalized block hash
	root, err := o.snapshots.get(blks.Root())
	if err != nil {
		o.log.Errorf("fail to retrieving snapshot of finalized block - number(%d) hash(%s)\n", finnum, blks.Root().Hex())
		return nil, err
	} else {
		if root.Number > uint64(o.accumulator.Height()+1) {
			o.log.Warnf("sync accmulator height current(%d) -> target(%d)\n", o.accumulator.Height()+1, root.Number)

		}
	}

	// find block to start updating
	leaf := o.findCanonicalBlock(finnum, blks)
	o.log.Debugf("leaf number(%d) hash(%s)", leaf.Number, leaf.Hash.Hex())

	done := false
	calc := newBlockFinalityCalculator(o.epoch, root, o.snapshots, o.log)
	o.log.Tracef("new calculator - addr(%p) number(%d) hash(%s)", calc, root.Number, root.Hash.Hex())
	calc.ensureFeeds(leaf)

	// collect headers to include in block update
	heads := make([]*types.Header, 0)
	for leaf.Number < o.status.curnum && !done {
		head, err := o.headerByNumber(context.Background(), leaf.Number+1)
		if err != nil {
			o.log.Errorf("fail to fetching header - number(%d) err(%s)", leaf.Number, err.Error())
			return nil, err
		}

		snap, err := o.snapshots.get(head.Hash())
		if err != nil {
			o.log.Panicf("NoSnapshot - number(%d) hash(%d) err(%s)",
				head.Number.Uint64(), head.Hash().Hex(), err.Error())
		}
		if !bytes.Equal(snap.ParentHash.Bytes(), leaf.Hash.Bytes()) {
			o.log.Panicf("InconsistentChain - expected(%s) actual(%s) number(%d)",
				snap.ParentHash.Hex(), leaf.Hash.Hex(), leaf.Number)
		}

		size := int64(math.Ceil(float64(head.Size())))
		if limit < size*5 {
			break
		} else {
			limit -= size * 5
		}

		heads = append(heads, head)

		// update extra status of verifier
		if err := blks.Add(snap.ParentHash, snap.Hash); err != nil {
			o.log.Panicf("fail to update block tree - parent(%s) hash(%s) err(%s)",
				snap.ParentHash.Hex(), snap.Hash.Hex(), err.Error())
		}

		// calculate newly finalized blocks with a supplied block
		if fnzs, err := calc.feed(snap.Hash); err != nil {
			return nil, err
		} else {
			// update extra status of verifier
			if len(fnzs) > 0 {
				finnum += uint64(len(fnzs))
				blks.Prune(fnzs[len(fnzs)-1])
			}

			for _, fnz := range fnzs {
				if msgs, err := o.client.MessagesByBlockHash(context.Background(), fnz); err != nil {
					for _, msg := range msgs {
						o.log.Debugf("message exist in finalized block - number(%s) hash(%s) sequence(%d)",
							msg.Raw.BlockNumber, msg.Raw.BlockHash, msg.Seq.Uint64())
						o.msgs.Add(msg.Seq.Uint64(), msg)
					}
					done = true
				}
			}
		}
		leaf = snap
	}

	if len(heads) > 0 {
		o.log.Debugf("BU{ H(%d) ~ H(%d)}\n", heads[0].Number.Uint64(), heads[len(heads)-1].Number.Uint64())
		return []link.BlockUpdate{
			&BlockUpdate{
				heads:  heads,
				start:  uint64(status.Verifier.Height + 1),
				height: finnum,
				status: &VerifierStatus{
					Offset:    uint64(o.accumulator.Offset()),
					BlockTree: blks,
				},
			},
		}, nil
	} else {
		return []link.BlockUpdate{}, nil
	}
}

func (o *receiver) headerByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	if head, ok := o.heads.Get(number); ok {
		o.log.Tracef("hit head cache - number(%d)", number)
		return head.(*types.Header), nil
	}
	o.log.Debugf("miss head cache - number(%d)", number)
	return o.client.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
}

func (o *receiver) BuildBlockProof(status *btp.BMCLinkStatus, target int64) (link.BlockProof, error) {
	o.log.Tracef("++Recv::BuildBlockProof - number(%d) sequence(%d) target(%d)", status.Verifier.Height, status.RxSeq, target)
	defer o.log.Tracef("--Recv::BuildBlockProof")
	// Accumulate missing block hashes
	finnum := status.Verifier.Height
	o.log.Debugf("current accumulator height(%d), current finalized height(%d)", o.accumulator.Height(), finnum)

	if err := o.syncAcc(ToVerifierStatus(status.Verifier.Extra).BlockTree.Root()); err != nil {
		return nil, err
	}

	_, witness, err := o.accumulator.WitnessForAt(target+1, finnum+1, o.accumulator.Offset())
	if err != nil {
		o.log.Errorf("fail to retrieve witness - error(%s)\n", err.Error())
		return nil, err
	}

	hs := mta.WitnessesToHashes(witness)
	for i, b := range hs {
		o.log.Debugf("W={%d : %s}", i, hex.EncodeToString(b))
	}

	if head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(target)); err != nil {
		o.log.Errorf("fail to fetching header - number(%d) error(%s)\n", target, err.Error())
		return nil, err
	} else {
		o.log.Debugf("BP{ H(%d:%s) N(%d) }", head.Number.Uint64(), head.Hash(), finnum+1)
		return &BlockProof{
			Header:    head,
			AccHeight: uint64(finnum + 1),
			Witness:   mta.WitnessesToHashes(witness),
		}, nil
	}
}

func (o *receiver) BuildMessageProof(status *btp.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	o.log.Tracef("++Recv::BuildMessageProof(%d, %d, %d)", status.Verifier.Height, status.RxSeq, limit)
	defer o.log.Tracef("--Recv::BuildMessageProof")
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
		o.log.Debugf("MP{ hash(%s) nproofs(%d) start sequence(%d)",
			msgproof.Hash.Hex(), len(msgproof.Proofs), msgproof.sequence)
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

func (o *receiver) FinalizedStatus(bls <-chan *btp.BMCLinkStatus) {
	go func() {
		for {
			select {
			case status := <-bls:
				o.log.Debugf("on status finalized - number(%d) sequence(%d)",
					status.Verifier.Height, status.RxSeq)
				root := ToVerifierStatus(status.Verifier.Extra).BlockTree.Root()
				if err := o.syncAcc(root); err != nil {
					o.log.Errorf("fail to sync accumulator - number(%d) hash(%s) accnum(%d)",
						status.Verifier.Height, root, o.accumulator.Height())
				}
				o.purgeAndWakeupIfNear(status)
			}
		}
	}()
}

func (o *receiver) syncAcc(until common.Hash) error {
	snap, err := o.snapshots.get(until)
	if err != nil {
		return err
	}

	o.log.Debugf("synchronize accumulator... - curr(%d) targ(%d)",
		o.accumulator.Height(), snap.Number+1)

	hashes := make([]common.Hash, 0)
	for o.accumulator.Height() <= int64(snap.Number) {
		hashes = append(hashes, snap.Hash)
		snap, err = o.snapshots.get(snap.ParentHash)
		if err != nil {
			return err
		}
	}
	for i := range hashes {
		o.accumulator.AddHash(hashes[len(hashes)-1-i].Bytes())
	}
	if err := o.accumulator.Flush(); err != nil {
		return err
	}
	return nil
}

func (o *receiver) waitIfTooAhead(cursor uint64) {
	o.cond.L.Lock()
	defer o.cond.L.Unlock()
	if o.finality.number+CacheSize < cursor {
		o.log.Debugf("pause block fetching loop - ")
		o.cond.Wait()
	}
}

func (o *receiver) purgeAndWakeupIfNear(newFinality *btp.BMCLinkStatus) {
	o.log.Traceln("++Recv::purgeAndWakeupIfNear")
	defer o.log.Traceln("--Recv::purgeAndWakeupIfNear")
	o.cond.L.Lock()
	defer o.cond.L.Unlock()

	for i := o.finality.number; i <= uint64(newFinality.Verifier.Height); i++ {
		o.log.Tracef("purge head cache - (%d)", i)
		o.heads.Remove(i)
	}

	for i := o.finality.sequence; i <= uint64(newFinality.RxSeq); i++ {
		o.log.Tracef("purge msg cache - (%d)", i)
		o.msgs.Remove(i)
	}

	if o.finality.number*2/3 < uint64(newFinality.Verifier.Height) {
		defer func() {
			o.log.Debugf("resume block fetching loop")
			o.cond.Broadcast()
		}()
	}

	o.finality.number = uint64(newFinality.Verifier.Height)
	o.finality.sequence = uint64(newFinality.RxSeq)
}

// ensure initial merkle accumulator and snapshot
func (o *receiver) prepare() error {
	o.log.Debugf("prepare receiver")
	number := big.NewInt(o.accumulator.Offset())
	if number.Uint64()%o.epoch != 0 {
		o.log.Panicf("Must be epoch block: epoch: %d number: %d", number.Uint64(), o.epoch)
	}
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)", number.Uint64(), err.Error())
		return err
	}

	// No initial trusted block hash
	if o.accumulator.Len() <= 0 {
		o.log.Infof("accumulate initial block hash: hash(%s)", head.Hash().Hex())
		o.accumulator.AddHash(head.Hash().Bytes())
		if err := o.accumulator.Flush(); err != nil {
			o.log.Errorln("fail to flush accumulator: err(%s)", err.Error())
			return err
		}
	}

	ok, err := hasSnapshot(o.database, head.Hash())
	if err != nil {
		o.log.Errorln("fail to check snapshot existent: err(%s)", err.Error())
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

		o.log.Infof("create initial snapshot: %s\n", snap.String())
	}
	return nil
}

func (o *receiver) synchronize(until *big.Int) error {
	o.log.Debugln("synchronize receiver")
	// synchronize up to `target` block
	target, err := o.client.HeaderByNumber(context.Background(), until)
	if err != nil {
		return err
	}
	hash := target.Hash()

	// synchronize snapshots
	o.log.Debugf("synchronize block snapshots...")
	if err := o.snapshots.ensure(hash); err != nil {
		return err
	}

	// synchronize accumulator
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

func newMTPWithReceipts(receipts types.Receipts) (*trie.Trie, error) {
	if trie, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New())); err != nil {
		return nil, err
	} else {
		for _, receipt := range receipts {
			key, err := rlp.EncodeToBytes(receipt.TransactionIndex)
			if err != nil {
				return nil, err
			}

			buf := new(bytes.Buffer)
			buf.Reset()
			if err := receipt.EncodeRLP(buf); err != nil {
				return nil, err
			}
			trie.Update(key, buf.Bytes())
		}
		return trie, nil
	}
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

// implements link.ReceiveStatus
type srcStatus struct {
	finnum   uint64
	curnum   uint64
	sequence uint64
}

// TODO reference
func (s srcStatus) Height() int64 {
	return int64(s.finnum)
}

func (s srcStatus) Seq() int64 {
	return int64(s.sequence)
}

func ToVerifierStatus(blob []byte) *VerifierStatus {
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(blob, vs); err != nil {
		panic(fmt.Sprintf("fail to decode verifier status - blob(%s) err(%s)\n",
			hex.EncodeToString(blob), err.Error()))
	}
	return vs
}
