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
	height uint64
	status *VerifierStatus
}

func (o BlockUpdate) Type() link.MessageItemType {
	return link.TypeBlockUpdate
}

func (o BlockUpdate) Len() int64 {
	size := int64(0)
	for _, head := range o.heads {
		size += int64(math.Ceil(float64(head.Size())))
	}
	return size
}

func (o BlockUpdate) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	blob, err := rlp.EncodeToBytes(o.status)
	if err != nil {
		return err
	}
	status.Verifier.Height = int64(o.height)
	status.Verifier.Extra = blob
	return nil
}

func (o BlockUpdate) ProofHeight() int64 {
	return 0
}

func (o BlockUpdate) SrcHeight() int64 {
	if len(o.heads) <= 0 {
		return -1
	} else {
		return o.heads[0].Number.Int64()
	}
}

func (o BlockUpdate) TargetHeight() int64 {
	if len(o.heads) <= 0 {
		return -1
	} else {
		return o.heads[len(o.heads)-1].Number.Int64()
	}
}

func (o BlockUpdate) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, o.heads)
}

// Implement BlockProof
type BSCBlockProof struct {
	Header    *types.Header
	AccHeight uint64
	Witness   [][]byte
}

func (o BSCBlockProof) Type() link.MessageItemType {
	return link.TypeBlockProof
}

func (o BSCBlockProof) Len() int64 {
	return int64(0)
}

func (o BSCBlockProof) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	return nil
}

func (o BSCBlockProof) ProofHeight() int64 {
	return int64(0)
}

// Implement MessageProof
type BSCMessageProof struct {
	Hash   common.Hash
	Proofs []BSCReceiptProof
}

func (o BSCMessageProof) Type() link.MessageItemType {
	return link.TypeMessageProof
}

func (o BSCMessageProof) Len() int64 {
	return 0
}

func (o BSCMessageProof) UpdateBMCLinkStatus(status *btp.BMCLinkStatus) error {
	status.RxSeq += int64(len(o.Proofs))
	return nil
}

func (o BSCMessageProof) StartSeqNum() int64 {
	return 0
}

func (o BSCMessageProof) LastSeqNum() int64 {
	return 0
}

type BSCReceiptProof struct {
	Key   []byte
	Proof [][]byte
}
