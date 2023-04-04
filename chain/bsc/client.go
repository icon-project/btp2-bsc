package bsc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"

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
	log              log.Logger
}

func NewClient(url string, from, to btp.BtpAddress, log log.Logger) *client {
	o := &client{
		from: from,
		to:   to,
		log:  log,
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
	return receipts, nil
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
				if err == ethereum.NotFound {
					time.Sleep(time.Second)
					continue
				} else {
					// TODO handle error
					panic(err)
				}
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
			channel <- data
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
				panic(fmt.Sprintf("TODO:) watch header ctx done - error(%s)\n", ctx.Err()))
			default:
				if head, err := o.HeaderByNumber(ctx, number); err != nil {
					if err == ethereum.NotFound {
						o.log.Debugf("not found - number(%d)\n", number.Uint64())
						time.Sleep(time.Second)
					} else {
						panic(fmt.Sprintf("TODO:) fail to fetching header - error(%s)\n", err.Error()))
					}
				} else {
					number.Add(number, big.NewInt(1))
					channel <- head
				}
			}
		}
	}()
	return nil
}

func (o *client) MessagesByBlockHash(ctx context.Context, hash common.Hash) ([]*BTPMessageCenterMessage, error) {
	head, err := o.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	number := head.Number.Uint64()
	if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
		Context: ctx,
		Start:   number,
		End:     &number,
	}, []string{
		string(o.to),
	}, nil); err != nil {
		o.log.Errorf("fail to fetching btp message - block hash(%s) err(%s)\n", hash.Hex(), err.Error())
		return nil, err
	} else {
		msgs := make([]*BTPMessageCenterMessage, 0)
		for iter.Next() {
			msgs = append(msgs, iter.Event)
		}
		return msgs, nil
	}
}

func (o *client) FetchMissingMessages(ctx context.Context, from, until uint64, sequence uint64) ([]*BTPMessageCenterMessage, error) {

	if sequence > 0 {
		if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
			Context: ctx,
			Start:   from,
			End:     &until,
		}, []string{
			//string(o.to),
		}, []*big.Int{
			new(big.Int).SetUint64(sequence),
		}); err != nil {
			panic(err)
		} else {
			iter.Next()
			m := iter.Event
			from = m.Raw.BlockNumber
			o.log.Debugf("FinalizedMessage BLK Number(%d)\n", from)
		}
	}

	msgs := make([]*BTPMessageCenterMessage, 0)
	if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
		Context: ctx,
		Start:   from,
		End:     &until,
	}, []string{
		string(o.to),
	}, nil); err != nil {
		panic(err)
	} else {
		for iter.Next() {
			o.log.Debugf("Missing Message: number(%d) sequence(%d)\n", iter.Event.Raw.BlockNumber, iter.Event.Seq.Uint64())
			msgs = append(msgs, iter.Event)
		}
		return msgs, nil
	}
}

func (o *client) MessageBySequence(opts *bind.FilterOpts, sequence uint64) (*BTPMessageCenterMessage, error) {
	if iter, err := o.BTPMessageCenter.FilterMessage(opts, []string{
		string(o.to),
	}, []*big.Int{
		new(big.Int).SetUint64(sequence),
	}); err != nil {
		return nil, err
	} else {
		iter.Next()
		return iter.Event, nil
	}
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

func (o *client) WatchStatus(ctx context.Context, channel chan<- *TypesLinkStatus) error {
	head := make(chan *types.Header)
	if _, err := o.SubscribeNewHead(ctx, head); err != nil {
		return err
	}

	var oldStatus *TypesLinkStatus
	go func() {
		for {
			select {
			case <-head:
				newStatus, err := o.BTPMessageCenter.GetStatus(nil, o.to.String())
				if err != nil {
					panic(fmt.Sprintf("TODO:) watch status error - error(%s)\n", err.Error()))
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

func diff(v1, v2 *TypesLinkStatus) bool {
	return v1.TxSeq.Cmp(v2.TxSeq) != 0 && v1.Verifier.Height.Cmp(v2.Verifier.Height) != 0
}
