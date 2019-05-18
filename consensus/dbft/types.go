package dbft

import (
	"fmt"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/rlp"
	"io"
	"math"
	"math/big"
)

// Proposal supports retrieving height and serialized block to be used during dbft consensus.
type Proposal interface {
	// Number retrieves the sequence number of this proposal.
	Number() *big.Int

	Time() *big.Int

	// Hash retrieves the hash of this proposal.
	Hash() common.Hash

	EncodeRLP(w io.Writer) error

	DecodeRLP(s *rlp.Stream) error
}

// View includes proposer address and a sequence number.
// Sequence is the block number we'd like to commit.
type View struct {
	Proposer common.Address
	Sequence *big.Int
}

// EncodeRLP serializes b into the Ethereum RLP format.
func (v *View) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{v.Proposer, v.Sequence})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (v *View) DecodeRLP(s *rlp.Stream) error {
	var view struct {
		Proposer common.Address
		Sequence *big.Int
	}

	if err := s.Decode(&view); err != nil {
		return err
	}
	v.Proposer, v.Sequence = view.Proposer, view.Sequence
	return nil
}

func (v *View) String() string {
	return fmt.Sprintf("{Proposer: %s, Sequence: %d}", v.Proposer.Hex(), v.Sequence.Uint64())
}

func (v *View) Cmp(y *View) int {
	if v.Sequence.Cmp(y.Sequence) != 0 {
		return v.Sequence.Cmp(y.Sequence)
	}

	if v.Proposer != y.Proposer {
		return -1
	}

	return 0
}

type Preprepare struct {
	View     *View
	Proposal Proposal
}

// EncodeRLP serializes b into the Ethereum RLP format.
func (b *Preprepare) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{b.View, b.Proposal})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (b *Preprepare) DecodeRLP(s *rlp.Stream) error {
	var preprepare struct {
		View     *View
		Proposal *types.Block
	}

	if err := s.Decode(&preprepare); err != nil {
		return err
	}
	b.View, b.Proposal = preprepare.View, preprepare.Proposal

	return nil
}

type Subject struct {
	View   *View
	Digest common.Hash
}

// EncodeRLP serializes b into the Ethereum RLP format.
func (b *Subject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{b.View, b.Digest})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (b *Subject) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		View   *View
		Digest common.Hash
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}
	b.View, b.Digest = subject.View, subject.Digest
	return nil
}

func (b *Subject) String() string {
	return fmt.Sprintf("{View: %v, Digest: %v}", b.View, b.Digest.Hex())
}

type Validators []common.Address

// Get the maximum number of faulty nodes
func (v Validators) F() int {
	return int(math.Ceil(float64(len(v))/3)) - 1
}

func (v Validators) IsValidator(validator common.Address) bool {
	for _, val := range v {
		if val == validator {
			return true
		}
	}

	return false
}

func (v Validators) Addresses() []common.Address {
	return []common.Address(v)
}
