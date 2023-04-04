package bsc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	ethereum "github.com/ethereum/go-ethereum"
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

type Config struct {
	ChainID     int    `json:"chain_id"`
	Epoch       int    `json:"epoch"`
	StartNumber int    `json:"start_height"`
	Endpoint    string `json:"endpoint"`
	DBType      string `json:"db_type"`
	DBPath      string `json:"db_path"`
	// TODO support multi-depth env options
	//DB          DBConfig `json:"db"`
	SrcAddress btp.BtpAddress
	DstAddress btp.BtpAddress
}

type DBConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type receiver struct {
	start       uint64   // TODO extract to config
	chainId     *big.Int // TODO extract to config
	epoch       uint64   // TODO extract to config
	client      *client
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

func NewReceiver(config Config, log log.Logger) *receiver {
	o := &receiver{
		chainId: big.NewInt(int64(config.ChainID)),
		epoch:   uint64(config.Epoch),
		start:   uint64(config.StartNumber),
		client:  NewClient(config.Endpoint, config.SrcAddress, config.DstAddress, log),
		status:  srcStatus{},
		cond:    sync.NewCond(&sync.Mutex{}),
		log:     log,
	}

	if cache, err := lru.NewARC(CacheSize); err != nil {
		panic(err)
	} else {
		o.heads = cache
	}

	if cache, err := lru.NewARC(CacheSize); err != nil {
		panic(err)
	} else {
		o.msgs = cache
	}

	o.log.Infof("Create BSC Receiver - chainid: %d, epoch: %d, startnumber: %d\n",
		o.chainId.Uint64(), o.epoch, o.start)

	o.log.Infoln("Open database - path(%s) type(%s)\n", config.DBPath, string(db.GoLevelDBBackend))
	if database, err := db.Open(config.DBPath, string(db.GoLevelDBBackend), DBName); err != nil {
		panic(fmt.Sprintf("Fail to open database - %s\n", err.Error()))
	} else {
		o.database = database
		if bucket, err := database.GetBucket(BucketName); err != nil {
			panic(err)
		} else {
			o.accumulator = mta.NewExtAccumulator([]byte(AccStateKey), bucket, int64(o.start))
			if bucket.Has([]byte(AccStateKey)) {
				if err = o.accumulator.Recover(); err != nil {
					panic(fmt.Sprintf("Fail to recover accumulator - err: %s\n", err.Error()))
				}
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
	o.finality.number = o.status.finnum
	o.finality.sequence = o.status.sequence

	// ensure initial mta & snapshot
	if err := o.prepare(); err != nil {
		return nil, err
	}

	// sync mta & snapshots based on peer status
	fn := big.NewInt(finnum)
	if err := o.synchronize(fn); err != nil {
		return nil, err
	}

	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		return nil, err
	}

	snap, err := o.snapshots.get(vs.BlockTree.Root())
	if err != nil {
		return nil, err
	}

	go o.loop(finnum, finseq, snap, och)
	return och, nil
}

func (o *receiver) loop(height, sequence int64, snap *Snapshot, och chan<- link.ReceiveStatus) {
	// collect missing messages
	if msgs, err := o.client.FetchMissingMessages(context.Background(), o.start, uint64(height), uint64(sequence)); err != nil {
		panic(err)
	} else {
		for _, msg := range msgs {
			o.msgs.Add(msg.Seq.Uint64(), msg)
		}
		if len(msgs) > 0 {
			o.log.Debugf("Send Last Missing Message Sequence(%d)\n", msgs[len(msgs)-1].Seq.Uint64())
			o.updateAndSendReceiverStatus(big.NewInt(height), msgs[len(msgs)-1].Seq, nil, och)
		}
	}

	// watch new head & messages
	headCh := make(chan *types.Header)
	calc := newBlockFinalityCalculator(o.epoch, snap, o.snapshots, o.log)
	// rewatch with other height
	o.client.WatchHeader(context.Background(), big.NewInt(height+1), headCh)
	for {
		select {
		case head := <-headCh:
			o.waitIfTooAhead(head.Number.Uint64())
			var err error
			// records block snapshot
			if snap, err = snap.apply(head, o.chainId); err != nil {
				panic(err.Error())
			}
			if err = o.snapshots.add(snap); err != nil {
				panic(err.Error())
			}

			fnzs, err := calc.feed(snap.Hash)
			if err != nil {
				panic(err.Error())
			}

			var finnum, sequence *big.Int
			if len(fnzs) > 0 {
				for _, fnz := range fnzs {
					head, err := o.client.HeaderByHash(context.Background(), fnz)
					if err != nil {
						panic(err)
					}
					finnum = head.Number
					o.heads.Add(head.Number.Uint64(), head)

					msgs, err := o.client.MessagesByBlockHash(context.Background(), fnz)
					if err != nil {
						// TODO
						panic(err)
					}

					if len(msgs) > 0 {
						sequence = msgs[len(msgs)-1].Seq
						for _, msg := range msgs {
							o.log.Debugf("Cache Message: Sequence(%d) Message(%s)\n", sequence.Uint64(), hex.EncodeToString(msg.Msg))
							o.msgs.Add(sequence.Uint64(), msg)
						}
						o.log.Debug("Update Sequence(%d)\n", sequence.Uint64())
					}
				}
			}
			go o.updateAndSendReceiverStatus(finnum, sequence, new(big.Int).SetUint64(snap.Number), och)
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
	// TODO handle nil status

	return o.status, nil
}

func (o *receiver) GetMarginForLimit() int64 {
	return 0
}

func (o *receiver) GetHeightForSeq(sequence int64) int64 {
	o.log.Debugf("++GetHeightForSequence - sequence(%d)\n", sequence)
	defer o.log.Debugf("--GetHeightForSequence")
	if msg, ok := o.msgs.Get(uint64(sequence)); ok {
		m := msg.(*BTPMessageCenterMessage)
		o.log.Debugf("Hit cache number(%d)\n", m.Raw.BlockNumber)
		return int64(m.Raw.BlockNumber)
	}
	// TODO using cache
	if msg, err := o.client.FindMessage(context.Background(), o.start, nil, big.NewInt(sequence)); err != nil {
		return 0
	} else {
		return int64(msg.Raw.BlockNumber)
	}
}

func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	o.log.Debugln("++BuildBlockUpdate")
	defer o.log.Debugln("--BuildBlockUpdate")
	finnum := uint64(status.Verifier.Height)
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		o.log.Errorf("fail to decoding verifier extra status - blob(%s)\n",
			hex.EncodeToString(status.Verifier.Extra))
		return nil, err
	}

	curnum := status.Verifier.Height
	blks := vs.BlockTree

	// accumulate missing finalized block hash
	if snap, err := o.snapshots.get(blks.Root()); err != nil {
		o.log.Errorf("fail to retrieving snapshot of finalized block - number(%d) hash(%s)\n", curnum, blks.Root().Hex())
		return nil, err
	} else {
		if snap.Number > uint64(o.accumulator.Height()+1) {
			o.log.Warnf("sync accmulator height current(%d) -> target(%d)\n", o.accumulator.Height()+1, snap.Number)

		}
	}

	// find block to start updating
	var leaf *Snapshot
	var err error
	for i := 0; i < len(blks.Nodes()) && leaf == nil; i++ {
		leaf, err = o.snapshots.get(blks.Nodes()[i])
		if err != nil {
			if err == ethereum.NotFound {
				continue
			}
			return nil, err
		}
	}

	if leaf == nil {
		panic("InconsistentChain")
	}

	done := false
	calc := newBlockFinalityCalculator(o.epoch, leaf, o.snapshots, o.log)
	// collect headers to include in block update
	heads := make([]*types.Header, 0)
	o.log.Debugf("Leaf Snap Number(%d) Hash(%s)\n", leaf.Number, leaf.Hash.Hex())
	for leaf.Number < o.status.curnum && !done {
		head, err := o.headerByNumber(context.Background(), leaf.Number+1)
		if err != nil {
			o.log.Errorf("fail to fetching header - number(%d) err(%s)\n", leaf.Number, err.Error())
			return nil, err
		}

		snap, err := o.snapshots.get(head.Hash())
		if err != nil {
			panic(fmt.Sprintf("NoSnapshot - number(%d) hash(%d) err(%s)\n",
				head.Number.Uint64(), head.Hash().Hex(), err.Error()))
		}
		if !bytes.Equal(snap.ParentHash.Bytes(), leaf.Hash.Bytes()) {
			panic(fmt.Sprintf("InconsistentChain - expected(%s) actual(%s) number(%d)\n",
				snap.ParentHash.Hex(), leaf.Hash.Hex(), leaf.Number))
		}

		size := int64(math.Ceil(float64(head.Size())))
		if limit < size {
			break
		} else {
			limit -= size
		}

		heads = append(heads, head)

		// update extra status of verifier
		if err := blks.Add(snap.ParentHash, snap.Hash); err != nil {
			panic(fmt.Sprintf("fail to update block tree - parent(%s) hash(%s) err(%s)\n",
				snap.ParentHash.Hex(), snap.Hash.Hex(), err.Error()))
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
				// update accumulator with hashes of finalized blocks
				o.accumulator.AddHash(fnz.Bytes())
				if err := o.accumulator.Flush(); err != nil {
					return nil, err
				}
				if msgs, err := o.client.MessagesByBlockHash(context.Background(), fnz); err != nil {
					for _, msg := range msgs {
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
			BlockUpdate{
				heads:  heads,
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
		return head.(*types.Header), nil
	}
	return o.client.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
}

func (o *receiver) BuildBlockProof(status *btp.BMCLinkStatus, target int64) (link.BlockProof, error) {
	o.log.Debugf("++BuildBlockProof")
	defer o.log.Debugf("--BuildBlockProof")
	// Accumulate missing block hashes
	accnum := o.accumulator.Height()
	finnum := status.Verifier.Height
	for accnum <= finnum {
		accnum++
		if head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(accnum)); err != nil {
			panic(err.Error())
		} else {
			o.accumulator.AddHash(head.Hash().Bytes())
			if err := o.accumulator.Flush(); err != nil {
				return nil, err
			}
		}
	}

	_, witness, err := o.accumulator.WitnessForAt(target+1, finnum+1, o.accumulator.Offset())
	if err != nil {
		o.log.Errorf("fail to retrieve witness - error(%s)\n", err.Error())
		return nil, err
	}

	if head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(target)); err != nil {
		o.log.Errorf("fail to fetching header - number(%d) error(%s)\n", target, err.Error())
		return nil, err
	} else {
		return BSCBlockProof{
			Header: head,
			//AccHeight: uint64(accnum),
			AccHeight: uint64(finnum + 1),
			Witness:   mta.WitnessesToHashes(witness),
		}, nil
	}
}

func (o *receiver) BuildMessageProof(status *btp.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	sequence := uint64(status.RxSeq)
	var msgs []*BTPMessageCenterMessage
	for {
		sequence++
		raw, ok := o.msgs.Get(sequence)
		if !ok {
			break
		}
		msg := raw.(*BTPMessageCenterMessage)
		if uint64(status.Verifier.Height) < msg.Raw.BlockNumber {
			break
		}

		o.log.Debugf("MSG: (%+v)\n", msg)
		if msgs == nil {
			msgs = append(make([]*BTPMessageCenterMessage, 0), msg)
		} else if msgs[0].Raw.BlockNumber == msg.Raw.BlockNumber {
			msgs = append(msgs, msg)
		} else {
			break
		}
	}

	if len(msgs) <= 0 {
		return nil, nil
	}

	hash := msgs[0].Raw.BlockHash
	receipts, err := o.client.ReceiptsByBlockHash(context.Background(), hash)
	if err != nil {
		return nil, err
	}

	mpt, err := newMTPWithReceipts(receipts)
	if err != nil {
		return nil, err
	}

	var msgproof *BSCMessageProof
	for _, msg := range msgs {
		key, err := rlp.EncodeToBytes(msg.Raw.TxIndex)
		if err != nil {
			return nil, err
		}

		proof, err := newProofOf(mpt, key)
		if err != nil {
			return nil, err
		}

		if msgproof == nil {
			msgproof = &BSCMessageProof{
				Hash:     hash,
				Proofs:   make([]BSCReceiptProof, 0),
				sequence: msg.Seq.Uint64(),
			}
		}
		// TODO FIXME not exact size...
		size := int64(0)
		for i := 0; i < len(proof); i++ {
			size += int64(len(proof[i]))
		}
		if limit < size {
			break
		}
		limit -= size
		msgproof.Proofs = append(msgproof.Proofs, BSCReceiptProof{
			Key:   key,
			Proof: proof,
		})
	}

	return msgproof, nil
}

func (o *receiver) BuildRelayMessage(parts []link.RelayMessageItem) ([]byte, error) {
	o.log.Debugf("++receiver::BuildRelayMessage - size(%d)\n", len(parts))
	defer o.log.Debugln("--receiver::BuildRelayMessage")
	msg := &BSCRelayMessage{
		TypePrefixedMessages: make([]BSCTypePrefixedMessage, 0),
	}

	for _, part := range parts {
		blob, err := rlp.EncodeToBytes(part)
		if err != nil {
			o.log.Errorf("fail to encode type prefixed message - (%d)\n", typeToUint(part.Type()))
			return nil, err
		}

		msg.TypePrefixedMessages = append(msg.TypePrefixedMessages, BSCTypePrefixedMessage{
			Type:    typeToUint(part.Type()),
			Payload: blob,
		})
	}

	if blob, err := rlp.EncodeToBytes(msg); err != nil {
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
				o.purgeAndWakeupIfNear(status)
			}
		}
	}()
}

func (o *receiver) waitIfTooAhead(cursor uint64) {
	o.cond.L.Lock()
	defer o.cond.L.Unlock()
	if o.finality.number+CacheSize < cursor {
		o.cond.Wait()
	}
}

func (o *receiver) purgeAndWakeupIfNear(newFinality *btp.BMCLinkStatus) {
	o.log.Debugln("++purgeAndWakeupIfNear")
	defer o.log.Debugln("--purgeAndWakeupIfNear")
	o.cond.L.Lock()
	defer o.cond.L.Unlock()

	for i := o.finality.number; i <= uint64(newFinality.Verifier.Height); i++ {
		o.heads.Remove(i)
	}

	for i := o.finality.sequence; i <= uint64(newFinality.RxSeq); i++ {
		o.msgs.Remove(i)
	}

	if o.finality.number*2/3 < uint64(newFinality.Verifier.Height) {
		defer o.cond.Broadcast()
	}

	o.finality.number = uint64(newFinality.Verifier.Height)
	o.finality.sequence = uint64(newFinality.RxSeq)

}

// ensure initial merkle accumulator and snapshot
func (o *receiver) prepare() error {
	o.log.Debugln("++receiver::prepare")
	defer o.log.Debugln("--receiver::prepare")

	number := big.NewInt(o.accumulator.Offset())
	if number.Uint64()%o.epoch != 0 {
		panic(fmt.Sprintf("Must be epoch block: epoch: %d number: %d\n", number.Uint64(), o.epoch))
	}
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)\n", number.Uint64(), err.Error())
		return err
	}

	// No initial trusted block hash
	if o.accumulator.Len() <= 0 {
		o.log.Infof("accumulate initial block hash: hash(%s)\n", head.Hash().Hex())
		o.accumulator.AddHash(head.Hash().Bytes())
		if err := o.accumulator.Flush(); err != nil {
			o.log.Errorln("fail to flush accumulator: err(%s)\n", err.Error())
			return err
		}
	}

	ok, err := hasSnapshot(o.database, head.Hash())
	if err != nil {
		o.log.Errorln("fail to check snapshot existent: err(%s)\n", err.Error())
		return err
	}

	if !ok {
		valBytes := head.Extra[extraVanity : len(head.Extra)-extraSeal]
		vals, err := ParseValidators(valBytes)
		if err != nil {
			o.log.Errorf("fail to parsing candidates set bytes - %s\n", hex.EncodeToString(valBytes))
			return err
		}

		var snap *Snapshot
		if head.Number.Uint64() == 0 {
			snap = newSnapshot(head.Number.Uint64(), head.Hash(), vals,
				vals, make([]common.Address, 0), head.Coinbase, head.ParentHash)
		} else {
			number := big.NewInt(head.Number.Int64() - int64(o.epoch))
			oldHead, err := o.client.HeaderByNumber(context.Background(), number)
			if err != nil {
				o.log.Errorf("fail to fetching header - number(%d) error(%s)\n", number.Uint64(), err.Error())
				return err
			}

			oldValBytes := oldHead.Extra[extraVanity : len(oldHead.Extra)-extraSeal]
			oldVals, err := ParseValidators(oldValBytes)
			if err != nil {
				o.log.Errorf("fail to parsing validators set bytes - %s\n", hex.EncodeToString(oldValBytes))
				return err
			}

			recents := make([]common.Address, 0)
			for i := head.Number.Int64() - int64(len(oldVals)/2); i <= head.Number.Int64(); i++ {
				if oldHead, err = o.client.HeaderByNumber(context.Background(), big.NewInt(i)); err != nil {
					o.log.Errorf("fail to fetching header - number(%d) error(%s)\n", i, err.Error())
					return err
				} else {
					recents = append(recents, oldHead.Coinbase)
				}
			}
			snap = newSnapshot(head.Number.Uint64(), head.Hash(), oldVals,
				vals, recents, head.Coinbase, head.ParentHash)
		}

		if err := snap.store(o.database); err != nil {
			return err
		}

		o.log.Infof("create initial snapshot: %s\n", snap.String())
	}
	return nil
}

func (o *receiver) synchronize(until *big.Int) error {
	o.log.Infof("++receiver::synchronize(%d)\n", until.Uint64())
	defer o.log.Infoln("--receiver::synchronize")
	// synchronize up to `target` block
	target, err := o.client.HeaderByNumber(context.Background(), until)
	if err != nil {
		return err
	}
	hash := target.Hash()

	// synchronize snapshots
	if err := o.snapshots.ensure(hash); err != nil {
		return err
	}

	snap, err := o.snapshots.get(hash)
	if err != nil {
		return err
	}

	// synchronize accumulator
	o.log.Infof("current accmulator height: %d\n", o.accumulator.Height())
	o.log.Infof("current finalized block height: %d\n", target.Number.Uint64())
	hashes := make([]common.Hash, 0)
	for o.accumulator.Height() <= int64(snap.Number) {
		o.log.Infof("accumulate missing block proof: number(%d) hash(%s)\n", snap.Number, snap.Hash.Hex())
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
