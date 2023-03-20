package bsc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/icon-project/btp2/common/db"
	"golang.org/x/crypto/sha3"
)

const (
	validatorBytesLength = common.AddressLength
	extraVanity          = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal            = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

// validatorsAscending implements the sort interface to allow sorting a list of addresses
type validatorsAscending []common.Address

func (s validatorsAscending) Len() int           { return len(s) }
func (s validatorsAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s validatorsAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type Snapshot struct {
	// config // epoch

	Number     uint64                      `json:"number"`
	Hash       common.Hash                 `json:"hash"`
	ParentHash common.Hash                 `json:"parent_hash"`
	Sealer     common.Address              `json:"sealer"`
	Validators map[common.Address]struct{} `json:"validators"`
	Candidates map[common.Address]struct{} `json:"candidates"`
	Recents    map[uint64]common.Address   `json:"recents"`
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
	sealer common.Address,
	parentHash common.Hash,
) *Snapshot {
	snap := &Snapshot{
		Number:     number,
		Hash:       hash,
		ParentHash: parentHash,
		Validators: make(map[common.Address]struct{}),
		Candidates: make(map[common.Address]struct{}),
		Recents:    make(map[uint64]common.Address),
		Sealer:     sealer,
	}
	for _, v := range validators {
		snap.Validators[v] = struct{}{}
	}
	for _, v := range candidates {
		snap.Candidates[v] = struct{}{}
	}
	for i, v := range recents {
		snap.Recents[number+uint64(i)] = v
	}

	return snap
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(database db.Database) error {
	fmt.Println("++Snapshot::store")
	defer fmt.Println("--Snapshot::store", s.Number, s.Hash)
	if bucket, err := database.GetBucket("Snapshot"); err != nil {
		return err
	} else {
		blob, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return bucket.Set(append([]byte("snap-"), s.Hash[:]...), blob)
	}
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(database db.Database, hash common.Hash) (*Snapshot, error) {
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
		return snap, nil
	}
}

func hasSnapshot(database db.Database, hash common.Hash) (bool, error) {
	if bucket, err := database.GetBucket("Snapshot"); err != nil {
		return false, err
	} else {
		return bucket.Has(append([]byte("snap-"), hash[:]...)), nil
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
	return cpy
}

// TODO handle recent fork hash
func (s *Snapshot) apply(head *types.Header, cid *big.Int) (*Snapshot, error) {
	if head == nil {
		return s, nil
	}

	if s.Number+1 != head.Number.Uint64() {
		return nil, errors.New("Out of range block")
	}
	if !bytes.Equal(s.Hash.Bytes(), head.ParentHash.Bytes()) {
		return nil, errors.New("Inconsistent Block Hash")
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

	if number > 0 && number%uint64(200) == uint64(len(snap.Validators)/2) {
		oldLimit := len(snap.Validators)/2 + 1
		newLimit := len(snap.Candidates)/2 + 1
		if newLimit < oldLimit {
			for i := 0; i < oldLimit-newLimit; i++ {
				delete(snap.Recents, number-uint64(newLimit)-uint64(i))
			}
		}
		snap.Validators = snap.Candidates
	}

	if number > 0 && number%uint64(200) == 0 {
		validatorBytes := head.Extra[extraVanity : len(head.Extra)-extraSeal]
		newValArr, err := ParseValidators(validatorBytes)
		if err != nil {
			return nil, err
		}
		newVals := make(map[common.Address]struct{}, len(newValArr))
		for _, val := range newValArr {
			newVals[val] = struct{}{}
		}
		snap.Candidates = newVals
	}
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

func ParseValidators(validatorsBytes []byte) ([]common.Address, error) {
	if len(validatorsBytes)%validatorBytesLength != 0 {
		return nil, errors.New("invalid validators bytes")
	}
	n := len(validatorsBytes) / validatorBytesLength
	result := make([]common.Address, n)
	for i := 0; i < n; i++ {
		address := make([]byte, validatorBytesLength)
		copy(address, validatorsBytes[i*validatorBytesLength:(i+1)*validatorBytesLength])
		result[i] = common.BytesToAddress(address)
	}
	return result, nil
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
	// If the signature's already cached, return that
	// hash := header.Hash()
	// if address, known := sigCache.Get(hash); known {
	// 	return address.(common.Address), nil
	// }
	// Retrieve the signature from the header extra-data
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

	// sigCache.Add(hash, signer)
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
