package bsc2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/icon-project/btp2/common/log"
)

type BlockFinalityCalculator struct {
	checkpoint common.Hash
	feeds      []common.Hash
	snaps      *Snapshots
	log        log.Logger
}

func newBlockFinalityCalculator(checkpoint common.Hash, feeds []common.Hash, snaps *Snapshots, log log.Logger) *BlockFinalityCalculator {
	return &BlockFinalityCalculator{
		checkpoint: checkpoint,
		feeds:      feeds,
		snaps:      snaps,
		log:        log,
	}
}

func (o *BlockFinalityCalculator) feed(feed common.Hash) ([]common.Hash, error) {
	o.feeds = append(o.feeds, feed)
	finalities := make([]common.Hash, 0)
	if finality, err := o.calculate(); err != nil {
		o.log.Errorf("fail to calculate block finality - err(%s)", err)
		return nil, err
	} else {
		for i := 0; i < len(o.feeds); i++ {
			if o.feeds[i] == finality {
				finalities = o.feeds[0 : i+1]
				o.feeds = o.feeds[i+1 : len(o.feeds)]
				break
			}
		}

		o.checkpoint = finality
	}
	return finalities, nil
}

func (o *BlockFinalityCalculator) calculate() (common.Hash, error) {
	snap, err := o.snaps.get(o.feeds[len(o.feeds)-1])
	if err != nil {
		return common.Hash{}, err
	}

	checkpoint, err := o.snaps.get(o.checkpoint)
	if err != nil {
		return common.Hash{}, err
	}

	finality := checkpoint.Hash
	for !bytes.Equal(snap.Hash.Bytes(), checkpoint.Hash.Bytes()) {
		if snap.Attestation.SourceNumber < checkpoint.Number {
			break
		}

		if snap.Attestation.TargetNumber == snap.Attestation.SourceNumber+1 {
			finality = snap.Attestation.SourceHash
			break
		}

		snap, err = o.snaps.get(snap.ParentHash)
		if err != nil {
			return common.Hash{}, err
		}
	}
	return finality, nil
}
