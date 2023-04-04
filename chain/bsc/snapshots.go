package bsc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
	"github.com/icon-project/btp2/common/db"
	"github.com/icon-project/btp2/common/log"
)

const (
	SnapCheckPoint = 1024
)

type Snapshots struct {
	chainId  *big.Int
	database db.Database
	cache    *lru.ARCCache
	client   *ethclient.Client
	log      log.Logger
}

func newSnapshots(chainId *big.Int, client *ethclient.Client, cacheSize int, database db.Database, log log.Logger) *Snapshots {
	snaps := &Snapshots{
		chainId:  chainId,
		client:   client,
		database: database,
		log:      log,
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
		s := snap.(*Snapshot)
		// o.log.Debugf("load snapshot on cache - number(%d) id(%s)\n", s.Number, s.Hash.Hex())
		return s, nil
	}

	// on database
	if ok, err := hasSnapshot(o.database, id); err != nil {
		return nil, err
	} else if ok {
		if snap, err := loadSnapshot(o.database, id); err != nil {
			return nil, err
		} else {
			// o.log.Debugf("load snapshot on database - number(%d) id(%s)\n", snap.Number, snap.Hash.Hex())
			return snap, nil
		}
	}

	// on network
	if err := o.ensure(id); err != nil {
		return nil, err
	}

	if snap, ok := o.cache.Get(id); !ok {
		panic("fail to fetching snapshots")
	} else {
		s := snap.(*Snapshot)
		// o.log.Debugf("load snapshot on network - number(%d) id(%s)\n", s.Number, s.Hash.Hex())
		return s, nil
	}
}

func (o *Snapshots) add(snap *Snapshot) error {
	o.cache.Add(snap.Hash, snap)
	if o.database != nil && snap.Number%SnapCheckPoint == 0 {
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
		if ok := o.cache.Contains(id); ok {
			s, _ := o.cache.Get(id)
			snap = s.(*Snapshot)
			break
		}

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

	if len(heads) > 0 {
		o.log.Debugf("preload snapshots with heads on network: %d ~ %d\n", heads[0].Number.Uint64(), heads[len(heads)-1].Number.Uint64())
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
