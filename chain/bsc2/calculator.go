package bsc

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/icon-project/btp2/common/log"
)

type BlockFinalityCalculator struct {
	epoch      uint64
	feeds      []common.Hash
	checkpoint *Snapshot
	snapshots  *Snapshots
	log        log.Logger
}

func newBlockFinalityCalculator(epoch uint64, checkpoint *Snapshot, snapshots *Snapshots, log log.Logger) *BlockFinalityCalculator {
	return &BlockFinalityCalculator{
		epoch:      epoch,
		feeds:      make([]common.Hash, 0),
		snapshots:  snapshots,
		checkpoint: checkpoint,
		log:        log,
	}
}

func (o *BlockFinalityCalculator) ensureFeeds(snap *Snapshot) {
	var hash common.Hash
	if len(o.feeds) > 0 {
		hash = o.feeds[len(o.feeds)-1]
	} else {
		hash = o.checkpoint.Hash
	}

	feeds := make([]common.Hash, 0)
	var err error
	for !bytes.Equal(hash.Bytes(), snap.Hash.Bytes()) {
		feeds = append(feeds, snap.Hash)
		snap, err = o.snapshots.get(snap.ParentHash)
		if err != nil {
			o.log.Panicln(err.Error())
		}
	}

	for i := len(feeds) - 1; i >= 0; i-- {
		o.feeds = append(o.feeds, feeds[i])
	}
	o.log.Tracef("calc(%p) update initial feeds - feeds(%+v)", o.feeds)
}

// Ensure some signer has not been signed recently
// TODO make to safe
func (o *BlockFinalityCalculator) feed(feed common.Hash) ([]common.Hash, error) {
	o.feeds = append(o.feeds, feed)
	o.log.Tracef("calc(%p) checkpoint(%d:%s) feeds(%+v)", o, o.checkpoint.Number, o.checkpoint.Hash.Hex(), o.feeds)
	fnzs, err := o.calculate()
	if err != nil {
		o.log.Errorf("fail to calculate block finality - err(%s)\n", err.Error())
		return nil, err
	}
	o.log.Tracef("calc(%p) finalized blocks(%+v)", o, fnzs)
	if len(fnzs) > 0 {
		// set last finalized hash
		var err error
		o.checkpoint, err = o.snapshots.get(fnzs[len(fnzs)-1])
		if err != nil {
			o.log.Panicf("fail to retrieve finalized block snapshot - hash(%s) err(%s)\n", fnzs[0].Hex(), err.Error())
		}
		o.log.Tracef("calc(%p) new checkpoint(%d:%s)", o, o.checkpoint.Number, o.checkpoint.Hash.Hex())
		// dispose of finalized feeds
		o.feeds = o.feeds[len(fnzs):len(o.feeds)]
		o.log.Tracef("calc(%p) leftover feeds(%+v)", o, o.feeds)
	}
	return fnzs, nil
}

func numberOfAuthorized(authorities map[common.Address]struct{}, participants map[common.Address]struct{}) int {
	count := 0
	for participant, _ := range participants {
		if _, ok := authorities[participant]; ok {
			count++
		}
	}
	return count
}

// calculate finality of blocks
func (o *BlockFinalityCalculator) calculate() ([]common.Hash, error) {
	fnzs := make([]common.Hash, 0)
	sealers := make(map[common.Address]struct{})

	snap, err := o.snapshots.get(o.feeds[len(o.feeds)-1])
	if err != nil {
		return nil, err
	}

	o.log.Tracef("calc(%p) calculate between (%d:%s) ~ (%d:%s)", o,
		o.checkpoint.Number, o.checkpoint.Hash.Hex(), snap.Number, snap.Hash.Hex())
	for !bytes.Equal(snap.Hash.Bytes(), o.checkpoint.Hash.Bytes()) {
		o.log.Tracef("calc(%p) target(%d:%s) sealer(%s)", o, snap.Number, snap.Hash.Hex(), snap.Sealer)
		sealers[snap.Sealer] = struct{}{}
		o.log.Tracef("calc(%p) cans(%+v) vals(%+v) seals(%+v)", o, snap.Candidates, snap.Validators, sealers)
		if snap.Number%o.epoch == 0 {
			if numberOfAuthorized(snap.Validators, sealers) > len(snap.Validators)/2 &&
				numberOfAuthorized(snap.Candidates, sealers) > len(snap.Candidates)*2/3 {
				fnzs = append(fnzs, snap.Hash)
			} else {
				fnzs = fnzs[:0]
			}
		} else if len(fnzs) > 0 || numberOfAuthorized(snap.Candidates, sealers) > len(snap.Candidates)*2/3 {
			fnzs = append(fnzs, snap.Hash)
		}

		var err error
		snap, err = o.snapshots.get(snap.ParentHash)
		if err != nil {
			return nil, err
		}
	}
	// ascending by block height
	Reverse(fnzs)
	return fnzs, nil
}

func Reverse(s []common.Hash) {
	for i := 0; i < len(s)/2; i++ {
		t := s[i]
		s[i] = s[len(s)-1-i]
		s[len(s)-1-i] = t
	}
}
