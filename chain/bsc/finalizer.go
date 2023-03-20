package bsc

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BlockFinalizer struct {
	checkpoint *Snapshot
	signers    map[common.Address]struct{}
	feeds      []Feed
	snaps      *Snapshots
	epoch      *big.Int
}

func newBlockFinalizer(epoch *big.Int, checkpoint *Snapshot, snaps *Snapshots) *BlockFinalizer {
	return &BlockFinalizer{
		snaps:      snaps,
		checkpoint: checkpoint,
		feeds:      make([]Feed, 0),
		signers:    make(map[common.Address]struct{}),
		epoch:      epoch,
	}
}

type Feed interface {
	ID() common.Hash
}

// Ensure some signer has not been signed recently
func (o *BlockFinalizer) feed(feeds ...Feed) ([]Feed, error) {
	o.feeds = append(o.feeds, feeds...)

	// if o.checkpoint == nil {
	// 	if snap, err := o.snaps.get(o.feeds[0].ID()); err != nil {
	// 		return nil, err
	// 	} else {
	// 		o.checkpoint = snap
	// 		o.signers = make(map[common.Address]struct{})
	// 	}
	// }

	snap, err := o.snaps.get(o.feeds[len(o.feeds)-1].ID())
	if err != nil {
		return nil, err
	}

	fins := make([]common.Hash, 0)
	newCheckpoint := snap
	for !bytes.Equal(snap.Hash.Bytes(), o.checkpoint.Hash.Bytes()) {
		o.signers[snap.Sealer] = struct{}{}
		if snap.Number%o.epoch.Uint64() == 0 {
			nsigners := 0
			for key, _ := range snap.Candidates {
				if _, ok := o.signers[key]; ok {
					nsigners++
				}
			}
			if len(o.signers) > len(snap.Validators)/2 && nsigners > len(snap.Candidates)*2/3 {
				fins = append(fins, snap.Hash)
			} else {
				fins = fins[:0]
			}
		} else if len(fins) > 0 {
			fins = append(fins, snap.Hash)
		} else {
			nsigners := 0
			for key, _ := range snap.Candidates {
				if _, ok := o.signers[key]; ok {
					nsigners++
				}
			}
			if nsigners > len(snap.Candidates)*2/3 {
				fins = append(fins, snap.Hash)
			}
		}
		snap, _ = o.snaps.get(snap.ParentHash)
	}

	// dispose of old signers
	for _, hash := range fins {
		if snap, err := o.snaps.get(hash); err != nil {
			return nil, err
		} else {
			delete(o.signers, snap.Sealer)
		}
	}
	o.checkpoint = newCheckpoint
	ret := o.feeds[0:len(fins)]
	o.feeds = o.feeds[len(fins):len(o.feeds)]
	return ret, nil
}

func finalize(epoch uint64, snapshots *Snapshots, from, until common.Hash) ([]common.Hash, error) {
	// TODO test clone attacks
	snap, err := snapshots.get(from)
	if err != nil {
		return nil, err
	}

	cnfs := make([]common.Hash, 0)
	signers := make(map[common.Address]struct{})
	// TODO extract duplicates
	for !bytes.Equal(snap.Hash.Bytes(), until.Bytes()) {
		signers[snap.Sealer] = struct{}{}
		if snap.Number%epoch == 0 {
			nsigners := 0
			for k, _ := range snap.Candidates {
				if _, ok := signers[k]; ok {
					nsigners++
				}
			}
			if len(signers) > len(snap.Validators)/2 && nsigners > len(snap.Candidates)*2/3 {
				cnfs = append(cnfs, snap.Hash)
			} else {
				cnfs = cnfs[:0]
			}
		} else if len(cnfs) > 0 {
			cnfs = append(cnfs, snap.Hash)
		} else {
			nsigners := 0
			for k, _ := range snap.Candidates {
				if _, ok := signers[k]; ok {
					nsigners++
				}
			}
			if nsigners > len(snap.Candidates)*2/3 {
				cnfs = append(cnfs, snap.Hash)
			}
		}
		snap, _ = snapshots.get(snap.ParentHash)
	}
	return cnfs, nil
}
