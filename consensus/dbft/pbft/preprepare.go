package pbft

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/rlp"
)

func (e *engine) PrePrepare(msg *dbft.Message) (error) {
	var prepare dbft.Preprepare
	e.logger.Trace("handle preprepare")
	if err := msg.Decode(&prepare); err != nil {
		return err
	}

	return e.preprepare(prepare.Proposal, msg.Address)
}

func (e *engine) preprepare(proposal dbft.Proposal, proposer common.Address) (error) {
	validators := e.backend.Validators(proposal)
	if validators == nil {
		return errInvalidProposal
	}
	if !validators.IsValidator(proposer) {
		return errUnauthorizedAddress
	}

	if err := e.backend.Verify(proposal); err != nil {
		return err
	}

	state := newState(validators, proposer, proposal)

	e.sendPrepare(validators, state.Subject(), proposal)

	e.setState(proposer, state)
	return nil
}

func (e *engine) sendPrePrepare(validators dbft.Validators, proposal dbft.Proposal) {
	preprepare := &dbft.Preprepare{
		View: &dbft.View{
			Proposer: e.address,
			Sequence: proposal.Number(),
		},
		Proposal: proposal,
	}
	msg, err := rlp.EncodeToBytes(preprepare)
	if err != nil {
		return
	}
	message := &dbft.Message{
		Code: dbft.MsgPreprepare,
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
