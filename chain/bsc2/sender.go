package bsc2

import (
	"context"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

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
	src, dst   btp.BtpAddress
	chainId    *big.Int
	epoch      uint64
	client     *Client
	finality   *Snapshot
	snapshots  *Snapshots
	log        log.Logger
	wallet     btp.Wallet
	transactor *MessageTransactor
}

type SenderConfig struct {
	SrcAddress btp.BtpAddress
	DstAddress btp.BtpAddress
	Endpoint   string
	ChainID    uint64
	Epoch      uint64
}

func newSender(config SenderConfig, wallet btp.Wallet, log log.Logger) btp.Sender {
	o := &sender{
		src:     config.SrcAddress,
		dst:     config.DstAddress,
		chainId: new(big.Int).SetUint64(config.ChainID),
		epoch:   config.Epoch,
		wallet:  wallet,
		log:     log,
		client:  NewClient(config.Endpoint, config.DstAddress, config.SrcAddress, log),
	}
	o.snapshots = newSnapshots(o.chainId, o.client.Client, CacheSize, nil, log)
	o.transactor = newMessageTransactor(o.snapshots, o.log)
	return o
}

func (o *sender) Start() (<-chan *btp.RelayResult, error) {
	o.log.Traceln("++Sender::Start")
	defer o.log.Traceln("--Sender::Start")
	if err := o.prepare(); err != nil {
		return nil, err
	}

	replies := make(chan *btp.RelayResult)
	go o.transactor.Run(replies)
	go func() {
		for {
			if err := recoverable(o.watchBlockFinalities()); err != ErrRecoverable {
				o.log.Errorf("Fail to watch block finalities - err(%s)", err.Error())
				break
			}
		}
	}()
	return replies, nil
}

func (o *sender) prepare() error {
	o.log.Traceln("++Sender::prepare")
	defer o.log.Traceln("--Sender::prepare")

	latest, err := o.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	o.log.Infof("Watch from block(%d:%s)", latest.Number.Uint64(), latest.Hash())
	// the nearest epoch block
	number := new(big.Int).SetUint64(latest.Number.Uint64() - (latest.Number.Uint64() % o.epoch))
	head, err := o.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		o.log.Errorf("fail to fetching header: number(%d) error(%s)\n", number.Uint64(), err.Error())
		return err
	}

	// check block finality by the nearest epoch block
	if snap, err := BootSnapshot(o.epoch, head, o.client.Client, o.log); err != nil {
		return err
	} else {
		o.log.Debugf("make initial snapshot - number(%d) hash(%s)", snap.Number, snap.Hash.Hex())
		o.finality = snap
		o.snapshots.add(snap)
		return nil
	}
}

func (o *sender) watchBlockFinalities() error {
	o.log.Debugf("Enter Sender Loop")
	defer o.log.Debugf("Exit Sender Loop")
	headCh := make(chan *types.Header)
	number := new(big.Int).SetUint64(o.finality.Number + uint64(1))
	snap := o.finality
	calc := newBlockFinalityCalculator(o.finality.Hash, make([]common.Hash, 0), o.snapshots, o.log)
	o.log.Tracef("new calculator - addr(%p) number(%d) hash(%s)", calc, o.finality.Number, o.finality.Hash.Hex())
	var err error
	sub := o.client.WatchHeader(context.Background(), number, headCh)
	for {
		select {
		case err := <-sub.Err():
			o.log.Errorf("Watcher error(%+v)", err)
			return err
		case head := <-headCh:
			snap, err = snap.apply(head, o.chainId)
			if err != nil {
				o.log.Errorf("fail to apply snapshot - err(%+v)", err)
				sub.Unsubscribe()
				return err
			}

			if err = o.snapshots.add(snap); err != nil {
				o.log.Errorf("fail to add snapshot - err(%+v)", err)
				sub.Unsubscribe()
				return err
			}

			if fnzs, err := calc.feed(snap.Hash); err != nil {
				o.log.Errorf("fail to calculate finality - err(%+v)", err)
				sub.Unsubscribe()
				return err
			} else {
				if len(fnzs) <= 0 {
					break
				}
				fn, err := o.snapshots.get(fnzs[len(fnzs)-1])
				if err != nil {
					o.log.Errorf("fail to get snapshot - err(%+v)", err)
					sub.Unsubscribe()
					return err
				}
				o.transactor.NotifyFinality(fn.Hash)
				o.finality = fn
			}
		}
	}
}

func (o *sender) Stop() {
	// TODO
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

func (o *sender) Relay(rm btp.RelayMessage) (string, error) {
	o.log.Traceln("++Sender::Relay")
	defer o.log.Traceln("--Sender::Relay")
	if o.transactor.Busy() {
		o.log.Traceln("SenderBusy")
		return "", errors.ErrInvalidState
	}

	opts, err := bind.NewKeyedTransactorWithChainID(o.wallet.(*wallet.EvmWallet).Skey, o.chainId)
	if err != nil {
		o.log.Errorf("fail to make signed transaction - err(%+v)", err)
		return "", err
	}
	o.transactor.Send(newMessageTx(rm.Id(), o.src.String(), o.client, opts, rm.Bytes(), o.log))
	return rm.Id(), nil
}

func (o *sender) GetPreference() btp.Preference {
	o.log.Traceln("++Sender::GetPreference")
	defer o.log.Traceln("--Sender::GetPreference")
	p := btp.Preference{
		TxSizeLimit:       int64(txSizeLimit),
		MarginForLimit:    int64(0),
		LatestResult:      false,
		FilledBlockUpdate: false,
	}

	return p
}
