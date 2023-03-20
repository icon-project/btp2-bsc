package bsc

import (
	"bytes"
	"context"
	"fmt"
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
	"math"
	"math/big"
	"sync"
)

const (
	CacheSize          = 512
	CheckpointInterval = 256
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
	//snapshots   *lru.ARCCache
	heads  *WaitQueue
	msgs   *WaitQueue
	mutex  sync.Mutex
	status *srcStatus
	log    log.Logger
}

func NewReceiver(config Config, log log.Logger) *receiver {
	o := &receiver{
		chainId: big.NewInt(int64(config.ChainID)),
		epoch:   uint64(config.Epoch),
		start:   uint64(config.StartNumber),
		heads:   NewWaitQueue(256),
		msgs:    NewWaitQueue(256),
		client:  NewClient(config.Endpoint, config.SrcAddress, config.DstAddress, log),
		status:  &srcStatus{},
		log:     log,
	}

	// if snaps, err := lru.NewARC(CacheSize); err != nil {
	// 	panic(err)
	// } else {
	// 	o.snapshots = snaps
	// }

	if database, err := db.Open(config.DBPath, string(db.GoLevelDBBackend), "receiver"); err != nil {
		panic(err)
	} else {
		o.database = database
		if bucket, err := database.GetBucket("bsc-accumulator"); err != nil {
			panic(err)
		} else {
			o.accumulator = mta.NewExtAccumulator([]byte("accumulator"), bucket, int64(o.start))
			if bucket.Has([]byte("accumulator")) {
				if err = o.accumulator.Recover(); err != nil {
					panic(err)
				}
			}
			fmt.Println("Initial Acc Height:", o.accumulator.Height())
		}
	}
	o.snapshots = newSnapshots(o.chainId, o.client.Client, CacheSize, o.database)
	return o
}

type BSCFeed struct {
	hash     common.Hash
	number   uint64
	sequence *big.Int
}

func (o *BSCFeed) ID() common.Hash {
	return o.hash
}

func (o *BSCFeed) Number() uint64 {
	return o.number
}

func (o *BSCFeed) Sequence() *big.Int {
	return o.sequence
}

func (o *receiver) Start(status *btp.BMCLinkStatus) (<-chan link.ReceiveStatus, error) {
	och := make(chan link.ReceiveStatus)
	o.status = &srcStatus{
		height:   status.Verifier.Height,
		sequence: status.RxSeq,
	}
	go func() {
		// ensure initial mta & snapshot
		if err := o.prepare(); err != nil {
			panic(err)
		}

		// sync mta & snapshots based on peer status
		fn := big.NewInt(status.Verifier.Height)
		if err := o.synchronize(fn); err != nil {
			panic(err)
		}

		// TODO implements
		// TODO too many missing messages (case exceeding buffer capability)
		// collect missing messages

		// watch new head & messages
		vs := &VerifierStatus{}
		if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
			panic(err)
		}
		snap, err := o.snapshots.get(vs.BlockTree.Root())
		if err != nil {
			panic(err)
		}

		fch := make(chan *BTPData)
		finalizer := newBlockFinalizer(big.NewInt(int64(o.epoch)), snap, o.snapshots)
		o.client.WatchBTPData(context.Background(), fn.Add(fn, big.NewInt(1)), fch)
		for {
			select {
			case data := <-fch:
				var err error
				// records block snapshot
				if snap, err = snap.apply(data.header, o.chainId); err != nil {
					panic(err)
				}
				if err = o.snapshots.add(snap); err != nil {
					panic(err)
				}

				feed := &BSCFeed{
					hash:     snap.Hash,
					number:   snap.Number,
					sequence: nil,
				}
				if len(data.messages) > 0 {
					feed.sequence = data.messages[len(data.messages)-1].Sequence
				}
				if fins, err := finalizer.feed(feed); err != nil {
					panic(err)
				} else if len(fins) > 0 {
					o.mutex.Lock()
					o.status.height = int64(fins[len(fins)-1].(*BSCFeed).number)
					for _, c := range fins {
						f := c.(*BSCFeed)
						if f.sequence != nil {
							o.status.sequence = f.sequence.Int64()
							break
						}
					}
					och <- o.status
					o.mutex.Unlock()
				}
				o.heads.Enqueue(data.header)
				for _, m := range data.messages {
					o.msgs.Enqueue(m)
				}
			}
		}
	}()
	return och, nil
}

func (o *receiver) Stop() {
	// TODO dispose resources
	o.cancel()
}

func (o *receiver) GetStatus() (link.ReceiveStatus, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.status == nil {
		panic("NotReady")
	}
	return o.status, nil
}

func (o *receiver) GetMarginForLimit() int64 {
	return 0
}

func (o *receiver) GetHeightForSeq(sequence int64) int64 {
	msg, err := o.client.FindMessage(context.Background(), o.start, nil, big.NewInt(sequence))
	if err != nil {
		return 0
	}
	return int64(msg.Raw.BlockNumber)
}

func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	fmt.Println("BuildBlockUpdate Height=", status.Verifier.Height, " RxSeq=", status.RxSeq)
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		return nil, err
	}
	number := status.Verifier.Height
	tree := vs.BlockTree
	heads := make([]*types.Header, 0)
	var fins []common.Hash

	for o.heads.Len() > 0 {
		//for {
		//	if o.heads.Len() <= 0 {
		//		time.Sleep(time.Second)
		//		continue
		//	}
		head, _ := o.heads.First().(*types.Header)

		// drop already finalized blocks
		if head.Number.Int64() <= number {
			o.accumulator.AddHash(head.Hash().Bytes())
			if err := o.accumulator.Flush(); err != nil {
				return nil, err
			}
			o.heads.Dequeue()
			continue
		}

		// drop blocks already known to verifier
		if tree.Has(head.Hash()) {
			o.heads.Dequeue()
			continue
		}

		// check block consistency
		if !tree.Has(head.ParentHash) {
			panic(fmt.Sprintf("inconsistent block tree: %s", head.ParentHash))
		}

		// check doesn't exceed size of allowed block update
		size := int64(math.Ceil(float64(head.Size())))
		if limit < size {
			break
		} else {
			limit -= size
		}

		// consume collected head
		o.heads.Dequeue()
		heads = append(heads, head)

		// update estimated status
		if err := tree.Add(head.ParentHash, head.Hash()); err != nil {
			panic(fmt.Sprintf("Fail to append block hash: parent: %v, hash: %v cause: %s\n",
				head.ParentHash, head.Hash(), err.Error()))
		}

		var err error
		// TODO improve
		if fins, err = finalize(o.epoch, o.snapshots, head.Hash(), tree.Root()); err != nil {
			return nil, err
		} else {
			if len(fins) > 0 && o.msgs.Len() > 0 {
				msg, _ := o.msgs.First().(*BTPMessageCenterMessage)
				// TODO improve duplicated bytes comparison
				for _, fin := range fins {
					if bytes.Equal(msg.Raw.BlockHash.Bytes(), fin.Bytes()) {
						break
					}
				}
			}
		}
	}

	if len(fins) > 0 {
		tree.Prune(fins[0])
		for _, cnf := range fins {
			o.accumulator.AddHash(cnf.Bytes())
			if err := o.accumulator.Flush(); err != nil {
				return nil, err
			}
		}
	}

	if len(heads) > 0 {
		return []link.BlockUpdate{
			BlockUpdate{
				heads:  heads,
				height: uint64(status.Verifier.Height + int64(len(fins))),
				status: &VerifierStatus{
					Offset:    uint64(o.accumulator.Offset()),
					BlockTree: tree,
				},
			},
		}, nil
	} else {
		return []link.BlockUpdate{}, nil
	}

}

// func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
// 	fmt.Println("BuildBlockUpdate")
// 	vs := &VerifierStatus{}
// 	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
// 		return nil, err
// 	}
//
// 	number := status.Verifier.Height
// 	tree := vs.BlockTree
// 	heads := make([]*types.Header, 0)
// 	var confirmations []common.Hash
// 	// exit loop
// 	// 1) `block updates` size exceeds limited size
// 	// 2) `btp message` exists
// 	for o.heads.Len() > 0 {
// 		head, _ := o.heads.First().(*types.Header)
//
// 		// block already finalized by verifier
// 		if head.Number.Int64() <= number {
// 			o.accumulator.AddHash(head.Hash().Bytes())
// 			if err := o.accumulator.Flush(); err != nil {
// 				return nil, err
// 			}
// 			o.heads.Dequeue()
// 			continue
// 		}
//
// 		// block already known to verifier
// 		if tree.Has(head.Hash()) {
// 			o.heads.Dequeue()
// 			continue
// 		}
//
// 		if !tree.Has(head.ParentHash) {
// 			panic(fmt.Sprintf("inconsistent block tree: %s", head.ParentHash))
// 		}
//
// 		size := int64(math.Ceil(float64(head.Size())))
// 		if limit < size {
// 			break
// 		}
//
// 		o.heads.Dequeue()
// 		heads = append(heads, head)
// 		if err := tree.Add(head.ParentHash, head.Hash()); err != nil {
// 			panic(fmt.Sprintf("Fail to append block hash: parent: %v, hash: %v cause: %s\n",
// 				head.ParentHash, head.Hash(), err.Error()))
// 		}
// 		limit -= size
//
// 		var err error
// 		if confirmations, err = o.confirm(head.Hash(), tree.Root()); err != nil {
// 			return nil, err
// 		} else {
// 			if len(confirmations) > 0 && o.msgs.Len() > 0 {
// 				msg, _ := o.msgs.First().(*BTPMessageCenterMessage)
// 				// TODO improve duplicated bytes comparison
// 				for _, confirmation := range confirmations {
// 					if bytes.Equal(msg.Raw.BlockHash.Bytes(), confirmation.Bytes()) {
// 						break
// 					}
// 				}
// 			}
// 		}
// 	}
//
// 	if len(confirmations) > 0 {
// 		tree.Prune(confirmations[0])
// 		for _, cnf := range confirmations {
// 			o.accumulator.AddHash(cnf.Bytes())
// 			if err := o.accumulator.Flush(); err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
//
// 	if len(heads) > 0 {
// 		return []link.BlockUpdate{
// 			BSCBlockUpdate{
// 				heads:  heads,
// 				height: uint64(status.Verifier.Height + int64(len(confirmations))),
// 				status: &VerifierStatus{
// 					Offset:    uint64(o.accumulator.Offset()),
// 					BlockTree: tree,
// 				},
// 			},
// 		}, nil
// 	} else {
// 		return []link.BlockUpdate{}, nil
// 	}
// }

func (o *receiver) BuildBlockProof(status *btp.BMCLinkStatus, height int64) (link.BlockProof, error) {
	fmt.Println("BuildBlockProof Height=", status.Verifier.Height, "RxSeq=", status.RxSeq)
	for o.accumulator.Height() < status.Verifier.Height+1 {
		head, _ := o.heads.Dequeue().(*types.Header)
		if head.Number.Int64() != o.accumulator.Height() {
			panic(fmt.Sprintf("Forbidden block number: actual: %d, expect: %d\n", head.Number.Int64(), o.accumulator.Height()))
		}

		o.accumulator.AddHash(head.Hash().Bytes())
		if err := o.accumulator.Flush(); err != nil {
			return nil, err
		}
	}
	_, witness, err := o.accumulator.WitnessForAt(height+1, status.Verifier.Height+1, o.accumulator.Offset())
	if err != nil {
		return nil, err
	}
	head, err := o.client.HeaderByNumber(context.Background(), big.NewInt(height))
	if err != nil {
		return nil, err
	}
	return BSCBlockProof{
		Header:    head,
		AccHeight: uint64(status.Verifier.Height + 1),
		Witness:   mta.WitnessesToHashes(witness),
	}, nil
}

func (o *receiver) BuildMessageProof(status *btp.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	fmt.Println("BuildMessageProof Height=", status.Verifier.Height, "RxSeq=", status.RxSeq)
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		return nil, err
	}

	// TODO use lru cache
	mpts := make(map[common.Hash]*trie.Trie)
	var mp *BSCMessageProof
	sequence := uint64(status.RxSeq + 1)
	if o.msgs.Len() > 0 {
		msg, _ := o.msgs.First().(*BTPMessageCenterMessage)
		if msg.Sequence.Uint64() < sequence {
			o.msgs.Dequeue()
		}

		if msg.Sequence.Uint64() > sequence {
			panic(fmt.Sprintf("discontinuous message sequence: actual: %d expect: %d", msg.Sequence.Uint64(), sequence))
		}

		if msg.Sequence.Uint64() == sequence {
			hash := msg.Raw.BlockHash
			if _, ok := mpts[msg.Raw.BlockHash]; !ok {
				ctx := context.Background()
				if receipts, err := o.client.ReceiptsByBlockHash(ctx, hash); err != nil {
					return nil, err
				} else {
					if mpt, err := receiptTrie(receipts); err != nil {
						return nil, err
					} else {
						mpts[hash] = mpt
					}
				}
			}

			key, err := rlp.EncodeToBytes(msg.Raw.TxIndex)
			if err != nil {
				return nil, err
			}

			proof, err := receiptProof(mpts[hash], key)
			if err != nil {
				return nil, err
			}

			if mp == nil {
				mp = &BSCMessageProof{
					Hash: hash,
					Proofs: []BSCReceiptProof{
						BSCReceiptProof{
							Key:   key,
							Proof: proof,
						},
					},
				}
			} else {
				mp.Proofs = append(mp.Proofs, BSCReceiptProof{
					Key:   key,
					Proof: proof,
				})
			}
			o.msgs.Dequeue()
			// TODO decrease limit
		}
	}
	return *mp, nil
}

func (o *receiver) BuildRelayMessage(parts []link.RelayMessageItem) ([]byte, error) {
	msg := &BSCRelayMessage{
		TypePrefixedMessages: make([]BSCTypePrefixedMessage, 0),
	}
	for _, part := range parts {
		blob, err := rlp.EncodeToBytes(part)
		if err != nil {
			return nil, err
		}

		fmt.Println("parts:", len(parts), " type:", typeToUint(part.Type()))
		msg.TypePrefixedMessages = append(msg.TypePrefixedMessages, BSCTypePrefixedMessage{
			Type:    typeToUint(part.Type()),
			Payload: blob,
		})
	}

	blob, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return nil, err
	}

	return blob, err
}

func (o *receiver) FinalizedStatus(bls <-chan *btp.BMCLinkStatus) {
}

// ensure initial merkle accumulator and snapshot
func (o *receiver) prepare() error {
	number := big.NewInt(o.accumulator.Offset())
	if number.Uint64()%o.epoch != 0 {
		panic(fmt.Sprintf("Must be epoch block: epoch: %d number: %d\n", number.Uint64(), o.epoch))
	}
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		return err
	}

	// No initial trusted block hash
	if o.accumulator.Len() <= 0 {
		o.accumulator.AddHash(head.Hash().Bytes())
		if err := o.accumulator.Flush(); err != nil {
			return err
		}
	}

	ok, err := hasSnapshot(o.database, head.Hash())
	if err != nil {
		return err
	}

	if !ok {
		valBytes := head.Extra[extraVanity : len(head.Extra)-extraSeal]
		vals, err := ParseValidators(valBytes)
		if err != nil {
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
				return err
			}

			oldValBytes := oldHead.Extra[extraVanity : len(oldHead.Extra)-extraSeal]
			oldVals, err := ParseValidators(oldValBytes)
			if err != nil {
				return err
			}

			recents := make([]common.Address, 0)
			for i := head.Number.Int64() - int64(len(oldVals)/2); i <= head.Number.Int64(); i++ {
				if oldHead, err = o.client.HeaderByNumber(context.Background(), big.NewInt(i)); err != nil {
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
	if err := o.snapshots.ensure(hash); err != nil {
		return err
	}

	snap, err := o.snapshots.get(hash)
	if err != nil {
		return err
	}

	// synchronize accumulator
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

// synchronize `merkle accumulator` and `snapshot` to state of `until` block
//func (o *receiver) synchronize(until *big.Int) error {
//	// synchronize up to `target` block
//	target, err := o.client.HeaderByNumber(context.Background(), until)
//	if err != nil {
//		return err
//	}
//
//	hash := target.Hash()
//	heads := make([]*types.Header, 0)
//	for {
//		ok, err := hasSnapshot(o.database, hash)
//		if err != nil {
//			return err
//		} else if ok {
//			break
//		}
//
//		if head, err := o.client.HeaderByHash(context.Background(), hash); err != nil {
//			return err
//		} else {
//			hash = head.ParentHash
//			heads = append(heads, head)
//		}
//	}
//
//	snap, err := loadSnapshot(o.database, hash)
//	if err != nil {
//		return err
//	}
//
//	for i := len(heads) - 1; i >= 0; i-- {
//		snap, err = snap.apply(heads[i], o.chainId)
//		if err != nil {
//			return err
//		}
//
//		if o.accumulator.Height() == int64(snap.Number+1) {
//			o.accumulator.AddHash(snap.Hash.Bytes())
//		}
//
//		o.snapshots.Add(snap.Hash, snap)
//		if snap.Number%CheckpointInterval == 0 {
//			if snap.store(o.database); err != nil {
//				return err
//			}
//		}
//	}
//	if err := o.accumulator.Flush(); err != nil {
//		return err
//	}
//	return nil
//}

//func (o *receiver) snapshot(hash common.Hash) (*Snapshot, error) {
//	// TODO consider to using network sync
//	if snap, ok := o.snapshots.Get(hash); ok {
//		return snap.(*Snapshot), nil
//	}
//	snap, err := loadSnapshot(o.database, hash)
//	if err != nil {
//		return nil, err
//	}
//	return snap, nil
//}

//func (o *receiver) confirm(from, until common.Hash) ([]common.Hash, error) {
//	// TODO test clone attacks
//	snap, err := o.snapshot(from)
//	if err != nil {
//		return nil, err
//	}
//
//	cnfs := make([]common.Hash, 0)
//	signers := make(map[common.Address]struct{})
//	// TODO extract duplicates
//	for !bytes.Equal(snap.Hash.Bytes(), until.Bytes()) {
//		signers[snap.Sealer] = struct{}{}
//		if snap.Number%o.epoch == 0 {
//			nsigners := 0
//			for k, _ := range snap.Candidates {
//				if _, ok := signers[k]; ok {
//					nsigners++
//				}
//			}
//			if len(signers) > len(snap.Validators)/2 && nsigners > len(snap.Candidates)*2/3 {
//				cnfs = append(cnfs, snap.Hash)
//			} else {
//				cnfs = cnfs[:0]
//			}
//		} else if len(cnfs) > 0 {
//			cnfs = append(cnfs, snap.Hash)
//		} else {
//			nsigners := 0
//			for k, _ := range snap.Candidates {
//				if _, ok := signers[k]; ok {
//					nsigners++
//				}
//			}
//			if nsigners > len(snap.Candidates)*2/3 {
//				cnfs = append(cnfs, snap.Hash)
//			}
//		}
//		snap, _ = o.snapshot(snap.ParentHash)
//	}
//	return cnfs, nil
//}

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

func receiptTrie(receipts types.Receipts) (*trie.Trie, error) {
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

func receiptProof(trie *trie.Trie, key []byte) ([][]byte, error) {
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
	height   int64
	sequence int64
}

func (s srcStatus) Height() int64 {
	return s.height
}

func (s srcStatus) Seq() int64 {
	return s.sequence
}
