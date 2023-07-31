// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bsc2

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

// IBMVVerifierStatus is an auto generated low-level Go binding around an user-defined struct.
type IBMVVerifierStatus struct {
	Height *big.Int
	Extra  []byte
}

// TypesBTPMessage is an auto generated low-level Go binding around an user-defined struct.
type TypesBTPMessage struct {
	Src     string
	Dst     string
	Svc     string
	Sn      *big.Int
	Message []byte
	Nsn     *big.Int
	FeeInfo TypesFeeInfo
}

// TypesFeeInfo is an auto generated low-level Go binding around an user-defined struct.
type TypesFeeInfo struct {
	Network string
	Values  []*big.Int
}

// TypesLinkStatus is an auto generated low-level Go binding around an user-defined struct.
type TypesLinkStatus struct {
	RxSeq         *big.Int
	TxSeq         *big.Int
	Verifier      IBMVVerifierStatus
	CurrentHeight *big.Int
}

// BTPMessageCenterMetaData contains all meta data concerning the BTPMessageCenter contract.
var BTPMessageCenterMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"_src\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"int256\",\"name\":\"_nsn\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"_next\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"_event\",\"type\":\"string\"}],\"name\":\"BTPEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"_receiver\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"_nsn\",\"type\":\"int256\"}],\"name\":\"ClaimReward\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"_nsn\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_result\",\"type\":\"uint256\"}],\"name\":\"ClaimRewardResult\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"_next\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_seq\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"}],\"name\":\"Message\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"_prev\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_seq\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_ecode\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"_emsg\",\"type\":\"string\"}],\"name\":\"MessageDropped\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_bmcManagementAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_bmcServiceAddr\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBtpAddress\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNetworkAddress\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_prev\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"}],\"name\":\"handleRelayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_to\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_svc\",\"type\":\"string\"},{\"internalType\":\"int256\",\"name\":\"_sn\",\"type\":\"int256\"},{\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"}],\"name\":\"sendMessage\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNetworkSn\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_to\",\"type\":\"string\"},{\"internalType\":\"bool\",\"name\":\"_response\",\"type\":\"bool\"}],\"name\":\"getFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_link\",\"type\":\"string\"}],\"name\":\"getStatus\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"rxSeq\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"txSeq\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extra\",\"type\":\"bytes\"}],\"internalType\":\"structIBMV.VerifierStatus\",\"name\":\"verifier\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"currentHeight\",\"type\":\"uint256\"}],\"internalType\":\"structTypes.LinkStatus\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_next\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"_msg\",\"type\":\"bytes\"}],\"name\":\"sendInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_prev\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_seq\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"src\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"dst\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"svc\",\"type\":\"string\"},{\"internalType\":\"int256\",\"name\":\"sn\",\"type\":\"int256\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"internalType\":\"int256\",\"name\":\"nsn\",\"type\":\"int256\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"network\",\"type\":\"string\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"internalType\":\"structTypes.FeeInfo\",\"name\":\"feeInfo\",\"type\":\"tuple\"}],\"internalType\":\"structTypes.BTPMessage\",\"name\":\"_msg\",\"type\":\"tuple\"}],\"name\":\"dropMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_link\",\"type\":\"string\"}],\"name\":\"clearSeq\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_addr\",\"type\":\"address\"}],\"name\":\"getReward\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"internalType\":\"int256\",\"name\":\"_nsn\",\"type\":\"int256\"},{\"internalType\":\"uint256\",\"name\":\"_result\",\"type\":\"uint256\"}],\"name\":\"emitClaimRewardResult\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_network\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_receiver\",\"type\":\"string\"}],\"name\":\"claimReward\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
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

// GetBtpAddress is a free data retrieval call binding the contract method 0x4f63a21d.
//
// Solidity: function getBtpAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetBtpAddress(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getBtpAddress")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetBtpAddress is a free data retrieval call binding the contract method 0x4f63a21d.
//
// Solidity: function getBtpAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterSession) GetBtpAddress() (string, error) {
	return _BTPMessageCenter.Contract.GetBtpAddress(&_BTPMessageCenter.CallOpts)
}

// GetBtpAddress is a free data retrieval call binding the contract method 0x4f63a21d.
//
// Solidity: function getBtpAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetBtpAddress() (string, error) {
	return _BTPMessageCenter.Contract.GetBtpAddress(&_BTPMessageCenter.CallOpts)
}

// GetFee is a free data retrieval call binding the contract method 0x7d4c4f4a.
//
// Solidity: function getFee(string _to, bool _response) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetFee(opts *bind.CallOpts, _to string, _response bool) (*big.Int, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getFee", _to, _response)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFee is a free data retrieval call binding the contract method 0x7d4c4f4a.
//
// Solidity: function getFee(string _to, bool _response) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterSession) GetFee(_to string, _response bool) (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetFee(&_BTPMessageCenter.CallOpts, _to, _response)
}

// GetFee is a free data retrieval call binding the contract method 0x7d4c4f4a.
//
// Solidity: function getFee(string _to, bool _response) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetFee(_to string, _response bool) (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetFee(&_BTPMessageCenter.CallOpts, _to, _response)
}

// GetNetworkAddress is a free data retrieval call binding the contract method 0x6bf459cb.
//
// Solidity: function getNetworkAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetNetworkAddress(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getNetworkAddress")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetNetworkAddress is a free data retrieval call binding the contract method 0x6bf459cb.
//
// Solidity: function getNetworkAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterSession) GetNetworkAddress() (string, error) {
	return _BTPMessageCenter.Contract.GetNetworkAddress(&_BTPMessageCenter.CallOpts)
}

// GetNetworkAddress is a free data retrieval call binding the contract method 0x6bf459cb.
//
// Solidity: function getNetworkAddress() view returns(string)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetNetworkAddress() (string, error) {
	return _BTPMessageCenter.Contract.GetNetworkAddress(&_BTPMessageCenter.CallOpts)
}

// GetNetworkSn is a free data retrieval call binding the contract method 0x676286ba.
//
// Solidity: function getNetworkSn() view returns(int256)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetNetworkSn(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getNetworkSn")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNetworkSn is a free data retrieval call binding the contract method 0x676286ba.
//
// Solidity: function getNetworkSn() view returns(int256)
func (_BTPMessageCenter *BTPMessageCenterSession) GetNetworkSn() (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetNetworkSn(&_BTPMessageCenter.CallOpts)
}

// GetNetworkSn is a free data retrieval call binding the contract method 0x676286ba.
//
// Solidity: function getNetworkSn() view returns(int256)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetNetworkSn() (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetNetworkSn(&_BTPMessageCenter.CallOpts)
}

// GetReward is a free data retrieval call binding the contract method 0x64cd3750.
//
// Solidity: function getReward(string _network, address _addr) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterCaller) GetReward(opts *bind.CallOpts, _network string, _addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getReward", _network, _addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetReward is a free data retrieval call binding the contract method 0x64cd3750.
//
// Solidity: function getReward(string _network, address _addr) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterSession) GetReward(_network string, _addr common.Address) (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetReward(&_BTPMessageCenter.CallOpts, _network, _addr)
}

// GetReward is a free data retrieval call binding the contract method 0x64cd3750.
//
// Solidity: function getReward(string _network, address _addr) view returns(uint256)
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetReward(_network string, _addr common.Address) (*big.Int, error) {
	return _BTPMessageCenter.Contract.GetReward(&_BTPMessageCenter.CallOpts, _network, _addr)
}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256))
func (_BTPMessageCenter *BTPMessageCenterCaller) GetStatus(opts *bind.CallOpts, _link string) (TypesLinkStatus, error) {
	var out []interface{}
	err := _BTPMessageCenter.contract.Call(opts, &out, "getStatus", _link)

	if err != nil {
		return *new(TypesLinkStatus), err
	}

	out0 := *abi.ConvertType(out[0], new(TypesLinkStatus)).(*TypesLinkStatus)

	return out0, err

}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256))
func (_BTPMessageCenter *BTPMessageCenterSession) GetStatus(_link string) (TypesLinkStatus, error) {
	return _BTPMessageCenter.Contract.GetStatus(&_BTPMessageCenter.CallOpts, _link)
}

// GetStatus is a free data retrieval call binding the contract method 0x22b05ed2.
//
// Solidity: function getStatus(string _link) view returns((uint256,uint256,(uint256,bytes),uint256))
func (_BTPMessageCenter *BTPMessageCenterCallerSession) GetStatus(_link string) (TypesLinkStatus, error) {
	return _BTPMessageCenter.Contract.GetStatus(&_BTPMessageCenter.CallOpts, _link)
}

// ClaimReward is a paid mutator transaction binding the contract method 0xcb7477b5.
//
// Solidity: function claimReward(string _network, string _receiver) payable returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) ClaimReward(opts *bind.TransactOpts, _network string, _receiver string) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "claimReward", _network, _receiver)
}

// ClaimReward is a paid mutator transaction binding the contract method 0xcb7477b5.
//
// Solidity: function claimReward(string _network, string _receiver) payable returns()
func (_BTPMessageCenter *BTPMessageCenterSession) ClaimReward(_network string, _receiver string) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.ClaimReward(&_BTPMessageCenter.TransactOpts, _network, _receiver)
}

// ClaimReward is a paid mutator transaction binding the contract method 0xcb7477b5.
//
// Solidity: function claimReward(string _network, string _receiver) payable returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) ClaimReward(_network string, _receiver string) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.ClaimReward(&_BTPMessageCenter.TransactOpts, _network, _receiver)
}

// ClearSeq is a paid mutator transaction binding the contract method 0x019d2420.
//
// Solidity: function clearSeq(string _link) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) ClearSeq(opts *bind.TransactOpts, _link string) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "clearSeq", _link)
}

// ClearSeq is a paid mutator transaction binding the contract method 0x019d2420.
//
// Solidity: function clearSeq(string _link) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) ClearSeq(_link string) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.ClearSeq(&_BTPMessageCenter.TransactOpts, _link)
}

// ClearSeq is a paid mutator transaction binding the contract method 0x019d2420.
//
// Solidity: function clearSeq(string _link) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) ClearSeq(_link string) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.ClearSeq(&_BTPMessageCenter.TransactOpts, _link)
}

// DropMessage is a paid mutator transaction binding the contract method 0x39ff9dc1.
//
// Solidity: function dropMessage(string _prev, uint256 _seq, (string,string,string,int256,bytes,int256,(string,uint256[])) _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) DropMessage(opts *bind.TransactOpts, _prev string, _seq *big.Int, _msg TypesBTPMessage) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "dropMessage", _prev, _seq, _msg)
}

// DropMessage is a paid mutator transaction binding the contract method 0x39ff9dc1.
//
// Solidity: function dropMessage(string _prev, uint256 _seq, (string,string,string,int256,bytes,int256,(string,uint256[])) _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) DropMessage(_prev string, _seq *big.Int, _msg TypesBTPMessage) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.DropMessage(&_BTPMessageCenter.TransactOpts, _prev, _seq, _msg)
}

// DropMessage is a paid mutator transaction binding the contract method 0x39ff9dc1.
//
// Solidity: function dropMessage(string _prev, uint256 _seq, (string,string,string,int256,bytes,int256,(string,uint256[])) _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) DropMessage(_prev string, _seq *big.Int, _msg TypesBTPMessage) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.DropMessage(&_BTPMessageCenter.TransactOpts, _prev, _seq, _msg)
}

// EmitClaimRewardResult is a paid mutator transaction binding the contract method 0x947dfac2.
//
// Solidity: function emitClaimRewardResult(address _sender, string _network, int256 _nsn, uint256 _result) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) EmitClaimRewardResult(opts *bind.TransactOpts, _sender common.Address, _network string, _nsn *big.Int, _result *big.Int) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "emitClaimRewardResult", _sender, _network, _nsn, _result)
}

// EmitClaimRewardResult is a paid mutator transaction binding the contract method 0x947dfac2.
//
// Solidity: function emitClaimRewardResult(address _sender, string _network, int256 _nsn, uint256 _result) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) EmitClaimRewardResult(_sender common.Address, _network string, _nsn *big.Int, _result *big.Int) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.EmitClaimRewardResult(&_BTPMessageCenter.TransactOpts, _sender, _network, _nsn, _result)
}

// EmitClaimRewardResult is a paid mutator transaction binding the contract method 0x947dfac2.
//
// Solidity: function emitClaimRewardResult(address _sender, string _network, int256 _nsn, uint256 _result) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) EmitClaimRewardResult(_sender common.Address, _network string, _nsn *big.Int, _result *big.Int) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.EmitClaimRewardResult(&_BTPMessageCenter.TransactOpts, _sender, _network, _nsn, _result)
}

// HandleRelayMessage is a paid mutator transaction binding the contract method 0x21b1e9bb.
//
// Solidity: function handleRelayMessage(string _prev, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) HandleRelayMessage(opts *bind.TransactOpts, _prev string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "handleRelayMessage", _prev, _msg)
}

// HandleRelayMessage is a paid mutator transaction binding the contract method 0x21b1e9bb.
//
// Solidity: function handleRelayMessage(string _prev, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) HandleRelayMessage(_prev string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.HandleRelayMessage(&_BTPMessageCenter.TransactOpts, _prev, _msg)
}

// HandleRelayMessage is a paid mutator transaction binding the contract method 0x21b1e9bb.
//
// Solidity: function handleRelayMessage(string _prev, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) HandleRelayMessage(_prev string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.HandleRelayMessage(&_BTPMessageCenter.TransactOpts, _prev, _msg)
}

// Initialize is a paid mutator transaction binding the contract method 0x463fd1af.
//
// Solidity: function initialize(string _network, address _bmcManagementAddr, address _bmcServiceAddr) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) Initialize(opts *bind.TransactOpts, _network string, _bmcManagementAddr common.Address, _bmcServiceAddr common.Address) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "initialize", _network, _bmcManagementAddr, _bmcServiceAddr)
}

// Initialize is a paid mutator transaction binding the contract method 0x463fd1af.
//
// Solidity: function initialize(string _network, address _bmcManagementAddr, address _bmcServiceAddr) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) Initialize(_network string, _bmcManagementAddr common.Address, _bmcServiceAddr common.Address) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.Initialize(&_BTPMessageCenter.TransactOpts, _network, _bmcManagementAddr, _bmcServiceAddr)
}

// Initialize is a paid mutator transaction binding the contract method 0x463fd1af.
//
// Solidity: function initialize(string _network, address _bmcManagementAddr, address _bmcServiceAddr) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) Initialize(_network string, _bmcManagementAddr common.Address, _bmcServiceAddr common.Address) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.Initialize(&_BTPMessageCenter.TransactOpts, _network, _bmcManagementAddr, _bmcServiceAddr)
}

// SendInternal is a paid mutator transaction binding the contract method 0xdf9ccb10.
//
// Solidity: function sendInternal(string _next, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) SendInternal(opts *bind.TransactOpts, _next string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "sendInternal", _next, _msg)
}

// SendInternal is a paid mutator transaction binding the contract method 0xdf9ccb10.
//
// Solidity: function sendInternal(string _next, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterSession) SendInternal(_next string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendInternal(&_BTPMessageCenter.TransactOpts, _next, _msg)
}

// SendInternal is a paid mutator transaction binding the contract method 0xdf9ccb10.
//
// Solidity: function sendInternal(string _next, bytes _msg) returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) SendInternal(_next string, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendInternal(&_BTPMessageCenter.TransactOpts, _next, _msg)
}

// SendMessage is a paid mutator transaction binding the contract method 0x522a901e.
//
// Solidity: function sendMessage(string _to, string _svc, int256 _sn, bytes _msg) payable returns(int256)
func (_BTPMessageCenter *BTPMessageCenterTransactor) SendMessage(opts *bind.TransactOpts, _to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.Transact(opts, "sendMessage", _to, _svc, _sn, _msg)
}

// SendMessage is a paid mutator transaction binding the contract method 0x522a901e.
//
// Solidity: function sendMessage(string _to, string _svc, int256 _sn, bytes _msg) payable returns(int256)
func (_BTPMessageCenter *BTPMessageCenterSession) SendMessage(_to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendMessage(&_BTPMessageCenter.TransactOpts, _to, _svc, _sn, _msg)
}

// SendMessage is a paid mutator transaction binding the contract method 0x522a901e.
//
// Solidity: function sendMessage(string _to, string _svc, int256 _sn, bytes _msg) payable returns(int256)
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) SendMessage(_to string, _svc string, _sn *big.Int, _msg []byte) (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.SendMessage(&_BTPMessageCenter.TransactOpts, _to, _svc, _sn, _msg)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BTPMessageCenter *BTPMessageCenterTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BTPMessageCenter.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BTPMessageCenter *BTPMessageCenterSession) Receive() (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.Receive(&_BTPMessageCenter.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BTPMessageCenter *BTPMessageCenterTransactorSession) Receive() (*types.Transaction, error) {
	return _BTPMessageCenter.Contract.Receive(&_BTPMessageCenter.TransactOpts)
}

// BTPMessageCenterBTPEventIterator is returned from FilterBTPEvent and is used to iterate over the raw logs and unpacked data for BTPEvent events raised by the BTPMessageCenter contract.
type BTPMessageCenterBTPEventIterator struct {
	Event *BTPMessageCenterBTPEvent // Event containing the contract specifics and raw log

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
func (it *BTPMessageCenterBTPEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterBTPEvent)
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
		it.Event = new(BTPMessageCenterBTPEvent)
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
func (it *BTPMessageCenterBTPEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterBTPEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterBTPEvent represents a BTPEvent event raised by the BTPMessageCenter contract.
type BTPMessageCenterBTPEvent struct {
	Src   common.Hash
	Nsn   *big.Int
	Next  string
	Event string
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterBTPEvent is a free log retrieval operation binding the contract event 0x51f135d1c44e53689ca91af3b1bce4918d2b590d92bb76a854ab30e7de741828.
//
// Solidity: event BTPEvent(string indexed _src, int256 indexed _nsn, string _next, string _event)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterBTPEvent(opts *bind.FilterOpts, _src []string, _nsn []*big.Int) (*BTPMessageCenterBTPEventIterator, error) {

	var _srcRule []interface{}
	for _, _srcItem := range _src {
		_srcRule = append(_srcRule, _srcItem)
	}
	var _nsnRule []interface{}
	for _, _nsnItem := range _nsn {
		_nsnRule = append(_nsnRule, _nsnItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "BTPEvent", _srcRule, _nsnRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterBTPEventIterator{contract: _BTPMessageCenter.contract, event: "BTPEvent", logs: logs, sub: sub}, nil
}

// WatchBTPEvent is a free log subscription operation binding the contract event 0x51f135d1c44e53689ca91af3b1bce4918d2b590d92bb76a854ab30e7de741828.
//
// Solidity: event BTPEvent(string indexed _src, int256 indexed _nsn, string _next, string _event)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchBTPEvent(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterBTPEvent, _src []string, _nsn []*big.Int) (event.Subscription, error) {

	var _srcRule []interface{}
	for _, _srcItem := range _src {
		_srcRule = append(_srcRule, _srcItem)
	}
	var _nsnRule []interface{}
	for _, _nsnItem := range _nsn {
		_nsnRule = append(_nsnRule, _nsnItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "BTPEvent", _srcRule, _nsnRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterBTPEvent)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "BTPEvent", log); err != nil {
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

// ParseBTPEvent is a log parse operation binding the contract event 0x51f135d1c44e53689ca91af3b1bce4918d2b590d92bb76a854ab30e7de741828.
//
// Solidity: event BTPEvent(string indexed _src, int256 indexed _nsn, string _next, string _event)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseBTPEvent(log types.Log) (*BTPMessageCenterBTPEvent, error) {
	event := new(BTPMessageCenterBTPEvent)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "BTPEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BTPMessageCenterClaimRewardIterator is returned from FilterClaimReward and is used to iterate over the raw logs and unpacked data for ClaimReward events raised by the BTPMessageCenter contract.
type BTPMessageCenterClaimRewardIterator struct {
	Event *BTPMessageCenterClaimReward // Event containing the contract specifics and raw log

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
func (it *BTPMessageCenterClaimRewardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterClaimReward)
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
		it.Event = new(BTPMessageCenterClaimReward)
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
func (it *BTPMessageCenterClaimRewardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterClaimRewardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterClaimReward represents a ClaimReward event raised by the BTPMessageCenter contract.
type BTPMessageCenterClaimReward struct {
	Sender   common.Address
	Network  common.Hash
	Receiver string
	Amount   *big.Int
	Nsn      *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterClaimReward is a free log retrieval operation binding the contract event 0x92a1bf3ab6e4839bb58db5f99ae98936355e94c1223779807563d1a08a3bf449.
//
// Solidity: event ClaimReward(address indexed _sender, string indexed _network, string _receiver, uint256 _amount, int256 _nsn)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterClaimReward(opts *bind.FilterOpts, _sender []common.Address, _network []string) (*BTPMessageCenterClaimRewardIterator, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}
	var _networkRule []interface{}
	for _, _networkItem := range _network {
		_networkRule = append(_networkRule, _networkItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "ClaimReward", _senderRule, _networkRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterClaimRewardIterator{contract: _BTPMessageCenter.contract, event: "ClaimReward", logs: logs, sub: sub}, nil
}

// WatchClaimReward is a free log subscription operation binding the contract event 0x92a1bf3ab6e4839bb58db5f99ae98936355e94c1223779807563d1a08a3bf449.
//
// Solidity: event ClaimReward(address indexed _sender, string indexed _network, string _receiver, uint256 _amount, int256 _nsn)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchClaimReward(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterClaimReward, _sender []common.Address, _network []string) (event.Subscription, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}
	var _networkRule []interface{}
	for _, _networkItem := range _network {
		_networkRule = append(_networkRule, _networkItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "ClaimReward", _senderRule, _networkRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterClaimReward)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "ClaimReward", log); err != nil {
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

// ParseClaimReward is a log parse operation binding the contract event 0x92a1bf3ab6e4839bb58db5f99ae98936355e94c1223779807563d1a08a3bf449.
//
// Solidity: event ClaimReward(address indexed _sender, string indexed _network, string _receiver, uint256 _amount, int256 _nsn)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseClaimReward(log types.Log) (*BTPMessageCenterClaimReward, error) {
	event := new(BTPMessageCenterClaimReward)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "ClaimReward", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BTPMessageCenterClaimRewardResultIterator is returned from FilterClaimRewardResult and is used to iterate over the raw logs and unpacked data for ClaimRewardResult events raised by the BTPMessageCenter contract.
type BTPMessageCenterClaimRewardResultIterator struct {
	Event *BTPMessageCenterClaimRewardResult // Event containing the contract specifics and raw log

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
func (it *BTPMessageCenterClaimRewardResultIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterClaimRewardResult)
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
		it.Event = new(BTPMessageCenterClaimRewardResult)
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
func (it *BTPMessageCenterClaimRewardResultIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterClaimRewardResultIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterClaimRewardResult represents a ClaimRewardResult event raised by the BTPMessageCenter contract.
type BTPMessageCenterClaimRewardResult struct {
	Sender  common.Address
	Network common.Hash
	Nsn     *big.Int
	Result  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterClaimRewardResult is a free log retrieval operation binding the contract event 0xe4a536fde1966a83411167748abc5fbd80e0247bffe941ca4ddb5079fd730636.
//
// Solidity: event ClaimRewardResult(address indexed _sender, string indexed _network, int256 _nsn, uint256 _result)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterClaimRewardResult(opts *bind.FilterOpts, _sender []common.Address, _network []string) (*BTPMessageCenterClaimRewardResultIterator, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}
	var _networkRule []interface{}
	for _, _networkItem := range _network {
		_networkRule = append(_networkRule, _networkItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "ClaimRewardResult", _senderRule, _networkRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterClaimRewardResultIterator{contract: _BTPMessageCenter.contract, event: "ClaimRewardResult", logs: logs, sub: sub}, nil
}

// WatchClaimRewardResult is a free log subscription operation binding the contract event 0xe4a536fde1966a83411167748abc5fbd80e0247bffe941ca4ddb5079fd730636.
//
// Solidity: event ClaimRewardResult(address indexed _sender, string indexed _network, int256 _nsn, uint256 _result)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchClaimRewardResult(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterClaimRewardResult, _sender []common.Address, _network []string) (event.Subscription, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}
	var _networkRule []interface{}
	for _, _networkItem := range _network {
		_networkRule = append(_networkRule, _networkItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "ClaimRewardResult", _senderRule, _networkRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterClaimRewardResult)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "ClaimRewardResult", log); err != nil {
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

// ParseClaimRewardResult is a log parse operation binding the contract event 0xe4a536fde1966a83411167748abc5fbd80e0247bffe941ca4ddb5079fd730636.
//
// Solidity: event ClaimRewardResult(address indexed _sender, string indexed _network, int256 _nsn, uint256 _result)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseClaimRewardResult(log types.Log) (*BTPMessageCenterClaimRewardResult, error) {
	event := new(BTPMessageCenterClaimRewardResult)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "ClaimRewardResult", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BTPMessageCenterInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the BTPMessageCenter contract.
type BTPMessageCenterInitializedIterator struct {
	Event *BTPMessageCenterInitialized // Event containing the contract specifics and raw log

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
func (it *BTPMessageCenterInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterInitialized)
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
		it.Event = new(BTPMessageCenterInitialized)
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
func (it *BTPMessageCenterInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterInitialized represents a Initialized event raised by the BTPMessageCenter contract.
type BTPMessageCenterInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterInitialized(opts *bind.FilterOpts) (*BTPMessageCenterInitializedIterator, error) {

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterInitializedIterator{contract: _BTPMessageCenter.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterInitialized) (event.Subscription, error) {

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterInitialized)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseInitialized(log types.Log) (*BTPMessageCenterInitialized, error) {
	event := new(BTPMessageCenterInitialized)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
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
	Next common.Hash
	Seq  *big.Int
	Msg  []byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterMessage is a free log retrieval operation binding the contract event 0x37be353f216cf7e33639101fd610c542e6a0c0109173fa1c1d8b04d34edb7c1b.
//
// Solidity: event Message(string indexed _next, uint256 indexed _seq, bytes _msg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterMessage(opts *bind.FilterOpts, _next []string, _seq []*big.Int) (*BTPMessageCenterMessageIterator, error) {

	var _nextRule []interface{}
	for _, _nextItem := range _next {
		_nextRule = append(_nextRule, _nextItem)
	}
	var _seqRule []interface{}
	for _, _seqItem := range _seq {
		_seqRule = append(_seqRule, _seqItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "Message", _nextRule, _seqRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterMessageIterator{contract: _BTPMessageCenter.contract, event: "Message", logs: logs, sub: sub}, nil
}

// WatchMessage is a free log subscription operation binding the contract event 0x37be353f216cf7e33639101fd610c542e6a0c0109173fa1c1d8b04d34edb7c1b.
//
// Solidity: event Message(string indexed _next, uint256 indexed _seq, bytes _msg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchMessage(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterMessage, _next []string, _seq []*big.Int) (event.Subscription, error) {

	var _nextRule []interface{}
	for _, _nextItem := range _next {
		_nextRule = append(_nextRule, _nextItem)
	}
	var _seqRule []interface{}
	for _, _seqItem := range _seq {
		_seqRule = append(_seqRule, _seqItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "Message", _nextRule, _seqRule)
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
// Solidity: event Message(string indexed _next, uint256 indexed _seq, bytes _msg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseMessage(log types.Log) (*BTPMessageCenterMessage, error) {
	event := new(BTPMessageCenterMessage)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "Message", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BTPMessageCenterMessageDroppedIterator is returned from FilterMessageDropped and is used to iterate over the raw logs and unpacked data for MessageDropped events raised by the BTPMessageCenter contract.
type BTPMessageCenterMessageDroppedIterator struct {
	Event *BTPMessageCenterMessageDropped // Event containing the contract specifics and raw log

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
func (it *BTPMessageCenterMessageDroppedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BTPMessageCenterMessageDropped)
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
		it.Event = new(BTPMessageCenterMessageDropped)
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
func (it *BTPMessageCenterMessageDroppedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BTPMessageCenterMessageDroppedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BTPMessageCenterMessageDropped represents a MessageDropped event raised by the BTPMessageCenter contract.
type BTPMessageCenterMessageDropped struct {
	Prev  common.Hash
	Seq   *big.Int
	Msg   []byte
	Ecode *big.Int
	Emsg  string
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterMessageDropped is a free log retrieval operation binding the contract event 0x35348898083dc8da7eb4e604320d7cc250732cc1f584d2596f670b5d6bab8de5.
//
// Solidity: event MessageDropped(string indexed _prev, uint256 indexed _seq, bytes _msg, uint256 _ecode, string _emsg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) FilterMessageDropped(opts *bind.FilterOpts, _prev []string, _seq []*big.Int) (*BTPMessageCenterMessageDroppedIterator, error) {

	var _prevRule []interface{}
	for _, _prevItem := range _prev {
		_prevRule = append(_prevRule, _prevItem)
	}
	var _seqRule []interface{}
	for _, _seqItem := range _seq {
		_seqRule = append(_seqRule, _seqItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.FilterLogs(opts, "MessageDropped", _prevRule, _seqRule)
	if err != nil {
		return nil, err
	}
	return &BTPMessageCenterMessageDroppedIterator{contract: _BTPMessageCenter.contract, event: "MessageDropped", logs: logs, sub: sub}, nil
}

// WatchMessageDropped is a free log subscription operation binding the contract event 0x35348898083dc8da7eb4e604320d7cc250732cc1f584d2596f670b5d6bab8de5.
//
// Solidity: event MessageDropped(string indexed _prev, uint256 indexed _seq, bytes _msg, uint256 _ecode, string _emsg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) WatchMessageDropped(opts *bind.WatchOpts, sink chan<- *BTPMessageCenterMessageDropped, _prev []string, _seq []*big.Int) (event.Subscription, error) {

	var _prevRule []interface{}
	for _, _prevItem := range _prev {
		_prevRule = append(_prevRule, _prevItem)
	}
	var _seqRule []interface{}
	for _, _seqItem := range _seq {
		_seqRule = append(_seqRule, _seqItem)
	}

	logs, sub, err := _BTPMessageCenter.contract.WatchLogs(opts, "MessageDropped", _prevRule, _seqRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BTPMessageCenterMessageDropped)
				if err := _BTPMessageCenter.contract.UnpackLog(event, "MessageDropped", log); err != nil {
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

// ParseMessageDropped is a log parse operation binding the contract event 0x35348898083dc8da7eb4e604320d7cc250732cc1f584d2596f670b5d6bab8de5.
//
// Solidity: event MessageDropped(string indexed _prev, uint256 indexed _seq, bytes _msg, uint256 _ecode, string _emsg)
func (_BTPMessageCenter *BTPMessageCenterFilterer) ParseMessageDropped(log types.Log) (*BTPMessageCenterMessageDropped, error) {
	event := new(BTPMessageCenterMessageDropped)
	if err := _BTPMessageCenter.contract.UnpackLog(event, "MessageDropped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
