package bsc

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
	"github.com/icon-project/btp/common/db"
	"github.com/icon-project/btp/common/link"
	"github.com/icon-project/btp/common/log"
	"github.com/icon-project/btp/common/mta"
	btp "github.com/icon-project/btp/common/types"
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
	snapshots   *lru.ARCCache
	heads       *WaitQueue
	msgs        *WaitQueue
	mutex       sync.Mutex
	status      *srcStatus
	log         log.Logger
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

	if snaps, err := lru.NewARC(CacheSize); err != nil {
		panic(err)
	} else {
		o.snapshots = snaps
	}

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
	return o
}

func (o *receiver) Start(status *btp.BMCLinkStatus) (<-chan link.ReceiveStatus, error) {
	outCh := make(chan link.ReceiveStatus)
	linkCh := make(chan *TypesLinkStats)
	finnum := big.NewInt(status.Verifier.Height)
	go func() {
		if err := o.prepare(); err != nil {
			panic(err)
		}

		if err := o.synchronize(finnum); err != nil {
			panic(err)
		}

		headCh := make(chan *types.Header)
		o.client.WatchHeader(context.Background(), big.NewInt(1+finnum.Int64()), headCh)
		go func() {
			vs := &VerifierStatus{}
			if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
				return nil, err
			}
			tree := vs.BlockTree

			var snap *Snapshot
			for {
				select {
				case head := <-headCh:
					var err error
					if snap == nil {
						snap, err = o.snapshot(head.ParentHash)
						if err != nil {
							panic(err)
						}
					}
					if snap, err = snap.apply(head, o.chainId); err != nil {
						panic(err)
					} else {
						o.snapshots.Add(snap.Hash, snap)
						if snap.Number%CheckpointInterval == 0 {
							if snap.store(o.database); err != nil {
								panic(err)
							}
						}
					}

					o.heads.Enqueue(head)
					tree.Add(snap.ParentHash, snap.Hash)
					//if confirmations, err := o.confirm(snap.Hash, tree.Root())
				}
			}
		}()

		sequence := big.NewInt(status.RxSeq)
		msgCh := make(chan *BTPMessageCenterMessage)
		fromBlock := o.start
		if sequence.Uint64() > 0 {
			end := finnum.Uint64()
			if prevmsg, err := o.client.FindMessage(context.Background(), o.start, &end, sequence); err != nil {
				panic(err)
			} else {
				fromBlock = prevmsg.Raw.BlockNumber
			}
		}

		ctx := context.Background()
		next := big.NewInt(sequence.Int64() + 1)
		if err := o.client.WatchMessage(ctx, fromBlock, next, msgCh); err != nil {
			panic(err)
		}
		go func() {
			for {
				select {
				case msg := <-msgCh:
					o.msgs.Enqueue(msg)
				case <-ctx.Done():
					fmt.Println("TODO:)", ctx.Err())
					return
				}
			}
		}()

		o.client.WatchStatus(context.Background(), linkCh)
		for {
			select {
			case newStatus := <-linkCh:
				o.mutex.Lock()
				o.status.height = newStatus.Verifier.Height.Int64()
				o.status.sequence = newStatus.TxSeq.Int64()
				outCh <- o.status
				o.mutex.Unlock()
			}
		}
	}()
	return outCh, nil
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

func (o *receiver) GetMarginForLimit() int64 {
	return 0
}

func (o *receiver) GetHeightForSeq(sequence int64) int64 {
	msg, err := o.client.FindMessage(context.Background(), o.start, nil, big.NewInt(sequence))
	if err != nil {
		return -1
	}
	return int64(msg.Raw.BlockNumber)
}

func (o *receiver) BuildBlockUpdate(status *btp.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	fmt.Println("BuildBlockUpdate")
	vs := &VerifierStatus{}
	if err := rlp.DecodeBytes(status.Verifier.Extra, vs); err != nil {
		return nil, err
	}

	number := status.Verifier.Height
	tree := vs.BlockTree
	heads := make([]*types.Header, 0)
	var confirmations []common.Hash
	// exit loop
	// 1) `block updates` size exceeds limited size
	// 2) `btp message` exists
	for o.heads.Len() > 0 {
		head, _ := o.heads.First().(*types.Header)

		// block already finalized by verifier
		if head.Number.Int64() <= number {
			o.accumulator.AddHash(head.Hash().Bytes())
			if err := o.accumulator.Flush(); err != nil {
				return nil, err
			}
			o.heads.Dequeue()
			continue
		}

		// block already known to verifier
		if tree.Has(head.Hash()) {
			o.heads.Dequeue()
			continue
		}

		if !tree.Has(head.ParentHash) {
			panic(fmt.Sprintf("inconsistent block tree: %s", head.ParentHash))
		}

		size := int64(math.Ceil(float64(head.Size())))
		if limit < size {
			break
		}

		o.heads.Dequeue()
		heads = append(heads, head)
		if err := tree.Add(head.ParentHash, head.Hash()); err != nil {
			panic(fmt.Sprintf("Fail to append block hash: parent: %v, hash: %v cause: %s\n",
				head.ParentHash, head.Hash(), err.Error()))
		}
		limit -= size

		var err error
		if confirmations, err = o.confirm(head.Hash(), tree.Root()); err != nil {
			return nil, err
		} else {
			if len(confirmations) > 0 && o.msgs.Len() > 0 {
				msg, _ := o.msgs.First().(*BTPMessageCenterMessage)
				// TODO improve duplicated bytes comparison
				for _, confirmation := range confirmations {
					if bytes.Equal(msg.Raw.BlockHash.Bytes(), confirmation.Bytes()) {
						break
					}
				}
			}
		}
	}

	if len(confirmations) > 0 {
		tree.Prune(confirmations[0])
		for _, cnf := range confirmations {
			o.accumulator.AddHash(cnf.Bytes())
			if err := o.accumulator.Flush(); err != nil {
				return nil, err
			}
		}
	}

	if len(heads) > 0 {
		return []link.BlockUpdate{
			BSCBlockUpdate{
				heads:  heads,
				height: uint64(status.Verifier.Height + int64(len(confirmations))),
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

func (o *receiver) BuildBlockProof(status *btp.BMCLinkStatus, height int64) (link.BlockProof, error) {
	fmt.Println("BuildBlockProof")
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
	fmt.Println("BuildMessageProof")
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

// synchronize `merkle accumulator` and `snapshot` to state of `until` block
func (o *receiver) synchronize(until *big.Int) error {
	// synchronize up to `target` block
	target, err := o.client.HeaderByNumber(context.Background(), until)
	if err != nil {
		return err
	}

	hash := target.Hash()
	heads := make([]*types.Header, 0)
	for {
		ok, err := hasSnapshot(o.database, hash)
		if err != nil {
			return err
		} else if ok {
			break
		}

		if head, err := o.client.HeaderByHash(context.Background(), hash); err != nil {
			return err
		} else {
			hash = head.ParentHash
			heads = append(heads, head)
		}
	}

	snap, err := loadSnapshot(o.database, hash)
	if err != nil {
		return err
	}

	for i := len(heads) - 1; i >= 0; i-- {
		snap, err = snap.apply(heads[i], o.chainId)
		if err != nil {
			return err
		}

		if o.accumulator.Height() == int64(snap.Number+1) {
			o.accumulator.AddHash(snap.Hash.Bytes())
		}

		o.snapshots.Add(snap.Hash, snap)
		if snap.Number%CheckpointInterval == 0 {
			if snap.store(o.database); err != nil {
				return err
			}
		}
	}
	if err := o.accumulator.Flush(); err != nil {
		return err
	}
	return nil
}

func (o *receiver) snapshot(hash common.Hash) (*Snapshot, error) {
	// TODO consider to using network sync
	if snap, ok := o.snapshots.Get(hash); ok {
		return snap.(*Snapshot), nil
	}
	snap, err := loadSnapshot(o.database, hash)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (o *receiver) confirm(from, until common.Hash) ([]common.Hash, error) {
	// TODO test clone attacks
	snap, err := o.snapshot(from)
	if err != nil {
		return nil, err
	}

	cnfs := make([]common.Hash, 0)
	signers := make(map[common.Address]struct{})
	// TODO extract duplicates
	for !bytes.Equal(snap.Hash.Bytes(), until.Bytes()) {
		signers[snap.Sealer] = struct{}{}
		if snap.Number%o.epoch == 0 {
			nsigners := 0
			for k, _ := range snap.Candidates {
				if _, ok := signers[k]; ok {
					nsigners++
				}
			}
			if len(signers) > len(snap.Validators)/2 && nsigners > len(snap.Candidates)*2/3 {
				cnfs = append(cnfs, snap.Hash)
			} else {
				cnfs = cnfs[:0]
			}
		} else if len(cnfs) > 0 {
			cnfs = append(cnfs, snap.Hash)
		} else {
			nsigners := 0
			for k, _ := range snap.Candidates {
				if _, ok := signers[k]; ok {
					nsigners++
				}
			}
			if nsigners > len(snap.Candidates)*2/3 {
				cnfs = append(cnfs, snap.Hash)
			}
		}
		snap, _ = o.snapshot(snap.ParentHash)
	}
	return cnfs, nil
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
