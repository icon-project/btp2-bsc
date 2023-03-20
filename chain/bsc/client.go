package bsc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/icon-project/btp2/common/log"
	btp "github.com/icon-project/btp2/common/types"
)

const (
	WindowSize = 1024
)

type client struct {
	*ethclient.Client
	BTPMessageCenter *BTPMessageCenter
	from             btp.BtpAddress
	to               btp.BtpAddress
}

func NewClient(url string, from, to btp.BtpAddress, log log.Logger) *client {
	o := &client{
		from: from,
		to:   to,
	}

	if client, err := ethclient.Dial(url); err != nil {
		panic(err)
	} else {
		o.Client = client
	}

	if bmc, err := NewBTPMessageCenter(common.HexToAddress(from.Account()), o.Client); err != nil {
		panic(err)
	} else {
		o.BTPMessageCenter = bmc
	}

	return o
}

func (o *client) ReceiptsByBlockHash(ctx context.Context, hash common.Hash) (types.Receipts, error) {

	block, err := o.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	var receipts []*types.Receipt
	for _, transaction := range block.Transactions() {
		receipt, err := o.TransactionReceipt(ctx, transaction.Hash())
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return types.Receipts(receipts), nil
}

type BTPData struct {
	header   *types.Header
	messages []*BTPMessageCenterMessage
}

func (o *client) WatchBTPData(ctx context.Context, number *big.Int, channel chan<- *BTPData) error {
	go func() {
		for {
			data := &BTPData{
				messages: make([]*BTPMessageCenterMessage, 0),
			}
			if head, err := o.HeaderByNumber(ctx, number); err != nil {
				// TODO handle error
				return
			} else {
				data.header = head
			}

			n := number.Uint64()
			if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
				Start:   n,
				End:     &n,
				Context: ctx,
			}, []string{
				string(o.to),
			}, nil); err != nil {
				// TODO
				panic(err)
			} else {
				for iter.Next() {
					data.messages = append(data.messages, iter.Event)
				}
			}
			fmt.Println("Watch HEAD1):", data.header.Number.Uint64())
			channel <- data
			fmt.Println("Watch HEAD2):", data.header.Number.Uint64())
			number.Add(number, big.NewInt(1))
		}
	}()
	return nil
}

func (o *client) WatchHeader(ctx context.Context, number *big.Int, channel chan<- *types.Header) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("TODO:)", ctx.Err())
				return
			default:
				head, err := o.HeaderByNumber(ctx, number)
				if err != nil {
					// TODO handle error
					return
				}
				number.Add(number, big.NewInt(1))
				channel <- head
			}
		}
	}()
	return nil
}

func (o *client) WatchMessage(ctx context.Context, from uint64, sequence *big.Int, channel chan<- *BTPMessageCenterMessage) error {
	duration := 500 * time.Millisecond
	if sequence.Uint64() <= 0 {
		return errors.New("sequence should be bigger than zero")
	}

	var _End uint64
	if latest, err := o.BlockNumber(context.Background()); err != nil {
		return err
	} else {
		if from+WindowSize < latest {
			_End = uint64(from + WindowSize)
		} else {
			_End = latest
		}
	}

	_sequence := big.NewInt(sequence.Int64())
	opts := bind.FilterOpts{
		Context: ctx,
		Start:   from,
		End:     &_End,
	}

	go func() {
		for {
			iter, err := o.BTPMessageCenter.FilterMessage(&opts, []string{
				string(o.to),
			}, []*big.Int{
				_sequence,
			})

			if err != nil {
				// TODO handle error
				panic(err)
			}

			if iter.Next() {
				channel <- iter.Event
				opts.Start = iter.Event.Raw.BlockNumber
				_sequence.Add(_sequence, big.NewInt(1))
			} else {
				if opts.Start == *opts.End {
					var once sync.Once
					once.Do(func() {
						duration = 3 * time.Second
					})
				}
				opts.Start = *opts.End
				time.Sleep(duration)
			}

			if latest, err := o.BlockNumber(context.Background()); err != nil {
				// TODO handle error
				panic(err)
			} else {
				if opts.Start+WindowSize <= latest {
					_End = opts.Start + WindowSize
				} else {
					_End = latest
				}
			}
		}
	}()

	return nil
}

// func (o *client) MessagesByNumber(ctx context.Context, number uint64) ([]*BTPMessageCenterMessage, error) {
// 	if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
// 		Context: ctx,
// 		Start:   number,
// 		End:     &number,
// 	}, []string{
// 		string(o.to),
// 	}, nil); err != nil {
// 		return nil, err
// 	} else {
// 		msgs := make([]*BTPMessageCenterMessage, 0)
// 		for it.Next() {
// 			msgs = append(msgs, it.Event)
// 		}
// 		it.Close()
// 		return msgs, nil
// 	}
// }

// func (o *client) FindMessages(ctx context.Context, start uint64, end *uint64) ([]*BTPMessageCenterMessage, error) {
//
// }

func (o *client) FindMessage(ctx context.Context, start uint64, end *uint64, sequence *big.Int) (*BTPMessageCenterMessage, error) {

	var limit uint64
	if end == nil {
		if latest, err := o.BlockNumber(ctx); err != nil {
			return nil, err
		} else {
			limit = latest
		}
	} else {
		limit = *end
	}

	var pos uint64
	if start+WindowSize <= limit {
		pos = start + WindowSize
	} else {
		pos = limit
	}

	opts := bind.FilterOpts{
		Start:   start,
		End:     &pos,
		Context: ctx,
	}

	for {
		if iter, err := o.BTPMessageCenter.FilterMessage(&opts, []string{
			string(o.to),
		}, []*big.Int{
			sequence,
		}); err != nil {
			return nil, err
		} else {
			if iter.Next() {
				return iter.Event, nil
			}
		}

		opts.Start = *opts.End
		if pos+WindowSize <= limit {
			pos += WindowSize
		} else {
			if opts.Start == *opts.End {
				return nil, errors.New("NoMessage")
			}
			pos = limit
		}
	}
}

func (o *client) WatchStatus(ctx context.Context, channel chan<- *TypesLinkStats) error {
	head := make(chan *types.Header)
	if _, err := o.SubscribeNewHead(ctx, head); err != nil {
		return err
	}

	var oldStatus *TypesLinkStats
	go func() {
		for {
			select {
			case <-head:
				newStatus, err := o.BTPMessageCenter.GetStatus(nil, o.to.String())
				if err != nil {
					fmt.Println("err:", err)
					// TODO handle error
					return
				}
				if oldStatus == nil || diff(oldStatus, &newStatus) {
					oldStatus = &newStatus
					channel <- oldStatus
				}
			}
		}
	}()
	return nil
}

func diff(v1, v2 *TypesLinkStats) bool {
	var x common.Hash
	x = common.Hash{}
	fmt.Println(x)
	return v1.TxSeq.Cmp(v2.TxSeq) != 0 && v1.Verifier.Height.Cmp(v2.Verifier.Height) != 0
}
