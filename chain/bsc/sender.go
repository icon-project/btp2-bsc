package bsc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/log"
	btp "github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

const (
	txMaxDataSize   = 524288 //512 * 1024 // 512kB
	txOverheadScale = 0.37   //base64 encoding overhead 0.36, rlp and other fields 0.01
)

var (
	txSizeLimit = int(math.Ceil(txMaxDataSize / (1 + txOverheadScale)))
)

type relayResult struct {
	*btp.RelayResult
	tx      *types.Transaction
	receipt *types.Receipt
}

type sender struct {
	src, dst  btp.BtpAddress
	chainId   *big.Int
	epoch     uint64
	height    uint64
	client    *client
	mutex     sync.RWMutex
	finality  *Snapshot
	snapshots *Snapshots
	replies   chan *relayResult
	log       log.Logger
	pending   chan *relayResult
	executed  chan *relayResult
	finalized chan *Snapshot
	wallet    wallet.Wallet
}

func NewSender(src, dst btp.BtpAddress, w wallet.Wallet, endpoint string, opt map[string]interface{}, l log.Logger) btp.Sender {
	chainId := big.NewInt(99)
	endpoint = "ws://localhost:8545"

	o := &sender{
		src:       src,
		dst:       dst,
		chainId:   chainId,
		epoch:     uint64(200),
		client:    NewClient(endpoint, src, dst, l),
		finalized: make(chan *Snapshot),
		wallet:    w,
	}
	snapshots := newSnapshots(chainId, o.client.Client, CacheSize, nil, l)
	o.snapshots = snapshots
	return o
}

func (o *sender) prepare() error {
	o.log.Infoln("++sender::prepare")
	defer o.log.Infoln("--sender::prepare")

	latest, err := o.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	o.log.Infof("latest header - number(%d) hash(%s)\n", latest.Number.Uint64(), latest.Hash().Hex())

	// collect past headers for confirming block finality
	// near epoch block
	number := new(big.Int).SetUint64(latest.Number.Uint64() - (latest.Number.Uint64() % o.epoch))
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)\n", number.Uint64(), err.Error())
		return err
	}

	valBytes := head.Extra[extraVanity : len(head.Extra)-extraSeal]
	vals, err := ParseValidators(valBytes)
	if err != nil {
		o.log.Errorf("fail to parsing candidates set bytes - %s\n", hex.EncodeToString(valBytes))
		return err
	}

	var snap *Snapshot
	// genesis block
	if number.Uint64() == 0 {
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
	o.finality = snap
	o.snapshots.add(snap)
	if err := o.snapshots.ensure(snap.Hash); err != nil {
		return err
	}

	// TODO extract
	go func() {
		calc := newBlockFinalityCalculator(o.epoch, o.finality, o.snapshots, o.log)
		snap := o.finality
		for {
			head, err := o.client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(snap.Number))
			if err != nil {
				panic(fmt.Sprintf("fail to fetching header - number(%d) error(%s)",
					snap.Number, err.Error()))
			}

			snap, err = snap.apply(head, o.chainId)
			if err != nil {
				panic(fmt.Sprintf("fail to applying snapshot - snap number(%d) hash(%s) next header number(%d) hash(%s)\n", snap.Number, snap.Hash.Hex(), head.Number.Uint64(), head.Hash().Hex()))
			}
			if fnzs, err := calc.feed(snap.Hash); err != nil {
				o.log.Panicf("fail to calculate block finality - err(%s)\n", err.Error())
				panic(err)
			} else {
				for i, _ := range fnzs {
					final, err := o.snapshots.get(fnzs[len(fnzs)-1-i])
					if err != nil {
						TODO("fail to retrieve snapshot")
					}
					o.mutex.Lock()
					o.finality = final
					o.mutex.Unlock()
					// TODO broadcast
					o.finalized <- final
				}
			}
		}
	}()
	return nil
}

func (o *sender) Start() (<-chan *btp.RelayResult, error) {
	if err := o.prepare(); err != nil {
		return nil, err
	}
	_, err := o.client.BlockNumber(context.Background())
	if err != nil {
		panic(fmt.Sprintf("fail to fetching latest block number - err(%s)\n", err.Error()))
	}

	replies := make(chan *btp.RelayResult)
	go o.waitAndSendResults(replies)
	return replies, nil
}

func (o *sender) Stop() {
}

func (o *sender) GetStatus() (*btp.BMCLinkStatus, error) {
	if status, err := o.client.BTPMessageCenter.GetStatus(&bind.CallOpts{
		Pending:     true,
		BlockNumber: new(big.Int).SetUint64(o.finality.Number),
		Context:     context.Background(),
	}, o.dst.String()); err != nil {
		return nil, err
	} else {
		return &btp.BMCLinkStatus{
			RxSeq: status.RxSeq.Int64(),
			TxSeq: status.TxSeq.Int64(),
			Verifier: struct {
				Height int64
				Extra  []byte
			}{
				Height: status.Verifier.Height.Int64(),
				Extra:  status.Verifier.Extra,
			},
		}, nil
	}
}

// verbose return value....
func (o *sender) Relay(rm btp.RelayMessage) (int, error) {
	if len(o.pending) == cap(o.pending) || len(o.executed) == cap(o.executed) {
		o.log.Infoln("sender is busy")
		return 0, errors.New("sender is busy")
	}

	key := o.wallet.(*wallet.EvmWallet).Skey
	opts, err := bind.NewKeyedTransactorWithChainID(key, o.chainId)
	if err != nil {
		return 0, err
	}

	tx, err := o.client.BTPMessageCenter.HandleRelayMessage(opts, o.src.String(), rm.Bytes())
	if err != nil {
		return 0, err
	}

	rr := &btp.RelayResult{
		Id:        rm.Id(),
		Err:       errors.UnknownError,
		Finalized: false,
	}
	result := &relayResult{
		tx:      tx,
		receipt: nil,
	}
	result.RelayResult = rr
	o.pending <- result
	return 0, nil
}

// state transitions :)
// Pending(P), Dropped(D), Executed(E), Finalized(F), Reply(R)
// P -> D -> R
// P -> E -> R
// P -> E -> F -> R
// P -> E -> D -> R
func (o *sender) waitAndSendResults(replies chan<- *btp.RelayResult) {
	waitings := make([]*relayResult, 0)
	for {
		select {
		case r := <-o.pending:
			hash := r.tx.Hash()
			// TODO extract to methods
			attempt := 0
			for {
				_, pending, err := o.client.TransactionByHash(context.Background(), hash)
				if err != nil {
					if err == ethereum.NotFound {
						o.log.Errorf("transaction has dropped - hash(%s)\n", hash.Hex())
						r.Err = errors.UnknownError
						replies <- r.RelayResult
					} else {
						o.log.Errorf("fail to fetching transaction - hash(%s)\n", hash.Hex())
						replies <- r.RelayResult
					}
					break
				}

				if pending {
					if attempt > 3 {
						o.log.Errorf("fail to executing transaction in allowed time - hash(%s)\n", hash.Hex())
						r.Err = errors.TimeoutError
						replies <- r.RelayResult
					}
					attempt++
					time.Sleep(time.Second)
					continue
				} else {
					if receipt, err := o.client.TransactionReceipt(context.Background(), hash); err != nil {
						o.log.Errorf("fail to fetching receipt - hash(%s), err(%s)\n",
							hash.Hex(), err.Error())
					} else {
						r.receipt = receipt
						o.executed <- r
					}
					break
				}
			}
		case r := <-o.executed:
			// TODO extract methods
			if r.receipt.Status == types.ReceiptStatusFailed {
				from, err := types.Sender(types.NewEIP155Signer(r.tx.ChainId()), r.tx)
				if err != nil {
					panic("TODO err:" + err.Error())
				}

				opts := ethereum.CallMsg{
					From:     from,
					To:       r.tx.To(),
					Gas:      r.tx.Gas(),
					GasPrice: r.tx.GasPrice(),
					Value:    r.tx.Value(),
					Data:     r.tx.Data(),
				}
				_, err = o.client.CallContract(context.Background(), opts, r.receipt.BlockNumber)
				if err != nil {
					o.log.Debugf("reproduce failure transaction - hash(%s) error(%s)\n", r.tx.Hash().Hex(), err.Error())
					r.Err = errors.CodeOf(err)
					// TODO check real value
					TODO(fmt.Sprintf("err: %+v\n", err, errors.CodeOf(err)))
					// append to watings
					replies <- r.RelayResult
				} else {
					o.log.Warnf("fail to emulating reverts")
				}
				o.log.Infof("dispose of executed failure message\n")
			} else {
				r.Err = errors.SUCCESS
				// append to watings
			}
			o.replies <- r
		case snap := <-o.finalized:
			for _, r := range waitings {
				number := r.receipt.BlockNumber.Uint64()
				for number < snap.Number {
					var err error
					snap, err = o.snapshots.get(snap.ParentHash)
					if err != nil {
						panic(fmt.Sprintf("No block snapshot - number(%d) hash(%s)\n",
							r.receipt.BlockNumber, r.tx.Hash().Hex()))
					}
				}

				if bytes.Equal(r.tx.Hash().Bytes(), snap.Hash.Bytes()) {
					r.Finalized = true
					replies <- r.RelayResult
				} else {
					// dropped tx
					TODO(fmt.Sprintf("expected number(%d) hash(%s), actual number(%d) hash(%s)\n",
						snap.Number, snap.Hash.Hex(), r.receipt.BlockNumber.Uint64(), r.receipt.BlockHash.Hex()))
				}
			}
		}
	}
}

func (o *sender) GetMarginForLimit() int64 {
	return 0
}

func (o *sender) TxSizeLimit() int {
	return txSizeLimit
}

func TODO(m string) {
	_, filename, line, _ := runtime.Caller(1)
	panic(fmt.Sprintf("TODO:) %s : %d : %s\n", filename, line, m))
}
