package pbft

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/rlp"
)

func (e *engine) Prepare(msg *dbft.Message) (error) {
	var prepare dbft.Subject
	if err := msg.Decode(&prepare); err != nil {
		return err
	}

	return e.prepare(&prepare, msg.Address)
}

func (e *engine) prepare(prepare *dbft.Subject, validator common.Address) (error) {
	e.logger.Trace("handle prepare")
	logger := e.logger.New("prepare", &prepare)

	e.mutex.Lock()
	defer e.mutex.Unlock()
	var state *State
	state, ok := e.states[prepare.View.Proposer]; if !ok {
		return errFutureMessage
	}

	if state.prepared() || state.commited() {
		return nil
	}
	if !state.validators.IsValidator(validator) {
		return errUnauthorizedAddress
	}

	if err := state.verifyPrepare(prepare); err != nil {
		return err
	}

	logger.Trace("accept prepare")
	state.acceptPrepare(prepare, validator)

	if state.prepared() {
		logger.Trace("PBFT prepared")
		e.sendCommit(state.validators, state.Subject(), state.preprepare.Proposal)
	}
	return nil
}

func (e *engine) sendPrepare(validators dbft.Validators, prepare *dbft.Subject, proposal dbft.Proposal) {
	msg, err := rlp.EncodeToBytes(prepare)
	if err != nil {
		return
	}

	message := &dbft.Message{
		Code: dbft.MsgPrepare,
		Msg:  msg,
	}

	payload, err := e.finalizeMessage(message, proposal)

	go e.feed.Send(
		consensus.PbftMsg{
			Peers:   validators.Addresses(),
			Payload: payload,
		},
	)
}