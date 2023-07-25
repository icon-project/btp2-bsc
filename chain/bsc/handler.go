package bsc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/btp2/common/log"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	errs "github.com/icon-project/btp2/common/errors"
	btp "github.com/icon-project/btp2/common/types"
)

type MessageType int

const (
	Unknown MessageType = iota
	Created
	Pending
	Executing
	Executed
	Finalized
	Dropped
	Faulted
)

func (o MessageType) String() string {
	switch o {
	case Unknown:
		return "UnknownMessage"
	case Created:
		return "CreatedMessage"
	case Pending:
		return "PendingMessage"
	case Executing:
		return "ExecutingMessage"
	case Executed:
		return "ExecutedMessage"
	case Finalized:
		return "FinalizedMessage"
	case Dropped:
		return "DroppedMessage"
	case Faulted:
		return "FaultedMessage"
	default:
		return "UnknownMessage"
	}
}

type MessageTxHandler struct {
	snapshots  *Snapshots
	log        log.Logger
	replies    chan<- *btp.RelayResult
	finalities chan *Snapshot
	transition chan MessageTx
	m          sync.Mutex
	waitings   []MessageTx
}

func newMessageTxHandler(snapshots *Snapshots, log log.Logger) *MessageTxHandler {
	return &MessageTxHandler{
		snapshots:  snapshots,
		log:        log,
		finalities: make(chan *Snapshot),
		transition: make(chan MessageTx, 1),
		waitings:   make([]MessageTx, 0),
	}
}

func (o *MessageTxHandler) Busy() bool {
	return len(o.transition) == cap(o.transition) || len(o.waitings) == 128
}

func (o *MessageTxHandler) SetNewFinality(finality *Snapshot) {
	o.finalities <- finality
}

// C -> D
// C -> F
// C -> P -> D
// C -> P -> F
// C -> P -> E -> D
// C -> P -> E -> F
// C -> P -> E -> W -> nil
func (o *MessageTxHandler) Run(replies chan<- *btp.RelayResult) {
	o.replies = replies
	for {
		select {
		case msg := <-o.transition:
			go func() {
				if msg = msg.Transit(); msg != nil {
					o.Send(msg)
				}
			}()
		case finality := <-o.finalities:
			go func() {
				o.m.Lock()
				k := 0
				for _, msg := range o.waitings {
					if msg = o.tryFinalizeInLock(finality, msg); msg.Type() == Executed {
						o.waitings[k] = msg
						k++
					} else {
						o.Send(msg)
					}
				}
				o.waitings = o.waitings[:k]
				o.m.Unlock()
			}()
		}
	}
}

func (o *MessageTxHandler) Send(msg MessageTx) {
	typ := msg.Type()
	if typ == Faulted || typ == Dropped || typ == Executed || typ == Finalized {
		o.replies <- msg.Raw()
	}

	if typ == Created || typ == Pending || typ == Executing {
		o.transition <- msg
	}

	if impl, ok := msg.(*ExecutedMessage); ok {
		if impl.err == nil {
			o.m.Lock()
			o.waitings = append(o.waitings, msg)
			o.m.Unlock()
		}
	}
}

func (o *MessageTxHandler) tryFinalizeInLock(finality *Snapshot, msg MessageTx) MessageTx {
	exec, ok := msg.(*ExecutedMessage)
	if !ok {
		panic(fmt.Sprintf("ForbiddenMessage:%s", msg.Type()))
	}

	snap := finality
	if exec.number > snap.Number {
		return exec
	}
	var err error
	for exec.number < snap.Number {
		if snap, err = o.snapshots.get(snap.ParentHash); err != nil {
			o.log.Panicf(err.Error())
		}
	}
	if !bytes.Equal(exec.hash.Bytes(), snap.Hash.Bytes()) {
		o.log.Debugf("MessageTransition(EE->D)")
		return &DroppedMessage{
			id:   exec.id,
			err:  errors.New("TxDropped"),
			prev: exec.Type(),
		}
	}
	return exec.Transit()
}

type MessageTx interface {
	Type() MessageType
	Transit() MessageTx
	Raw() *btp.RelayResult
}

func newMessageTx(id string, from string, client *Client, opts *bind.TransactOpts, blob []byte, log log.Logger) MessageTx {
	log.Debugf("new message tx - id(%d)", id)
	return &CreatedMessage{
		log:    log,
		id:     id,
		from:   from,
		client: client,
		opts:   opts,
		blob:   blob,
	}
}

type CreatedMessage struct {
	log    log.Logger
	id     string
	from   string
	client *Client
	opts   *bind.TransactOpts
	blob   []byte
}

func (o *CreatedMessage) Type() MessageType {
	return Created
}

func (o *CreatedMessage) Transit() MessageTx {
	o.log.Tracef("CreatedMessage blob(%s)", hex.EncodeToString(o.blob))
	if tx, err := o.client.BTPMessageCenter.HandleRelayMessage(o.opts, o.from, o.blob); err != nil {
		o.log.Debugf("MessageTransition(C->D) ID(%d) Err(%s)", o.id, err.Error())
		return &DroppedMessage{
			id:  o.id,
			err: err,
		}
	} else {
		o.log.Debugf("MessageTransition(C->P) ID(%d) Tx(%s)", o.id, tx.Hash().Hex())
		return &PendingMessage{
			log:    o.log,
			id:     o.id,
			tx:     tx,
			client: o.client,
		}
	}
}

func (o *CreatedMessage) Raw() *btp.RelayResult {
	panic("Unsupported")
}

type PendingMessage struct {
	log    log.Logger
	id     string
	tx     *types.Transaction
	client *Client
}

func (o *PendingMessage) Type() MessageType {
	return Pending
}

func (o *PendingMessage) Raw() *btp.RelayResult {
	panic("Unsupported")
}

func (o *PendingMessage) Transit() MessageTx {
	var err error
	pending := true
	attempt := int64(0)
	hash := o.tx.Hash()
	for pending && attempt < 5 {
		time.Sleep(time.Duration(attempt*2) * time.Second)
		_, pending, err = o.client.TransactionByHash(context.Background(), hash)
		if err == ethereum.NotFound {
			o.log.Warnf("message dropped - hash(%s)", hash.Hex())
			o.log.Debugf("MessageTransition(P->D) ID(%d) Tx(%s)", o.id, hash.Hex())
			return &DroppedMessage{
				id:   o.id,
				err:  errors.New("DroppedByTxPool"),
				prev: o.Type(),
			}
		} else if err != nil {
			o.log.Errorf("fail to query transaction - hash(%s) err(%s)", hash.Hex(), err.Error())
			o.log.Debugf("MessageTransition(P->F) ID(%d) Tx(%s)", o.id, hash.Hex())
			return &FaultedMessage{
				id:   o.id,
				err:  errors.New("TxQueryFailure"),
				prev: o.Type(),
			}
		}
		attempt++
	}

	if pending {
		o.log.Debugf("MessageTransition(P->F) ID(%d) Tx(%s)", o.id, hash.Hex())
		return &FaultedMessage{
			id:  o.id,
			err: errors.New(fmt.Sprintf("TxPoolTimeout:%s", hash.Hex())),
		}
	}

	receipt, err := o.client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		o.log.Errorf("fail to fetch receipt - hash(%s)", hash.Hex())
		o.log.Debugf("MessageTransition(P->F) ID(%d) Tx(%s)", o.id, hash.Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("ReceiptQueryFailure"),
			prev: o.Type(),
		}
	}

	o.log.Debugf("MessageTransition(P->E) ID(%d) Tx(%s)", o.id, hash.Hex())
	return &ExecutingMessage{
		PendingMessage: o,
		receipt:        receipt,
	}
}

type ExecutingMessage struct {
	*PendingMessage
	receipt *types.Receipt
}

func (o *ExecutingMessage) Type() MessageType {
	return Executing
}

func (o *ExecutingMessage) Transit() MessageTx {
	if o.receipt.Status == types.ReceiptStatusSuccessful {
		o.log.Debugf("MessageTransition(E->EE) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &ExecutedMessage{
			id:     o.id,
			number: o.receipt.BlockNumber.Uint64(),
			hash:   o.receipt.BlockHash,
			tx:     o.receipt.TxHash,
			err:    nil,
			log:    o.log,
		}
	}

	from, err := types.Sender(types.NewEIP155Signer(o.tx.ChainId()), o.tx)
	if err != nil {
		o.log.Debugf("MessageTransition(E->F) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("TxSenderRecoveryFailure"),
			prev: o.Type(),
		}
	}

	if _, err = o.client.CallContract(context.Background(), ethereum.CallMsg{
		From:     from,
		To:       o.tx.To(),
		Gas:      o.tx.Gas(),
		GasPrice: o.tx.GasPrice(),
		Value:    o.tx.Value(),
		Data:     o.tx.Data(),
	}, o.receipt.BlockNumber); err != nil {
		o.log.Debugf("MessageTransition(E->EE) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &ExecutedMessage{
			id:     o.id,
			number: o.receipt.BlockNumber.Uint64(),
			hash:   o.receipt.BlockHash,
			tx:     o.receipt.TxHash,
			err:    err,
			log:    o.log,
		}
	} else {
		o.log.Debugf("MessageTransition(E->F) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("EmulRevertFailure"),
			prev: o.Type(),
		}
	}
}

func (o *ExecutingMessage) Raw() *btp.RelayResult {
	panic("Unuspported")
}

func (o *ExecutingMessage) Status() uint64 {
	return o.receipt.Status
}

type ExecutedMessage struct {
	log    log.Logger
	id     string
	number uint64
	hash   common.Hash
	tx     common.Hash
	err    error
}

func (o *ExecutedMessage) Type() MessageType {
	return Executed
}

func (o *ExecutedMessage) Transit() MessageTx {
	o.log.Debugf("MessageTransition(EE->FN) ID(%d) Tx(%s)", o.id, o.tx.Hex())
	return &FinalizedMessage{
		o,
	}
}

func (o *ExecutedMessage) Raw() *btp.RelayResult {
	code := errs.SUCCESS
	if o.err != nil {
		code = errs.CodeOf(o.err)
	}
	return &btp.RelayResult{
		Id:        o.id,
		Err:       code,
		Finalized: false,
	}
}

type FinalizedMessage struct {
	*ExecutedMessage
}

func (o *FinalizedMessage) Type() MessageType {
	return Finalized
}

func (o *FinalizedMessage) Transit() MessageTx {
	return nil
}

func (o *FinalizedMessage) Raw() *btp.RelayResult {
	r := o.ExecutedMessage.Raw()
	r.Finalized = true
	return r
}

type DroppedMessage struct {
	id   string
	err  error
	prev MessageType
}

func (o *DroppedMessage) Type() MessageType {
	return Dropped
}

func (o *DroppedMessage) Transit() MessageTx {
	return nil
}

func (o *DroppedMessage) Raw() *btp.RelayResult {
	return &btp.RelayResult{
		Id:        o.id,
		Err:       errs.CodeOf(o.err),
		Finalized: false,
	}
}

type FaultedMessage struct {
	id   string
	err  error
	prev MessageType
}

func (o *FaultedMessage) Type() MessageType {
	return Faulted
}

func (o *FaultedMessage) Transit() MessageTx {
	return nil
}

func (o *FaultedMessage) Raw() *btp.RelayResult {
	return &btp.RelayResult{
		Id:        o.id,
		Err:       errs.CodeOf(o.err),
		Finalized: false,
	}
}
