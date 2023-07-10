package bsc

import (
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

type MessageTransactor struct {
	snapshots  *Snapshots
	replies    chan<- *btp.RelayResult
	finalities chan common.Hash
	transition chan MessageTx
	mu         sync.Mutex
	waitings   []MessageTx
	log        log.Logger
}

func newMessageTransactor(snapshots *Snapshots, log log.Logger) *MessageTransactor {
	return &MessageTransactor{
		snapshots:  snapshots,
		log:        log,
		finalities: make(chan common.Hash),
		transition: make(chan MessageTx, 128),
		waitings:   make([]MessageTx, 0),
	}
}

func (o *MessageTransactor) Busy() bool {
	// return len(o.transition) == cap(o.transition) || len(o.waitings) == 1
	return len(o.transition) == cap(o.transition)
}

func (o *MessageTransactor) NotifyFinality(finality common.Hash) {
	o.finalities <- finality
}

// C -> D
// C -> F
// C -> P -> D
// C -> P -> F
// C -> P -> E -> D
// C -> P -> E -> F
// C -> P -> E -> W -> nil
func (o *MessageTransactor) Run(replies chan<- *btp.RelayResult) {
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
				o.mu.Lock()
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
				o.mu.Unlock()
			}()
		}
	}
}

func (o *MessageTransactor) Send(msg MessageTx) {
	typ := msg.Type()
	if typ == Faulted || typ == Dropped || typ == Executed || typ == Finalized {
		o.replies <- msg.Raw()
	}

	if typ == Created || typ == Pending || typ == Executing {
		o.transition <- msg
	}

	if impl, ok := msg.(*ExecutedMessage); ok {
		if impl.err == nil {
			o.mu.Lock()
			o.waitings = append(o.waitings, msg)
			o.mu.Unlock()
		}
	}
}

func (o *MessageTransactor) tryFinalizeInLock(finality common.Hash, msg MessageTx) MessageTx {
	exec, ok := msg.(*ExecutedMessage)
	if !ok {
		o.log.Panicf("ForbiddenMessage(%s)", msg.Type())
	}

	snap, err := o.snapshots.get(finality)
	if err != nil {
		o.log.Panicln(err.Error())
	}

	if exec.number > snap.Number {
		return exec
	}
	for exec.number < snap.Number {
		if snap, err = o.snapshots.get(snap.ParentHash); err != nil {
			o.log.Panicln(err.Error())
		}
	}
	if exec.hash != snap.Hash {
		o.log.Debugf("MessageTransition(EE->D)")
		return &DroppedMessage{
			id:   exec.id,
			err:  errors.New("TxDropped"),
			prev: exec.Type(),
			log:  o.log,
		}
	}
	return exec.Transit()
}

type MessageTx interface {
	Type() MessageType
	Transit() MessageTx
	Raw() *btp.RelayResult
}

func newMessageTx(id int, from string, client *Client, opts *bind.TransactOpts, blob []byte, log log.Logger) MessageTx {
	log.Infof("NewMessage - ID(%d) FROM(%s)", id, from)
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
	id     int
	from   string
	client *Client
	opts   *bind.TransactOpts
	blob   []byte
}

func (o *CreatedMessage) Type() MessageType {
	return Created
}

func (o *CreatedMessage) Transit() MessageTx {
	if o.opts.GasLimit != uint64(0) {
		o.log.Panicf("TODO Supports CustomGasLimit")
	}
	for ErrCounter := 0; ; ErrCounter++ {
		tx, err := o.client.BTPMessageCenter.HandleRelayMessage(o.opts, o.from, o.blob)
		if err != nil {
			if ErrCounter >= 3 {
				o.log.Warnf("MessageTransition(C->D) ID(%d) Blob(%s) Err(%s)",
					o.id, hex.EncodeToString(o.blob), err.Error())
				return &DroppedMessage{
					id:  o.id,
					err: err,
					log: o.log,
				}
			}
			o.log.Debugf("Retry to estimate gas limit - attempt(%d) err(%s)", ErrCounter, err.Error())
			time.Sleep(time.Second * 2)
			continue
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
}

func (o *CreatedMessage) Raw() *btp.RelayResult {
	o.log.Panicln("Unsupported")
	return nil
}

type PendingMessage struct {
	log    log.Logger
	id     int
	tx     *types.Transaction
	client *Client
}

func (o *PendingMessage) Type() MessageType {
	return Pending
}

func (o *PendingMessage) Raw() *btp.RelayResult {
	o.log.Panicln("Unsupported")
	return nil
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
			o.log.Warnf("MessageTransition(P->D) ID(%d) Tx(%s)", o.id, hash.Hex())
			return &DroppedMessage{
				id:   o.id,
				err:  errors.New("DroppedByTxPool"),
				prev: o.Type(),
				log:  o.log,
			}
		} else if err != nil {
			o.log.Warnf("MessageTransition(P->F) ID(%d) Tx(%s)", o.id, hash.Hex(), err.Error())
			return &FaultedMessage{
				id:   o.id,
				err:  errors.New("TxQueryFailure"),
				prev: o.Type(),
				log:  o.log,
			}
		}
		attempt++
	}

	if pending {
		o.log.Warnf("MessageTransition(P->F) ID(%d) Tx(%s) Err(Timeout)", o.id, hash.Hex())
		return &FaultedMessage{
			id:  o.id,
			err: errors.New(fmt.Sprintf("TxPoolTimeout:%s", hash.Hex())),
			log: o.log,
		}
	}

	receipt, err := o.client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		o.log.Warnf("MessageTransition(P->F) ID(%d) Tx(%s) Err(NoReceipt)", o.id, hash.Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("ReceiptQueryFailure"),
			prev: o.Type(),
			log:  o.log,
		}
	}

	o.log.Infof("MessageTransition(P->E) ID(%d) Tx(%s)", o.id, hash.Hex())
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
		o.log.Infof("MessageTransition(E->EE) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
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
		o.log.Errorf("MessageTransition(E->F) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("TxSenderRecoveryFailure"),
			prev: o.Type(),
			log:  o.log,
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
		o.log.Infof("MessageTransition(E->EE) ID(%d) Tx(%s)", o.id, o.tx.Hash().Hex())
		return &ExecutedMessage{
			id:     o.id,
			number: o.receipt.BlockNumber.Uint64(),
			hash:   o.receipt.BlockHash,
			tx:     o.receipt.TxHash,
			err:    err,
			log:    o.log,
		}
	} else {
		o.log.Warnf("MessageTransition(E->F) ID(%d) Tx(%s) Err(EmulRevertFailure)", o.id, o.tx.Hash().Hex())
		return &FaultedMessage{
			id:   o.id,
			err:  errors.New("EmulRevertFailure"),
			prev: o.Type(),
			log:  o.log,
		}
	}
}

func (o *ExecutingMessage) Raw() *btp.RelayResult {
	o.log.Panicln("Unuspported")
	return nil
}

func (o *ExecutingMessage) Status() uint64 {
	return o.receipt.Status
}

type ExecutedMessage struct {
	log    log.Logger
	id     int
	number uint64
	hash   common.Hash
	tx     common.Hash
	err    error
}

func (o *ExecutedMessage) Type() MessageType {
	return Executed
}

func (o *ExecutedMessage) Transit() MessageTx {
	o.log.Infof("MessageTransition(EE->FN) ID(%d) Tx(%s)", o.id, o.tx.Hex())
	return &FinalizedMessage{o}
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
	o.log.Debugf("MessageTransition(FN->Nil)")
	return nil
}

func (o *FinalizedMessage) Raw() *btp.RelayResult {
	r := o.ExecutedMessage.Raw()
	r.Finalized = true
	return r
}

type DroppedMessage struct {
	id   int
	err  error
	prev MessageType
	log  log.Logger
}

func (o *DroppedMessage) Type() MessageType {
	return Dropped
}

func (o *DroppedMessage) Transit() MessageTx {
	o.log.Debugf("MessageTransition(D->Nil)")
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
	id   int
	err  error
	prev MessageType
	log  log.Logger
}

func (o *FaultedMessage) Type() MessageType {
	return Faulted
}

func (o *FaultedMessage) Transit() MessageTx {
	o.log.Debugf("MessageTransition(F->Nil)")
	return nil
}

func (o *FaultedMessage) Raw() *btp.RelayResult {
	return &btp.RelayResult{
		Id:        o.id,
		Err:       errs.CodeOf(o.err),
		Finalized: false,
	}
}
