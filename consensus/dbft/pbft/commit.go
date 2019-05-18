package pbft

import (
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/rlp"
)

func (e *engine) Commit(msg *dbft.Message) (error) {
	var commit dbft.Subject
	if err := msg.Decode(&commit); err != nil {
		return err
	}
	logger := e.logger.New("commit", &commit)

	e.mutex.Lock()
	defer e.mutex.Unlock()

	var state *State
	state, ok := e.states[commit.View.Proposer]; if !ok {
		return errFutureMessage
	}

	if state.commited() || state.finished {
		return nil
	}
	if !state.validators.IsValidator(msg.Address) {
		return errUnauthorizedAddress
	}

	if err := state.verifyPrepare(&commit); err != nil {
		return err
	}

	state.acceptCommit(msg)
	logger.Trace("accept commit")

	if state.commited() {
		logger.Trace("PBFT committed")

		e.backend.Commit(state.preprepare.Proposal, state.commitSeals())
		state.finished = true
	}

	return nil
}

func (e *engine) sendCommit(validators dbft.Validators, commit *dbft.Subject, proposal dbft.Proposal) {
	msg, err := rlp.EncodeToBytes(commit)
	if err != nil {
		return
	}

	message := &dbft.Message{
		Code: dbft.MsgCommit,
		Msg:  msg,
	}

	payload, err := e.finalizeMessage(message, proposal)
	if err != nil {
		return
	}

	go e.feed.Send(
		consensus.PbftMsg{
			Peers:   validators.Addresses(),
			Payload: payload,
		},
	)
}