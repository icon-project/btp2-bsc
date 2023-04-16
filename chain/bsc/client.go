package bsc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/icon-project/btp2/common/log"
	btp "github.com/icon-project/btp2/common/types"
)

type client struct {
	*ethclient.Client
	BTPMessageCenter *BTPMessageCenter
	from             btp.BtpAddress
	to               btp.BtpAddress
	log              log.Logger
}

func ChainID(url string) uint64 {
	client, err := ethclient.Dial(url)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	if cid, err := client.ChainID(context.Background()); err != nil {
		panic(err)
	} else {
		return cid.Uint64()
	}
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

func (o *client) WatchHeader(ctx context.Context, number *big.Int, channel chan<- *types.Header) error {
	go func() {
		retry := 0
		for {
			select {
			case <-ctx.Done():
				panic(fmt.Sprintf("TODO:) watch header ctx done - error(%s)\n", ctx.Err()))
			default:
				if head, err := o.HeaderByNumber(ctx, number); err != nil {
					if err == ethereum.NotFound {
						time.Sleep(3 * time.Second)
					} else {
						if retry < 3 {
							retry++
							o.log.Errorf("fail to fetch header - retry(%d) err(%+v)", retry, err)
							continue
						}
						panic(fmt.Sprintf("fail to fetching header - error(%+v)", err))
					}
				} else {
					retry = 0
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
		o.log.Errorf("fail to fetching btp message - block number(%d) hash(%s) err(%s)", number, hash.Hex(), err.Error())
		return nil, err
	} else {
		msgs := make([]*BTPMessageCenterMessage, 0)
		for iter.Next() {
			msgs = append(msgs, iter.Event)
		}
		return msgs, nil
	}
}

const (
	maxQueryRange = 2048
)

func (o *client) MessageBySequence(opts *bind.FilterOpts, sequence uint64) (*BTPMessageCenterMessage, error) {
	sp, ep := opts.Start, *opts.End
	if ep-opts.Start > maxQueryRange {
		sp = ep - maxQueryRange
	}

	for {
		if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
			Start:   sp,
			End:     &ep,
			Context: context.Background(),
		}, []string{
			string(o.to),
		}, []*big.Int{
			new(big.Int).SetUint64(sequence),
		}); err != nil {
			if err == ethereum.NotFound && sp != ep {
				ep = sp
				if ep-opts.Start > maxQueryRange {
					sp = ep - maxQueryRange
				} else {
					sp = opts.Start
				}
				continue
			} else {
				return nil, err
			}
		} else {
			iter.Next()
			return iter.Event, nil
		}
	}
}

func (o *client) MessagesAfterSequence(opts *bind.FilterOpts, sequence uint64) ([]*BTPMessageCenterMessage, error) {
	sp, ep := opts.Start, *opts.End
	if ep-opts.Start > maxQueryRange {
		sp = ep - maxQueryRange
	}

	msgs := make([]*BTPMessageCenterMessage, 0)
	for {
		o.log.Tracef("range search - sp(%d) ~ ep(%d)", sp, ep)
		if iter, err := o.BTPMessageCenter.FilterMessage(&bind.FilterOpts{
			Start:   sp,
			End:     &ep,
			Context: context.Background(),
		}, []string{
			string(o.to),
		}, []*big.Int{}); err != nil {
			o.log.Errorf("fail to fetch messages - err(%s)", err.Error())
			return nil, err
		} else {
			for iter.Next() {
				m := iter.Event
				if m.Seq.Uint64() <= sequence {
					continue
				} else {
					o.log.Debugf("message - sequence(%d) number(%d) hash(%s)", m.Seq.Uint64(), m.Raw.BlockNumber, m.Raw.BlockHash.Hex())
					msgs = append(msgs, m)
				}
			}
			if sp >= ep {
				return msgs, nil
			}
			ep = sp
			if ep-opts.Start > maxQueryRange {
				sp = ep - maxQueryRange
			} else {
				sp = opts.Start
			}
		}
	}
}
