package bsc

import (
	"io"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/icon-project/btp2/common/link"
	btp "github.com/icon-project/btp2/common/types"
)

type BSCRelayMessage struct {
	TypePrefixedMessages []BSCTypePrefixedMessage
}

type BSCTypePrefixedMessage struct {
	Type    uint
	Payload []byte
}

// Implement BlockUpdate
type BlockUpdate struct {
	heads  []*types.Header
	start  uint64
	height uint64
	status *VerifierStatus
}

func (o *BlockUpdate) Headers() []*types.Header {
	return o.heads
}

func (o *BlockUpdate) Type() link.MessageItemType {
	return link.TypeBlockUpdate
}

func (o *BlockUpdate) Len() int64 {
	size := int64(0)
	for _, head := range o.heads {
		size += int64(math.Ceil(float64(head.Size())))
	}
	return size
}

func (o *BlockUpdate) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	blob, err := rlp.EncodeToBytes(o.status)
	if err != nil {
		return err
	}
	status.Verifier.Height = int64(o.height)
	status.Verifier.Extra = blob
	return nil
}

func (o *BlockUpdate) ProofHeight() int64 {
	return int64(o.height)
}

func (o *BlockUpdate) SrcHeight() int64 {
	return int64(o.start)
}

func (o *BlockUpdate) TargetHeight() int64 {
	return int64(o.height)
}

func (o *BlockUpdate) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, o.heads)
}

type extbu struct {
	Headers []*types.Header
}

func (o *BlockUpdate) DecodeRLP(s *rlp.Stream) error {
	bu := &extbu{}
	if blob, err := s.Bytes(); err != nil {
		return err
	} else {
		rlp.DecodeBytes(blob, bu)
	}
	o.heads = bu.Headers
	return nil
}

// Implement BlockProof
type BlockProof struct {
	Header    *types.Header
	AccHeight uint64
	Witness   [][]byte
}

func (o *BlockProof) Type() link.MessageItemType {
	return link.TypeBlockProof
}

func (o *BlockProof) Len() int64 {
	return int64(0)
}

func (o *BlockProof) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	return nil
}

func (o *BlockProof) ProofHeight() int64 {
	return int64(0)
}

// Implement MessageProof
type MessageProof struct {
	Hash     common.Hash
	Proofs   []BSCReceiptProof
	sequence uint64
}

func (o *MessageProof) Type() link.MessageItemType {
	return link.TypeMessageProof
}

func (o *MessageProof) Len() int64 {
	size := int64(0)
	for _, rp := range o.Proofs {
		for _, part := range rp.Proof {
			size += int64(len(part))
		}
	}
	return size
}

func (o *MessageProof) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	status.RxSeq += int64(len(o.Proofs))
	return nil
}

func (o *MessageProof) StartSeqNum() int64 {
	return int64(o.sequence)
}

func (o *MessageProof) LastSeqNum() int64 {
	return int64(o.sequence + uint64(len(o.Proofs)) - 1)
}

type BSCReceiptProof struct {
	Key   []byte
	Proof [][]byte
}
