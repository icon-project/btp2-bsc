package bsc2

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

const (
	TYPE  = "bsc-hertz"
	Epoch = 200
)

func RegisterBscHertz() {
	link.RegisterFactory(&link.Factory{
		Type:             TYPE,
		ParseChainConfig: ParseChainConfig,
		NewReceiver:      NewReceiver,
		NewSender:        NewSender,
	})
}

func ParseChainConfig(raw json.RawMessage) (link.ChainConfig, error) {
	cfg := chain.BaseConfig{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if cfg.Type != TYPE {
		return nil, fmt.Errorf("invalid type (type:%s)", cfg.Type)
	}
	return cfg, nil
}

func NewReceiver(srcCfg link.ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (link.Receiver, error) {
	src := srcCfg.(chain.BaseConfig)

	return newReceiver(RecvConfig{
		ChainID:     ChainID(src.Endpoint),
		Epoch:       uint64(Epoch),
		StartNumber: convToUint64(src.Options, "start_number", 0),
		SrcAddress:  src.Address,
		DstAddress:  dstAddr,
		Endpoint:    src.Endpoint,
		DBType:      fmt.Sprintf("%v", src.Options["db_type"]),
		DBPath:      fmt.Sprintf("%v", src.Options["db_path"]),
	}, l), nil
}

func NewSender(srcAddr types.BtpAddress, dstCfg link.ChainConfig, baseDir string, l log.Logger) (types.Sender, error) {
	dst := dstCfg.(chain.BaseConfig)

	w, err := newWallet(dst.KeyStorePass, dst.KeySecret, dst.KeyStore)
	if err != nil {
		return nil, err
	}

	return newSender(SenderConfig{
		SrcAddress: srcAddr,
		DstAddress: dst.Address,
		Endpoint:   dst.Endpoint,
		ChainID:    ChainID(dst.Endpoint),
		Epoch:      uint64(Epoch)}, w, l), nil
}

func newWallet(passwd, secret string, keyStorePath string) (types.Wallet, error) {
	if keyStore, err := os.ReadFile(keyStorePath); err != nil {
		return nil, fmt.Errorf("fail to open KeyStore file path=%s", keyStorePath)
	} else {
		pw, err := resolvePassword(secret, passwd)
		if err != nil {
			return nil, err
		}
		return wallet.DecryptKeyStore(keyStore, pw)
	}
}

func resolvePassword(keySecret, keyStorePass string) ([]byte, error) {
	if keySecret != "" {
		return os.ReadFile(keySecret)
	} else {
		if keyStorePass != "" {
			return []byte(keyStorePass), nil
		}
	}
	return nil, nil
}

func convToUint64(m map[string]interface{}, k string, def uint64) uint64 {
	if val, ok := m[k]; !ok {
		return def
	} else {
		if val, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err != nil {
			panic(err)
		} else {
			return val
		}
	}
}
