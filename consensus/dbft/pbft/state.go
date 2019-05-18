package pbft

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/log"
	"math/big"
	"reflect"
)

type State struct {
	sequence   *big.Int
	validators dbft.Validators
	preprepare *dbft.Preprepare
	prepares   map[common.Address]bool
	commits    map[common.Address]*dbft.Message
	finished   bool
}

func newState(validators dbft.Validators, proposer common.Address, proposal dbft.Proposal) *State {
	return &State{
		sequence:   proposal.Number(),
		validators: validators,
		preprepare: &dbft.Preprepare{
			View: &dbft.View{
				Proposer: proposer,
				Sequence: proposal.Number(),
			},
			Proposal: proposal,
		},
		prepares: make(map[common.Address]bool),
		commits:  make(map[common.Address]*dbft.Message),
	}
}

func (s *State) Subject() *dbft.Subject {
	return &dbft.Subject{
		View:   s.preprepare.View,
		Digest: s.preprepare.Proposal.Hash(),
	}
}

func (s *State) verifyPrepare(prepare *dbft.Subject) (error) {
	subject := s.Subject()

	if common.Big1.Cmp(new(big.Int).Sub(prepare.View.Sequence, subject.View.Sequence)) == 0 {
		return errFutureMessage
	}

	if prepare.View.Sequence.Cmp(subject.View.Sequence) < 0 {
		return errOldMessage
	}

	if !reflect.DeepEqual(prepare, subject) {
		log.Warn("Inconsistent subjects between PREPARE and proposal", "expected", subject, "got", prepare)
		return errInconsistentSubject
	}

	return nil
}

func (s *State) acceptPrepare(subject *dbft.Subject, address common.Address) {
	s.prepares[address] = true
}

func (s *State) prepared() bool {
	if len(s.prepares) > 2*s.validators.F() {
		return true
	}
	return false
}

func (s *State) verifyCommit(commit *dbft.Subject) (error) {
	subject := s.Subject()

	if common.Big1.Cmp(new(big.Int).Sub(commit.View.Sequence, subject.View.Sequence)) == 0 {
		return errFutureMessage
	}

	if subject.View.Sequence.Cmp(subject.View.Sequence) < 0 {
		return errOldMessage
	}

	if !reflect.DeepEqual(commit, subject) {
		log.Warn("Inconsistent subjects between PREPARE and proposal", "expected", subject, "got", commit)
		return errInconsistentSubject
	}

	return nil
}

func (s *State) acceptCommit(msg *dbft.Message) {
	s.commits[msg.Address] = msg
}

func (s *State) commited() bool {
	if len(s.commits) > 2*s.validators.F() {
		return true
	}
	return false
}

func (s *State) commitSeals() [][]byte {
	committedSeals := make([][]byte, len(s.commits))
	i := 0

	for _, v := range s.commits {
		committedSeals[i] = make([]byte, types.DbftExtraSeal)
		copy(committedSeals[i][:], v.CommittedSeal[:])
		i++
	}

	return committedSeals
}
