package bsc2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type VerifierStatus struct {
	Offset    uint64
	BlockTree *BlockTree
}

func (vs VerifierStatus) String() string {
	return fmt.Sprintf("{\"Number\"=%d, \"BlockTree\"=%s}", vs.Offset, vs.BlockTree.String())
}

// EncodeRLP implements rlp.Encoder
func (vs *VerifierStatus) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		Offset    uint64
		BlockTree *BlockTree
	}{
		Offset:    vs.Offset,
		BlockTree: vs.BlockTree,
	})
}

// DecodeRLP implements rlp.Decoder
func (vs *VerifierStatus) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	if offset, err := s.Uint64(); err != nil {
		return err
	} else {
		vs.Offset = offset
	}

	if blob, err := s.Raw(); err != nil {
		return err
	} else {
		if err := rlp.DecodeBytes(blob, &vs.BlockTree); err != nil {
			return err
		}
	}

	if err := s.ListEnd(); err != nil {
		return err
	}
	return nil
}

type BlockTree struct {
	root  common.Hash
	nodes map[common.Hash][]common.Hash
}

func NewBlockTree(id common.Hash) *BlockTree {
	return &BlockTree{
		root: id,
		nodes: map[common.Hash][]common.Hash{
			id: make([]common.Hash, 0),
		},
	}
}

func (o *BlockTree) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("{\"Root\": \"%s\",", o.root))
	buf.WriteString("\"Nodes\": [")
	for k, v := range o.nodes {
		buf.WriteString(fmt.Sprintf("{\"ID\": \"%s\",", k))
		buf.WriteString("\"Children\": [")
		for i, e := range v {
			buf.WriteString(fmt.Sprintf("\"%s\"", e))
			if i+1 < len(v) {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("]}")
		buf.WriteString(",")
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString("]}")
	return buf.String()
}

func (o *BlockTree) Json() string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(o.String()), "", "  ")
	if err != nil {
		panic(err)
	}
	return out.String()
}

func (o *BlockTree) Root() common.Hash {
	return o.root
}

func (o *BlockTree) ChildrenOf(parent common.Hash) []common.Hash {
	children := o.nodes[parent]
	clone := make([]common.Hash, len(children))
	copy(clone, children)
	return clone
}

func (o *BlockTree) Add(parent, hash common.Hash) error {
	if _, ok := o.nodes[hash]; ok {
		return errors.New("DuplicatedHash")
	}
	if descendants, ok := o.nodes[parent]; !ok {
		return errors.New("NoParentHash")
	} else {
		o.nodes[parent] = append(descendants, hash)
		o.nodes[hash] = make([]common.Hash, 0)
	}
	return nil
}

func (o *BlockTree) Prune(until common.Hash) {
	removals := append(make([]common.Hash, 0), o.root)

	for len(removals) > 0 {
		buf := make([]common.Hash, 0)
		for _, removal := range removals {
			leaves, _ := o.nodes[removal]
			for _, leaf := range leaves {
				if leaf != until {
					buf = append(buf, leaf)
				}
			}
			delete(o.nodes, removal)
		}
		removals = buf
	}
	o.root = until
}

func (o *BlockTree) Has(id common.Hash) bool {
	_, ok := o.nodes[id]
	return ok
}

type pair struct {
	id common.Hash
	sz uint64
}

// EncodeRLP implements rlp.Encoder
func (o *BlockTree) EncodeRLP(w io.Writer) error {
	ret := make([]interface{}, 0)
	children := []common.Hash{
		o.root,
	}
	for len(children) > 0 {
		var node common.Hash
		node, children = children[0], children[1:]
		tmp := o.nodes[node]
		ret = append(ret, uint(len(tmp)), node)
		if len(tmp) > 0 {
			children = append(children, tmp...)
		}
	}
	return rlp.Encode(w, &ret)
}

// DecodeRLP implements rlp.Decoder
func (o *BlockTree) DecodeRLP(s *rlp.Stream) error {
	blob, err := s.Raw()
	if err != nil {
		return err
	}

	it, err := rlp.NewListIterator(blob)
	if err != nil {
		return err
	}

	if o.nodes == nil {
		o.nodes = make(map[common.Hash][]common.Hash, 0)
	}

	var root = pair{}
	if !it.Next() {
		return errors.New("Invalid BlockTree Blob: no leaf count of root")
	}
	if err := rlp.DecodeBytes(it.Value(), &root.sz); err != nil {
		return err
	}

	if !it.Next() {
		return errors.New("Invalid BlockTree Blob: no root id")
	}
	if err := rlp.DecodeBytes(it.Value(), &root.id); err != nil {
		return err
	}

	o.root = root.id
	items := []pair{root}

	for len(items) > 0 {
		var item pair
		item, items = items[0], items[1:]
		id := item.id
		children := make([]common.Hash, 0)
		for i := 0; i < int(item.sz); i++ {
			c := pair{}
			it.Next()
			if err := rlp.DecodeBytes(it.Value(), &c.sz); err != nil {
				return err
			}
			it.Next()
			if err := rlp.DecodeBytes(it.Value(), &c.id); err != nil {
				return err
			}
			children = append(children, c.id)
			items = append(items, c)
		}
		o.nodes[id] = children
	}
	return nil
}
