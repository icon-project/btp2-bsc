package bsc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"

	"github.com/icon-project/btp2/common/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/icon-project/btp2/common/db"
	"golang.org/x/crypto/sha3"
)

const (
	validatorBytesLength = common.AddressLength + types.BLSPublicKeyLength
	extraVanity          = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal            = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
	epoch                = 200
	validatorNumberSize  = 1 // Fixed number of extra prefix bytes reserved for validator number after Luban
)

// validatorsAscending implements the sort interface to allow sorting a list of addresses
type validatorsAscending []common.Address

func (s validatorsAscending) Len() int           { return len(s) }
func (s validatorsAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s validatorsAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type Snapshot struct {
	// config // epoch

	Number      uint64                      `json:"number"`
	Hash        common.Hash                 `json:"hash"`
	ParentHash  common.Hash                 `json:"parent_hash"`
	Sealer      common.Address              `json:"sealer"`
	Validators  map[common.Address]struct{} `json:"validators"`
	Candidates  map[common.Address]struct{} `json:"candidates"`
	Recents     map[uint64]common.Address   `json:"recents"`
	Attestation *types.VoteData             `json:"attestation"`
	log         log.Logger
}

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent validators, so only ever use it for
// the genesis block.
func newSnapshot(
	number uint64,
	hash common.Hash,
	validators []common.Address,
	candidates []common.Address,
	recents []common.Address,
	attestation *types.VoteData,
	sealer common.Address,
	parentHash common.Hash,
	log log.Logger,
) *Snapshot {
	snap := &Snapshot{
		Number:     number,
		Hash:       hash,
		ParentHash: parentHash,
		Validators: make(map[common.Address]struct{}),
		Candidates: make(map[common.Address]struct{}),
		Recents:    make(map[uint64]common.Address),
		Sealer:     sealer,
		log:        log,
	}
	for _, v := range validators {
		snap.Validators[v] = struct{}{}
	}
	for _, v := range candidates {
		snap.Candidates[v] = struct{}{}
	}
	for i, v := range recents {
		snap.Recents[number-uint64(len(recents)-1-i)] = v
	}
	snap.Attestation = &types.VoteData{
		SourceNumber: attestation.SourceNumber,
		SourceHash:   attestation.SourceHash,
		TargetNumber: attestation.TargetNumber,
		TargetHash:   attestation.TargetHash,
	}
	return snap
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(database db.Database) error {
	if bucket, err := database.GetBucket("Snapshot"); err != nil {
		return err
	} else {
		blob, err := json.Marshal(s)
		if err != nil {
			return err
		}
		s.log.Debugf("store snapshot - number(%d) hash(%s)", s.Number, s.Hash.Hex())
		return bucket.Set(append([]byte("snap-"), s.Hash[:]...), blob)
	}
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(database db.Database, hash common.Hash, log log.Logger) (*Snapshot, error) {
	if database == nil {
		return nil, errors.New("NoDatabase")
	}
	if bucket, err := database.GetBucket("Snapshot"); err != nil {
		return nil, err
	} else {
		blob, err := bucket.Get(append([]byte("snap-"), hash[:]...))
		if err != nil {
			return nil, err
		}
		snap := new(Snapshot)
		if err := json.Unmarshal(blob, snap); err != nil {
			return nil, err
		}
		snap.log = log
		return snap, nil
	}
}

func hasSnapshot(database db.Database, hash common.Hash) (bool, error) {
	if database == nil {
		return false, nil
	}
	if bucket, err := database.GetBucket("Snapshot"); err != nil {
		return false, err
	} else {
		return bucket.Has(append([]byte("snap-"), hash[:]...))
	}
}

// copy creates a deep copy of the snapshot
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		Number:     s.Number,
		Hash:       s.Hash,
		ParentHash: s.ParentHash,
		Validators: make(map[common.Address]struct{}),
		Candidates: make(map[common.Address]struct{}),
		Recents:    make(map[uint64]common.Address),
		Sealer:     s.Sealer,
		log:        s.log,
	}

	for v := range s.Validators {
		cpy.Validators[v] = struct{}{}
	}
	for v := range s.Candidates {
		cpy.Candidates[v] = struct{}{}
	}
	for block, v := range s.Recents {
		cpy.Recents[block] = v
	}
	if s.Attestation != nil {
		cpy.Attestation = &types.VoteData{
			SourceNumber: s.Attestation.SourceNumber,
			SourceHash:   s.Attestation.SourceHash,
			TargetNumber: s.Attestation.TargetNumber,
			TargetHash:   s.Attestation.TargetHash,
		}
	}
	return cpy
}

func (s *Snapshot) updateAttestation(header *types.Header) {
	// The attestation should have been checked in verify header, update directly
	atte, _ := getVoteAttestationFromHeader(header)
	if atte == nil {
		s.log.Infof("NoVoteAttestation - H(%d:%.8s)", header.Number.Uint64(), header.Hash().Hex())
		return
	}

	tn, th := atte.Data.TargetNumber, atte.Data.TargetHash
	sn, sh := atte.Data.SourceNumber, atte.Data.SourceHash
	s.log.Tracef("Attestation - H(%d:%.8s) T(%d:%.8s) S(%d:%.8s)",
		header.Number.Uint64(), header.Hash().Hex(), tn, th.Hex(), sn, sh.Hex())

	// Headers with bad attestation are accepted before Plato upgrade,
	// but Attestation of snapshot is only updated when the target block is direct parent of the header
	if th != header.ParentHash || tn+1 != header.Number.Uint64() {
		s.log.Warnf("InvalidVoteAttestation - Expected(%d:%.8s) Actual(%d:%.8s)",
			header.Number.Uint64()-1, header.ParentHash.Hex(), tn, th.Hex())
		return
	}

	// Update attestation
	if s.Attestation != nil && sn+1 != tn {
		s.Attestation.TargetNumber = tn
		s.Attestation.TargetHash = th
	} else {
		s.Attestation = atte.Data
	}
}

// TODO handle recent fork hash
func (s *Snapshot) apply(head *types.Header, cid *big.Int) (*Snapshot, error) {
	if head == nil {
		return s, nil
	}

	if s.Number+1 != head.Number.Uint64() {
		return nil, errors.New("Out of range block")
	}
	if s.Hash != head.ParentHash {
		s.log.Infof("inconsistent block hash - curr(%d:%s) next(%d:%s:%s)",
			s.Number, s.Hash.Hex(), head.Number.Uint64(), head.ParentHash.Hex(), head.Hash())
		return nil, ErrInconsistentBlock
	}

	snap := s.copy()
	number := head.Number.Uint64()
	if limit := uint64(len(snap.Validators)/2 + 1); number >= limit {
		delete(snap.Recents, number-limit)
	}
	validator, err := ecrecover(head, cid)
	if err != nil {
		return nil, err
	}
	if _, ok := snap.Validators[validator]; !ok {
		return nil, errors.New("UnauthorizedValidator")
	}
	for _, recent := range snap.Recents {
		if recent == validator {
			return nil, errors.New("RecentlySigned")
		}
	}
	snap.Recents[number] = validator

	if number > 0 && number%uint64(epoch) == uint64(len(snap.Validators)/2) {
		oldLimit := len(snap.Validators)/2 + 1
		newLimit := len(snap.Candidates)/2 + 1
		if newLimit < oldLimit {
			for i := 0; i < oldLimit-newLimit; i++ {
				delete(snap.Recents, number-uint64(newLimit)-uint64(i))
			}
		}
		snap.Validators = snap.Candidates
	}

	if number > 0 && number%uint64(epoch) == 0 {
		newValArr, err := parseValidators(head)
		if err != nil {
			return nil, err
		}
		newVals := make(map[common.Address]struct{}, len(newValArr))
		for _, val := range newValArr {
			newVals[val] = struct{}{}
		}
		snap.Candidates = newVals
	}
	snap.updateAttestation(head)
	snap.Number += uint64(1)
	snap.Hash = head.Hash()
	snap.ParentHash = head.ParentHash
	snap.Sealer = validator
	return snap, nil
}

// inturn returns if a validator at a given block height is in-turn or not.
func (s *Snapshot) inturn(validator common.Address) bool {
	validators := s.validators()
	offset := (s.Number + 1) % uint64(len(validators))
	return validators[offset] == validator
}

// validators retrieves the list of validators in ascending order.
func (s *Snapshot) validators() []common.Address {
	validators := make([]common.Address, 0, len(s.Validators))
	for v := range s.Validators {
		validators = append(validators, v)
	}
	sort.Sort(validatorsAscending(validators))
	return validators
}

// func ParseValidators(header *types.Header) ([]common.Address, error) {
// 	validatorsBytes := getValidatorBytesFromHeader(header)
// 	if len(validatorsBytes) == 0 {
// 		return nil, errors.New("invalid validators bytes")
// 	}
//
// 	n := len(validatorsBytes) / validatorBytesLength
// 	cnsAddrs := make([]common.Address, n)
// 	for i := 0; i < n; i++ {
// 		cnsAddrs[i] = common.BytesToAddress(validatorsBytes[i*validatorBytesLength : i*validatorBytesLength+common.AddressLength])
// 	}
// 	return cnsAddrs, nil
// }

func parseValidators(header *types.Header) ([]common.Address /*, []types.BLSPublicKey*/, error) {
	validatorsBytes := getValidatorBytesFromHeader(header)
	if len(validatorsBytes) == 0 {
		return nil, errors.New("invalid validators bytes")
	}

	n := len(validatorsBytes) / validatorBytesLength
	cnsAddrs := make([]common.Address, n)
	//voteAddrs := make([]types.BLSPublicKey, n)
	for i := 0; i < n; i++ {
		cnsAddrs[i] = common.BytesToAddress(validatorsBytes[i*validatorBytesLength : i*validatorBytesLength+common.AddressLength])
		//copy(voteAddrs[i][:], validatorsBytes[i*validatorBytesLength+common.AddressLength:(i+1)*validatorBytesLength])
	}
	return cnsAddrs, nil
}

// getValidatorBytesFromHeader returns the validators bytes extracted from the header's extra field if exists.
// The validators bytes would be contained only in the epoch block's header, and its each validator bytes length is fixed.
// On luban fork, we introduce vote attestation into the header's extra field, so extra format is different from before.
// Before luban fork: |---Extra Vanity---|---Validators Bytes (or Empty)---|---Extra Seal---|
// After luban fork:  |---Extra Vanity---|---Validators Number and Validators Bytes (or Empty)---|---Vote Attestation (or Empty)---|---Extra Seal---|
func getValidatorBytesFromHeader(header *types.Header) []byte {
	if len(header.Extra) <= extraVanity+extraSeal {
		return nil
	}

	if header.Number.Uint64()%epoch != 0 {
		return nil
	}
	num := int(header.Extra[extraVanity])
	if num == 0 || len(header.Extra) <= extraVanity+extraSeal+num*validatorBytesLength {
		return nil
	}
	start := extraVanity + 1
	end := start + num*validatorBytesLength
	return header.Extra[start:end]
}

// func FindAncientHeader(header *types.Header, ite uint64, chain consensus.ChainHeaderReader, candidateParents []*types.Header) *types.Header {
func FindAncientHeader(header *types.Header, ite uint64, candidateParents []*types.Header) *types.Header {
	ancient := header
	for i := uint64(1); i <= ite; i++ {
		parentHash := ancient.ParentHash
		parentHeight := ancient.Number.Uint64() - 1
		found := false
		if len(candidateParents) > 0 {
			index := sort.Search(len(candidateParents), func(i int) bool {
				return candidateParents[i].Number.Uint64() >= parentHeight
			})
			if index < len(candidateParents) && candidateParents[index].Number.Uint64() == parentHeight &&
				candidateParents[index].Hash() == parentHash {
				ancient = candidateParents[index]
				found = true
			}
		}
		// if !found {
		// 	ancient = chain.GetHeader(parentHash, parentHeight)
		// 	found = true
		// }
		if ancient == nil || !found {
			return nil
		}
	}
	return ancient
}

// ecrecover extracts the ethereum account address from a signed header.
// func ecrecover(header *types.Header, sigCache *lru.ARCCache, chainId *big.Int) (common.Address, error) {
func ecrecover(header *types.Header, chainId *big.Int) (common.Address, error) {
	if len(header.Extra) < extraSeal {
		return common.Address{}, errors.New("errMissingSignature")
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header, chainId).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	return signer, nil
}

// ===========================     utility function        ==========================
// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header, chainId *big.Int) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header, chainId)
	hasher.Sum(hash[:0])
	return hash
}

func encodeSigHeader(w io.Writer, header *types.Header, chainId *big.Int) {
	err := rlp.Encode(w, []interface{}{
		chainId,
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // this will panic if extra is too short, should check before calling encodeSigHeader
		header.MixDigest,
		header.Nonce,
	})
	if err != nil {
		panic("can't encode: " + err.Error())
	}
}

func BootSnapshot(epoch uint64, head *types.Header, client *ethclient.Client, log log.Logger) (*Snapshot, error) {
	curVals, err := parseValidators(head)
	if err != nil {
		return nil, err
	}

	// TODO null attestation
	attestation, err := getVoteAttestationFromHeader(head)
	if err != nil {
		return nil, err
	}

	if head.Number.Uint64() == 0 {
		return newSnapshot(head.Number.Uint64(), head.Hash(), curVals, curVals,
			make([]common.Address, 0), nil, head.Coinbase, head.ParentHash, log), nil
	}

	number := new(big.Int).SetUint64(head.Number.Uint64() - epoch)
	oldHead, err := client.HeaderByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	oldVals, err := parseValidators(oldHead)
	if err != nil {
		return nil, err
	}

	recents := make([]common.Address, 0)
	for i := head.Number.Int64() - int64(len(oldVals)/2); i <= head.Number.Int64(); i++ {
		if oldHead, err = client.HeaderByNumber(context.Background(), big.NewInt(i)); err != nil {
			return nil, err
		} else {
			recents = append(recents, oldHead.Coinbase)
		}
	}
	return newSnapshot(head.Number.Uint64(), head.Hash(), oldVals,
		curVals, recents, attestation.Data, head.Coinbase, head.ParentHash, log), nil
}

func getVoteAttestationFromHeader(header *types.Header) (*types.VoteAttestation, error) {
	if len(header.Extra) <= extraVanity+extraSeal {
		return nil, nil
	}

	var attestationBytes []byte
	if header.Number.Uint64()%epoch != 0 {
		attestationBytes = header.Extra[extraVanity : len(header.Extra)-extraSeal]
	} else {
		num := int(header.Extra[extraVanity])
		if len(header.Extra) <= extraVanity+extraSeal+validatorNumberSize+num*validatorBytesLength {
			return nil, nil
		}
		start := extraVanity + validatorNumberSize + num*validatorBytesLength
		end := len(header.Extra) - extraSeal
		attestationBytes = header.Extra[start:end]
	}

	var attestation types.VoteAttestation
	if err := rlp.Decode(bytes.NewReader(attestationBytes), &attestation); err != nil {
		return nil, fmt.Errorf("block %d has vote attestation info, decode err: %s", header.Number.Uint64(), err)
	}
	return &attestation, nil
}
