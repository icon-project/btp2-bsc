package bsc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
	"github.com/icon-project/btp2/common/db"
)

const (
	Checkpoint = 1024
)

type Snapshots struct {
	chainId  *big.Int
	database db.Database
	cache    *lru.ARCCache
	client   *ethclient.Client
}

func newSnapshots(chainId *big.Int, client *ethclient.Client, cacheSize int, database db.Database) *Snapshots {
	snaps := &Snapshots{
		chainId:  chainId,
		client:   client,
		database: database,
	}

	if cache, err := lru.NewARC(cacheSize); err != nil {
		panic(err)
	} else {
		snaps.cache = cache
	}
	return snaps
}

func (o *Snapshots) has(id common.Hash) bool {
	if bk, err := o.database.GetBucket("Snapshot"); err != nil {
		panic(err)
	} else {
		return bk.Has(append([]byte("snap-"), id[:]...))
	}
}

func (o *Snapshots) get(id common.Hash) (*Snapshot, error) {
	// on cache memory
	if snap, ok := o.cache.Get(id); ok {
		return snap.(*Snapshot), nil
	}

	// on persistent memory
	if ok, err := hasSnapshot(o.database, id); err != nil {
		return nil, err
	} else if ok {
		if snap, err := loadSnapshot(o.database, id); err != nil {
			return nil, err
		} else {
			return snap, nil
		}
	}

	// on network
	if err := o.ensure(id); err != nil {
		return nil, err
	}

	snap, _ := o.cache.Get(id)
	return snap.(*Snapshot), nil
}

func (o *Snapshots) add(snap *Snapshot) error {
	o.cache.Add(snap.Hash, snap)
	if snap.Number%Checkpoint == 0 {
		if err := snap.store(o.database); err != nil {
			return err
		}
	}
	return nil
}

func (o *Snapshots) ensure(id common.Hash) error {
	if _, ok := o.cache.Get(id); ok {
		return nil
	}

	heads := make([]*types.Header, 0)
	var snap *Snapshot
	for snap == nil {
		if ok, err := hasSnapshot(o.database, id); err != nil {
			return err
		} else if ok {
			snap, err = loadSnapshot(o.database, id)
			if err != nil {
				return err
			} else {
				break
			}
		}

		head, err := o.client.HeaderByHash(context.Background(), id)
		if err != nil {
			return err
		}

		heads = append(heads, head)
		id = head.ParentHash
	}

	for i := range heads {
		var err error
		snap, err = snap.apply(heads[len(heads)-1-i], o.chainId)
		if err != nil {
			return err
		}
		if err := o.add(snap); err != nil {
			return err
		}
	}
	return nil
}
