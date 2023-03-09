// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bsc

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// TypesLinkStats is an auto generated low-level Go binding around an user-defined struct.
type TypesLinkStats struct {
	RxSeq         *big.Int
	TxSeq         *big.Int
	Verifier      TypesVerifierStats
	CurrentHeight *big.Int
}

// TypesVerifierStats is an auto generated low-level Go binding around an user-defined struct.
type TypesVerifierStats struct {
	Height *big.Int
	Extra  []byte
}

// BTPMessageCenterMetaData contains all meta data concerning the BTPMessageCenter contract.
var BTPMessageCenterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"to\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"sequence\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"Message\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_to\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_svc\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_sn\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_link\",\"type\":\"string\"}],\"name\":\"getStatus\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"rxSeq\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"txSeq\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extra\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.VerifierStats\",\"name\":\"verifier\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"currentHeight\",\"type\":\"uint256\"}],\"internalType\":\"structTypes.LinkStats\",\"name\":\"_linkStats\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BTPMessageCenterABI is the input ABI used to generate the binding from.
// Deprecated: Use BTPMessageCenterMetaData.ABI instead.
var BTPMessageCenterABI = BTPMessageCenterMetaData.ABI

// BTPMessageCenter is an auto generated Go binding around an Ethereum contract.
type BTPMessageCenter struct {
	BTPMessageCenterCaller     // Read-only binding to the contract
	BTPMessageCenterTransactor // Write-only binding to the contract
	BTPMessageCenterFilterer   // Log filterer for contract events
}

// BTPMessageCenterCaller is an auto generated read-only Go binding around an Ethereum contract.
type BTPMessageCenterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BTPMessageCenterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BTPMessageCenterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BTPMessageCenterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BTPMessageCenterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BTPMessageCenterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BTPMessageCenterSession struct {
	Contract     *BTPMessageCenter // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BTPMessageCenterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BTPMessageCenterCallerSession struct {
	Contract *BTPMessageCenterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// BTPMessageCenterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BTPMessageCenterTransactorSession struct {
	Contract     *BTPMessageCenterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// BTPMessageCenterRaw is an auto generated low-level Go binding around an Ethereum contract.
type BTPMessageCenterRaw struct {
	Contract *BTPMessageCenter // Generic contract binding to access the raw methods on
}

// BTPMessageCenterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BTPMessageCenterCallerRaw struct {
	Contract *BTPMessageCenterCaller // Generic read-only contract binding to access the raw methods on
}

// BTPMessageCenterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BTPMessageCenterTransactorRaw struct {
	Contract *BTPMessageCenterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBTPMessageCenter creates a new instance of BTPMessageCenter, bound to a specific deployed contract.
func NewBTPMessageCenter(address common.Address, backend bind.ContractBackend) (*BTPMessageCenter, error) {
	contract, err := bindBTPMessageCenter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenter{BTPMessageCenterCaller: BTPMessageCenterCaller{contract: contract}, BTPMessageCenterTransactor: BTPMessageCenterTransactor{contract: contract}, BTPMessageCenterFilterer: BTPMessageCenterFilterer{contract: contract}}, nil
}

// NewBTPMessageCenterCaller creates a new read-only instance of BTPMessageCenter, bound to a specific deployed contract.
func NewBTPMessageCenterCaller(address common.Address, caller bind.ContractCaller) (*BTPMessageCenterCaller, error) {
	contract, err := bindBTPMessageCenter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterCaller{contract: contract}, nil
}

// NewBTPMessageCenterTransactor creates a new write-only instance of BTPMessageCenter, bound to a specific deployed contract.
func NewBTPMessageCenterTransactor(address common.Address, transactor bind.ContractTransactor) (*BTPMessageCenterTransactor, error) {
	contract, err := bindBTPMessageCenter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterTransactor{contract: contract}, nil
}

// NewBTPMessageCenterFilterer creates a new log filterer instance of BTPMessageCenter, bound to a specific deployed contract.
func NewBTPMessageCenterFilterer(address common.Address, filterer bind.ContractFilterer) (*BTPMessageCenterFilterer, error) {
	contract, err := bindBTPMessageCenter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterFilterer{contract: contract}, nil
}

// bindBTPMessageCenter binds a generic wrapper to an already deployed contract.
func bindBTPMessageCenter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BTPMessageCenterABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BTPMessageCenter *BTPMessageCenterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BTPMessageCenter.Contract.BTPMessageCenterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BTPMessageCenter *BTPMessageCenterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.BTPMessageCenterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BTPMessageCenter *BTPMessageCenterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.BTPMessageCenterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BTPMessageCenter *BTPMessageCenterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BTPMessageCenter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BTPMessageCenter *BTPMessageCenterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BTPMessageCenter *BTPMessageCenterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.contract.Transact(opts, method, params...)
}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256) _linkStats)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetStatus(opts *bind.CallOpts, _link string) (TypesLinkStats, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getStatus", _link)

	if err != nil {
		return *new(TypesLinkStats), err
	}

	out0 := *abi.ConvertType(out[0], new(TypesLinkStats)).(*TypesLinkStats)

	return out0, err

}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256) _linkStats)
func (_BTPMessageCenter *BTPMessageCenterSession) GetStatus(_link string) (TypesLinkStats, error) {
	return _BTPMessageCenter.Contract.GetStatus(&_BTPMessageCenter.CallOpts, _link)
}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256) _linkStats)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetStatus(_link string) (TypesLinkStats, error) {
	return _BTPMessageCenter.Contract.GetStatus(&_BTPMessageCenter.CallOpts, _link)
}

// SendMessage is a paid mutator transaction binding the contract method 0xbf6c1d9a.
//
// Solidity: function sendMessage(string _to, string _svc, uint256 _sn, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) SendMessage(opts *bind.TransactOpts, _to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "sendMessage", _to, _svc, _sn, _msg)
}

// SendMessage is a paid mutator transaction binding the contract method 0xbf6c1d9a.
//
// Solidity: function sendMessage(string _to, string _svc, uint256 _sn, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) SendMessage(_to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendMessage(&_BTPMessageCenter.TransactOpts, _to, _svc, _sn, _msg)
}

// SendMessage is a paid mutator transaction binding the contract method 0xbf6c1d9a.
//
// Solidity: function sendMessage(string _to, string _svc, uint256 _sn, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) SendMessage(_to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendMessage(&_BTPMessageCenter.TransactOpts, _to, _svc, _sn, _msg)
}

// BTPMessageCenterMessageIterator is returned from FilterMessage and is used to iterate over the raw logs and unpacked data for Message events raised by the BTPMessageCenter contract.
type BTPMessageCenterMessageIterator struct {
	Event *BTPMessageCenterMessage // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BTPMessageCenterMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterMessage)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BTPMessageCenterMessage)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BTPMessageCenterMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterMessage represents a Message event raised by the BTPMessageCenter contract.
type BTPMessageCenterMessage struct {
	To       common.Hash
	Sequence *big.Int
	Message  []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterMessage is a free log retrieval operation binding the contract event 0x37be353f216cf7e33639101fd610c542e6a0c0109173fa1c1d8b04d34edb7c1b.
//
// Solidity: event Message(string indexed to, uint256 indexed sequence, bytes message)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterMessage(opts *bind.FilterOpts, to []string, sequence []*big.Int) (*BTPMessageCenterMessageIterator, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var sequenceRule []interface{}
	for _, sequenceItem := range sequence {
		sequenceRule = append(sequenceRule, sequenceItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "Message", toRule, sequenceRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterMessageIterator{contract: _BTPMessageCenter.contract, event: "Message", logs: logs, sub: sub}, nil
}

// WatchMessage is a free log subscription operation binding the contract event 0x37be353f216cf7e33639101fd610c542e6a0c0109173fa1c1d8b04d34edb7c1b.
//
// Solidity: event Message(string indexed to, uint256 indexed sequence, bytes message)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchMessage(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterMessage, to []string, sequence []*big.Int) (event.Subscription, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var sequenceRule []interface{}
	for _, sequenceItem := range sequence {
		sequenceRule = append(sequenceRule, sequenceItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "Message", toRule, sequenceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterMessage)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "Message", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMessage is a log parse operation binding the contract event 0x37be353f216cf7e33639101fd610c542e6a0c0109173fa1c1d8b04d34edb7c1b.
//
// Solidity: event Message(string indexed to, uint256 indexed sequence, bytes message)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseMessage(log types.Log) (*BTPMessageCenterMessage, error) {
	event := new(BTPMessageCenterMessage)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "Message", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
