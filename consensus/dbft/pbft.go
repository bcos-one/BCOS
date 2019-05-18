package dbft

import (
	"errors"
	"fmt"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/event"
	"github.com/bcos-one/BCOS/p2p"
	"github.com/bcos-one/BCOS/rlp"
	"io"
)

const (
	MsgPreprepare uint64 = iota
	MsgPrepare
	MsgCommit
)

// Message defines  message format of the pbft engine
type Message struct {
	Code          uint64         // code type contains MsgPreprepare,MsgPrepare,MsgCommit
	Msg           []byte         // content of the Message
	Address       common.Address // address of the proposer
	Signature     []byte         // signed hash of the Msg by proposer
	CommittedSeal []byte         //
}

// EncodeRLP serializes m into the Ethereum RLP format.
func (m *Message) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{m.Code, m.Msg, m.Address, m.Signature, m.CommittedSeal})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (m *Message) DecodeRLP(s *rlp.Stream) error {
	var msg struct {
		Code          uint64
		Msg           []byte
		Address       common.Address
		Signature     []byte
		CommittedSeal []byte
	}

	if err := s.Decode(&msg); err != nil {
		return err
	}
	m.Code, m.Msg, m.Address, m.Signature, m.CommittedSeal = msg.Code, msg.Msg, msg.Address, msg.Signature, msg.CommittedSeal
	return nil
}

// FromPayload decode the payload from the p2p peer and check signature
func (m *Message) FromPayload(b []byte) error {
	// Decode message
	err := rlp.DecodeBytes(b, &m)
	if err != nil {
		return err
	}

	return m.CheckSignature()
}

func (m *Message) CheckSignature() error {
	var payload []byte
	payload, err := m.PayloadNoSig()
	if err != nil {
		return err
	}

	signer, err := CheckValidatorSignature(payload, m.Signature)
	if err != nil {
		return err
	}

	if signer != m.Address {
		return errors.New("invalid signature")
	}

	return nil
}

func (m *Message) Payload() ([]byte, error) {
	return rlp.EncodeToBytes(m)
}

func (m *Message) PayloadNoSig() ([]byte, error) {
	return rlp.EncodeToBytes(&Message{
		Code:          m.Code,
		Msg:           m.Msg,
		Address:       m.Address,
		Signature:     []byte{},
		CommittedSeal: m.CommittedSeal,
	})
}

func (m *Message) Decode(val interface{}) error {
	return rlp.DecodeBytes(m.Msg, val)
}

func (m *Message) String() string {
	return fmt.Sprintf("{Code: %v, Address: %v}", m.Code, m.Address.String())
}

type PBFT interface {
	Start() error
	Stop() error
	StartConsensus(validators Validators, proposal Proposal) (error)
	PrePrepare(msg *Message) (error)
	Prepare(msg *Message) (error)
	Commit(msg *Message) (error)
	SubscribeNewMsgEvent(chan<- consensus.PbftMsg) event.Subscription
	DispatchMsg(address common.Address, msg p2p.Msg) (bool, error)
}
