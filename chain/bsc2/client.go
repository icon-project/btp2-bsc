package bsc2

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/icon-project/btp2/common/log"
	btp "github.com/icon-project/btp2/common/types"
)

type Client struct {
	*ethclient.Client
	BTPMessageCenter *BTPMessageCenter
	from             btp.BtpAddress
	to               btp.BtpAddress
	log              log.Logger
}

type transport struct {
	transport http.RoundTripper
	log       log.Logger
}

func (o *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	defer req.Body.Close()
	var attempts int
	var res *http.Response
	var err error
	body := make([]byte, req.ContentLength)
	if _, err := req.Body.Read(body); err != nil {
		o.log.Panicln(err.Error())
	}

	for attempts < 5 {
		time.Sleep(time.Duration(attempts) * time.Second)
		cloned := req.Clone(context.Background())
		cloned.Body = io.NopCloser(bytes.NewBuffer(body))
		if res, err = o.transport.RoundTrip(cloned); err != nil {
			o.log.Debugf("RoundTrip Error(%s)", err.Error())
			return nil, err
		} else if res.StatusCode >= 500 {
			o.log.Warnf("server faults - code(%d)", res.StatusCode)
			attempts++
			continue
		} else {
			return res, nil
		}
	}
	o.log.Debugf("Max trial - err(%+v)", err)
	return nil, err
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

func NewClient(endpoint string, from, to btp.BtpAddress, log log.Logger) *Client {
	o := &Client{
		from: from,
		to:   to,
		log:  log,
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		o.log.Panicf("fail to parse endpoint - endpoint(%s) err(%s)", endpoint, err.Error())
	}

	var rc *rpc.Client
	switch u.Scheme {
	case "http", "https":
		if rc, err = rpc.DialHTTPWithClient(endpoint, &http.Client{
			Transport: &transport{
				transport: http.DefaultTransport,
				log:       o.log,
			},
		}); err != nil {
			o.log.Panicf("fail to make rpc connection based on http(s) - endpoint(%s) err(%s)", endpoint, err.Error())
		}
	default:
		if rc, err = rpc.DialContext(context.Background(), endpoint); err != nil {
			o.log.Panicf("fail to make rpc connection - endpoint(%s) err(%s)", endpoint, err.Error())
		}
	}

	o.Client = ethclient.NewClient(rc)
	if bmc, err := NewBTPMessageCenter(common.HexToAddress(from.Account()), o.Client); err != nil {
		o.log.Panicf("fail to create bmc proxy - address(%s) err(%s)", from.Account(), err.Error())
	} else {
		o.BTPMessageCenter = bmc
	}

	return o
}

func (o *Client) ReceiptsByBlockHash(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	block, err := o.BlockByHash(ctx, hash)
	if err != nil {
		o.log.Warnf("fail to fetch block - err(%+v)", err)
		return nil, err
	}

	var receipts []*types.Receipt
	for _, transaction := range block.Transactions() {
		receipt, err := o.TransactionReceipt(ctx, transaction.Hash())
		if err != nil {
			o.log.Warnf("fail to fetch receipts - err(%+v)", err)
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

func (o *Client) WatchHeader(ctx context.Context, number *big.Int, headCh chan<- *types.Header) *Subscription {
	return NewSubscription(func(quit <-chan struct{}) error {
		for {
			select {
			case <-quit:
				return nil
			default:
				if head, err := o.Client.HeaderByNumber(ctx, number); err != nil {
					if err == ethereum.NotFound {
						time.Sleep(3 * time.Second)
					} else if err != nil {
						return err
					}
				} else {
					number.Add(number, big.NewInt(1))
					headCh <- head
				}
			}
		}
	})
}

func (o *Client) MessagesByBlockHash(ctx context.Context, hash common.Hash) ([]*BTPMessageCenterMessage, error) {
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

func (o *Client) MessageBySequence(opts *bind.FilterOpts, sequence uint64) (*BTPMessageCenterMessage, error) {
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

func (o *Client) MessagesAfterSequence(opts *bind.FilterOpts, sequence uint64) ([]*BTPMessageCenterMessage, error) {
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
