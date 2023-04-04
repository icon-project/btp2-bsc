package bsc

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/icon-project/btp2/common/log"
)

type Feed interface {
	ID() common.Hash
}

type BlockFinalityCalculator struct {
	checkpoint *Snapshot
	sealers    map[common.Address]struct{}
	feeds      []common.Hash
	snaps      *Snapshots
	epoch      uint64
	log        log.Logger
}

func newBlockFinalityCalculator(epoch uint64, checkpoint *Snapshot, snaps *Snapshots, log log.Logger) *BlockFinalityCalculator {
	return &BlockFinalityCalculator{
		snaps:      snaps,
		checkpoint: checkpoint,
		feeds:      make([]common.Hash, 0),
		sealers:    make(map[common.Address]struct{}),
		epoch:      epoch,
	}
}

// Ensure some signer has not been signed recently
// TODO make to safe
func (o *BlockFinalityCalculator) feed(feeds ...common.Hash) ([]common.Hash, error) {
	o.feeds = append(o.feeds, feeds...)
	if len(o.feeds) <= 0 {
		return make([]common.Hash, 0), nil
	}

	leaf := o.feeds[len(o.feeds)-1]
	root := o.checkpoint.Hash
	fnzs, err := CalcBlockFinality(o.epoch, o.snaps, o.sealers, leaf, root)
	if err != nil {
		o.log.Errorf("fail to calculate block finality - err(%s)\n", err.Error())
		return nil, err
	}

	if len(fnzs) > 0 {
		// set last finalized hash
		var err error
		o.checkpoint, err = o.snaps.get(fnzs[0])
		if err != nil {
			panic(fmt.Sprintf("fail to retrieve finalized block snapshot - hash(%s) err(%s)\n", fnzs[0].Hex(), err.Error()))
		}
		// dbg
		//o.log.Debugln("fnzs:", len(fnzs))
		//o.log.Debugln("feeds[...]:", o.feeds[0:len(fnzs)])
		// dispose of finalized feeds
		o.feeds = o.feeds[len(fnzs):len(o.feeds)]

		// dispose of accumulated seal info
		for _, v := range fnzs {
			if snap, err := o.snaps.get(v); err != nil {
				panic(fmt.Sprintf("fail to retrieve snapshot - hash(%s) err(%s)\n", v.Hex(), err.Error()))
			} else {
				delete(o.sealers, snap.Sealer)
			}
		}
	}
	return fnzs, nil
}

func CalcBlockFinality(epoch uint64, snaps *Snapshots, sealers map[common.Address]struct{}, leaf, root common.Hash) ([]common.Hash, error) {
	fnzs := make([]common.Hash, 0)
	if sealers == nil {
		sealers = make(map[common.Address]struct{})
	}
	snap, err := snaps.get(leaf)
	if err != nil {
		return nil, err
	}

	for !bytes.Equal(snap.Hash.Bytes(), root.Bytes()) {
		sealers[snap.Sealer] = struct{}{}
		if snap.Number%epoch == 0 {
			nseal := 0
			for k, _ := range snap.Candidates {
				if _, ok := sealers[k]; ok {
					nseal++
				}
			}

			if len(sealers) > len(snap.Validators)/2 && nseal > len(snap.Candidates)*2/3 {
				fnzs = append(fnzs, snap.Hash)
			} else {
				fnzs = fnzs[:0]
			}
		} else if len(fnzs) > 0 {
			fnzs = append(fnzs, snap.Hash)
		} else {
			nseal := 0
			for k, _ := range snap.Candidates {
				if _, ok := sealers[k]; ok {
					nseal++
				}
			}
			if nseal > len(snap.Candidates)*2/3 {
				fnzs = append(fnzs, snap.Hash)
			}
		}

		var err error
		snap, err = snaps.get(snap.ParentHash)
		if err != nil {
			return nil, err
		}
	}
	// ascending by block height
	for i := 0; i < len(fnzs)/2; i++ {
		swap := fnzs[i]
		fnzs[i] = fnzs[len(fnzs)-1]
		fnzs[len(fnzs)-1] = swap
	}
	return fnzs, nil
}
