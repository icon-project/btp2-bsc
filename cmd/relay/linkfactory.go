package main

import (
	"fmt"
	"path"
	"strconv"

	"github.com/icon-project/btp2/bsc/chain/bsc"
	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/chain/icon"
	"github.com/icon-project/btp2/chain/icon/btp2"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

func NewLink(cfg *Config, srcWallet wallet.Wallet, dstWallet wallet.Wallet, modLevels map[string]string) error {
	var err error
	linkErrCh := make(chan error)

	switch cfg.Direction {
	case FrontDirection:
		srcLog := setLogger(cfg, srcWallet, modLevels)
		srcLog.Debugln(cfg.FilePath, cfg.BaseDir)
		if cfg.BaseDir == "" {
			cfg.BaseDir = path.Join(".", ".btp2", cfg.Src.Address.NetworkAddress())
		}
		if _, err = newLink(cfg.Src.Address.BlockChain(), cfg.Config, srcLog, dstWallet, linkErrCh); err != nil {
			return err
		}

	case ReverseDirection:
		dstCfg := chain.Config{
			Src: cfg.Dst,
			Dst: cfg.Src,
		}

		dstLog := setLogger(cfg, dstWallet, modLevels)
		dstLog.Debugln(cfg.FilePath, cfg.BaseDir)
		if cfg.BaseDir == "" {
			cfg.BaseDir = path.Join(".", ".btp2", cfg.Dst.Address.NetworkAddress())
		}
		if _, err = newLink(cfg.Dst.Address.BlockChain(), dstCfg, dstLog, srcWallet, linkErrCh); err != nil {
			return err
		}
	case BothDirection:
		srcLog := setLogger(cfg, srcWallet, modLevels)
		if cfg.BaseDir == "" {
			cfg.BaseDir = path.Join(".", ".btp2", cfg.Src.Address.NetworkAddress())
		}
		if _, err = newLink(cfg.Src.Address.BlockChain(), cfg.Config, srcLog, dstWallet, linkErrCh); err != nil {
			return err
		}

		dstCfg := chain.Config{
			Src: cfg.Dst,
			Dst: cfg.Src,
		}

		dstLog := setLogger(cfg, dstWallet, modLevels)
		if cfg.BaseDir == "" {
			cfg.BaseDir = path.Join(".", ".btp2", cfg.Dst.Address.NetworkAddress())
		}
		if _, err = newLink(dstCfg.Src.Address.BlockChain(), dstCfg, dstLog, srcWallet, linkErrCh); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Not supported direction:%s", cfg.Direction)
	}

	for {
		select {
		case err := <-linkErrCh:
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newLink(s string, cfg chain.Config, l log.Logger, w wallet.Wallet, linkErrCh chan error) (types.Link, error) {
	var lk types.Link
	r := newReceiver(s, cfg, l)
	lk = link.NewLink(&cfg, r, l)

	go func() {
		err := lk.Start(newSender(cfg.Dst.Address.BlockChain(), cfg.Src, cfg.Dst, w, l))
		select {
		case linkErrCh <- err:
		default:
		}
	}()

	return lk, nil
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

func newReceiver(s string, cfg chain.Config, l log.Logger) link.Receiver {
	var receiver link.Receiver
	switch s {
	case BSC:
		fmt.Println("##")
		if val, ok := cfg.Src.Options["db_path"]; !ok {
			fmt.Println("==??")
			cfg.Src.Options["db_path"] = "./data"
		} else {
			fmt.Println("==>", val)
		}
		receiver = bsc.NewReceiver(bsc.RecvConfig{
			ChainID:     bsc.ChainID(cfg.Src.Endpoint),
			Epoch:       uint64(200),
			StartNumber: convToUint64(cfg.Src.Options, "start_number", 0),
			SrcAddress:  cfg.Src.Address,
			DstAddress:  cfg.Dst.Address,
			Endpoint:    cfg.Src.Endpoint,
			DBType:      fmt.Sprintf("%v", cfg.Src.Options["db_type"]),
			DBPath:      fmt.Sprintf("%v", cfg.Src.Options["db_path"]),
		}, l)
	case ICON:
		receiver = btp2.NewBTP2(cfg.Src.Address, cfg.Dst.Address, cfg.Src.Endpoint, l)
	default:
		l.Fatalf("Not supported receiver for chain:%s", s)
		return nil
	}
	return receiver
}

func newSender(s string, src chain.BaseConfig, dst chain.BaseConfig, w wallet.Wallet, l log.Logger) types.Sender {
	var sender types.Sender
	switch s {
	case BSC:
		sender = bsc.NewSender(bsc.SenderConfig{
			SrcAddress: src.Address,
			DstAddress: dst.Address,
			Endpoint:   dst.Endpoint,
			ChainID:    bsc.ChainID(dst.Endpoint),
			Epoch:      uint64(200),
		}, w, l)
	case ICON:
		sender = icon.NewSender(src.Address, dst.Address, w, dst.Endpoint, src.Options, l)
	default:
		l.Fatalf("Not supported sender for chain:%s", s)
		return nil
	}
	return sender
}
