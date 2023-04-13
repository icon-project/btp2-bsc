package bsc

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	log       log.Logger
	pending   chan *relayResult
	executed  chan *relayResult
	wallet    wallet.Wallet
}

type SenderConfig struct {
	SrcAddress btp.BtpAddress
	DstAddress btp.BtpAddress
	Endpoint   string
	ChainID    uint64
	Epoch      uint64
}

func NewSender(config SenderConfig, wallet wallet.Wallet, log log.Logger) btp.Sender {
	o := &sender{
		src:     config.SrcAddress,
		dst:     config.DstAddress,
		chainId: new(big.Int).SetUint64(config.ChainID),
		epoch:   config.Epoch,
		wallet:  wallet,
		log:     log,

		client:   NewClient(config.Endpoint, config.DstAddress, config.SrcAddress, log),
		pending:  make(chan *relayResult, 128),
		executed: make(chan *relayResult, 128),
	}
	o.snapshots = newSnapshots(o.chainId, o.client.Client, CacheSize, nil, log)
	return o
}

func (o *sender) prepare() error {
	o.log.Traceln("++Sender::prepare")
	defer o.log.Traceln("--Sender::prepare")

	latest, err := o.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	o.log.Infof("latest header - number(%d) hash(%s)\n", latest.Number.Uint64(), latest.Hash().Hex())

	// the nearest epoch block
	number := new(big.Int).SetUint64(latest.Number.Uint64() - (latest.Number.Uint64() % o.epoch))
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)\n", number.Uint64(), err.Error())
		return err
	}

	// check block finality by the nearest epoch block
	if snap, err := BootSnapshot(o.epoch, head, o.client.Client); err != nil {
		return err
	} else {
		o.log.Debugf("make initial snapshot - number(%d) hash(%s)", snap.Number, snap.Hash.Hex())
		o.finality = snap
		o.snapshots.add(snap)
		return nil
	}
}

func (o *sender) Start() (<-chan *btp.RelayResult, error) {
	if err := o.prepare(); err != nil {
		return nil, err
	}

	finalities := make(chan *Snapshot)
	replies := make(chan *btp.RelayResult)
	go o.watchBlockFinalities(finalities)
	go o.waitAndSendResults(finalities, replies)
	return replies, nil
}

func (o *sender) watchBlockFinalities(finalities chan<- *Snapshot) {
	headCh := make(chan *types.Header)
	number := new(big.Int).SetUint64(o.finality.Number + uint64(1))
	snap := o.finality
	calc := newBlockFinalityCalculator(o.epoch, o.finality, o.snapshots, o.log)
	var err error
	o.client.WatchHeader(context.Background(), number, headCh)
	for {
		select {
		case head := <-headCh:
			snap, err = snap.apply(head, o.chainId)
			if err != nil {
				panic(err.Error())
			}

			if err = o.snapshots.add(snap); err != nil {
				panic(err.Error())
			}

			o.log.Tracef("feed new block for checking finality - number(%d) hash(%s)", snap.Number, snap.Hash.Hex())
			if fnzs, err := calc.feed(snap.Hash); err != nil {
				panic(err.Error())
			} else {
				if len(fnzs) <= 0 {
					break
				}
				fn, err := o.snapshots.get(fnzs[len(fnzs)-1])
				if err != nil {
					panic(err.Error())
				}
				o.log.Tracef("new block finality - number(%d) hash(%s)", fn.Number, fn.Hash.Hex())
				o.finality = fn
				finalities <- fn
			}
		}
	}
}

func (o *sender) Stop() {
}

func (o *sender) GetStatus() (*btp.BMCLinkStatus, error) {
	o.log.Traceln("++Sender::GetStatus")
	defer o.log.Traceln("--Sender::GetStatus")
	if status, err := o.client.BTPMessageCenter.GetStatus(&bind.CallOpts{
		Pending:     true,
		BlockNumber: new(big.Int).SetUint64(o.finality.Number),
		Context:     context.Background(),
	}, o.src.String()); err != nil {
		o.log.Errorf("fail to retrieve bmc status - err(%s)\n", err.Error())
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
	o.log.Traceln("++Sender::Relay")
	defer o.log.Traceln("--Sender::Relay")
	o.log.Tracef("pending len(%d) cap(%d) / executed len(%d) cap(%d)\n", len(o.pending), cap(o.pending), len(o.executed), cap(o.executed))
	if len(o.pending) == cap(o.pending) || len(o.executed) == cap(o.executed) {
		o.log.Warnln("sender is busy")
		return 0, errors.New("sender is busy")
	}

	key := o.wallet.(*wallet.EvmWallet).Skey
	opts, err := bind.NewKeyedTransactorWithChainID(key, o.chainId)
	if err != nil {
		o.log.Errorf("fail to create transactor - err(%s)", err.Error())
		return 0, err
	}

	tx, err := o.client.BTPMessageCenter.HandleRelayMessage(opts, o.src.String(), rm.Bytes())
	if err != nil {
		return 0, err
	}

	result := &relayResult{
		tx:      tx,
		receipt: nil,
		RelayResult: &btp.RelayResult{
			Id:        rm.Id(),
			Err:       errors.UnknownError,
			Finalized: false,
		},
	}
	o.pending <- result
	return 0, nil
}

// state transitions :)
// Pending(P), Dropped(D), Executed(E), Finalized(F), Reply(R)
// P -> D -> R
// P -> E -> R
// P -> E -> F -> R
// P -> E -> D -> R
func (o *sender) waitAndSendResults(finalities <-chan *Snapshot, replies chan<- *btp.RelayResult) {
	waitings := make([]*relayResult, 0)
	for {
		select {
		case r := <-o.pending:
			o.log.Debugf("process pending message - hash(%s)", r.tx.Hash().Hex())
			go func(r *relayResult) {
				hash := r.tx.Hash()
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
						time.Sleep(2 * time.Second)
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
			}(r)
		case r := <-o.executed:
			o.log.Debugln("process executed message - hash(%s)", r.receipt.TxHash.Hex())
			// TODO extract methods
			go func(r *relayResult) {
				if r.receipt.Status == types.ReceiptStatusFailed {
					from, err := types.Sender(types.NewEIP155Signer(r.tx.ChainId()), r.tx)
					if err != nil {
						panic("TODO err:" + err.Error())
					}

					if _, err = o.client.CallContract(context.Background(), ethereum.CallMsg{
						From:     from,
						To:       r.tx.To(),
						Gas:      r.tx.Gas(),
						GasPrice: r.tx.GasPrice(),
						Value:    r.tx.Value(),
						Data:     r.tx.Data(),
					}, r.receipt.BlockNumber); err != nil {
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
					o.log.Debugf("success handle message - hash(%s)", r.receipt.TxHash.Hex())
					r.Err = errors.SUCCESS
					waitings = append(waitings, r)
				}
				replies <- r.RelayResult
			}(r)
		case snap := <-finalities:
			o.log.Tracef("on finalized - number(%d) hash(%s)\n", snap.Number, snap.Hash.Hex())
			for i, r := range waitings {
				o.log.Debugf("waiting peer finality - number(%d) tx(%s)",
					r.receipt.BlockNumber.Uint64(), r.receipt.TxHash.Hex())
				number := r.receipt.BlockNumber.Uint64()
				if r.receipt.BlockNumber.Uint64() > snap.Number {
					continue
				}
				for number < snap.Number {
					o.log.Debugf("fin check target(%d) number(%d)\n", number, snap.Number)
					var err error
					snap, err = o.snapshots.get(snap.ParentHash)
					if err != nil {
						panic(fmt.Sprintf("No block snapshot - number(%d) hash(%s)\n",
							r.receipt.BlockNumber, r.tx.Hash().Hex()))
					}
				}

				if !bytes.Equal(r.receipt.BlockHash.Bytes(), snap.Hash.Bytes()) {
					// dropped tx
					o.log.Panicf("message in unknown block - number(%d) expected(%s) actual(%s)",
						number, snap.Hash.Hex(), r.receipt.BlockHash.Hex())
					return
				}

				o.log.Infof("finalize message - number(%d) hash(%s)",
					r.receipt.BlockNumber, r.receipt.BlockHash.Hex())
				r.Finalized = true
				// remove finalized message
				waitings[i] = waitings[len(waitings)-1]
				waitings = waitings[:len(waitings)-1]
				replies <- r.RelayResult
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
