package pbft

import (
	"errors"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"github.com/bcos-one/BCOS/event"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/p2p"
	"sync"
	"sync/atomic"
)

const msgChanSize = 64
const maxFutureMsgLen = 64

var (
	// errDecodeFailed is returned when decode message fails
	errDecodeFailed = errors.New("fail to decode dbft message")
	// ErrStoppedEngine is returned if the engine is stopped
	errStoppedEngine = errors.New("stopped engine")
	// ErrStartedEngine is returned if the engine is already started
	errStartedEngine = errors.New("started engine")

	errExpiredProposal = errors.New("expired proposal")

	// errFutureMessage is returned when current view is earlier than the
	// view of the received message.
	errFutureMessage = errors.New("future message")

	errOldMessage = errors.New("Old Meesage")

	// errInconsistentSubject is returned when received subject is different from
	// current subject.
	errInconsistentSubject = errors.New("inconsistent subjects")

	// errUnauthorizedAddress is returned when given address cannot be found in
	// current validator set.
	errUnauthorizedAddress = errors.New("unauthorized address")

	errInvalidProposal = errors.New("invalid proposal")
)
// New create pbft engine
func New(backend dbft.Backend, address common.Address) dbft.PBFT {
	e := &engine{
		backend:    backend,
		address:    address,
		states:     make(map[common.Address]*State),
		pendingMsg: make(map[common.Address][]*dbft.Message),
	}
	e.logger = log.New("address", e.address)

	return e
}

type engine struct {
	running int32
	logger  log.Logger

	backend dbft.Backend
	address common.Address

	states map[common.Address]*State // proposer address => State
	mutex  sync.RWMutex              // lock for states

	pendingMsg        map[common.Address][]*dbft.Message
	pendingRequestsMu sync.Mutex

	feed   event.Feed
	msgSub event.Subscription
}

func (e *engine) StartConsensus(validators dbft.Validators, proposal dbft.Proposal) (error) {
	logger := e.logger.New("proposal", proposal.Hash())
	logger.Trace("start consensus")

	e.sendPrePrepare(validators, proposal)
	return nil
}

func (e *engine) Start() error {
	if e.isRunning() {
		return errStartedEngine
	}

	msgCh := make(chan consensus.PbftMsg, msgChanSize)
	e.msgSub = e.SubscribeNewMsgEvent(msgCh)
	go e.handleEvent(msgCh)

	atomic.StoreInt32(&e.running, 1)
	return nil
}

func (e *engine) Stop() error {
	if !e.isRunning() {
		return errStoppedEngine
	}

	e.msgSub.Unsubscribe()
	atomic.StoreInt32(&e.running, 0)
	return nil
}

func (e *engine) SubscribeNewMsgEvent(ch chan<- consensus.PbftMsg) event.Subscription {
	return e.feed.Subscribe(ch)
}

func (e *engine) DispatchMsg(address common.Address, msg p2p.Msg) (bool, error) {
	if !e.isRunning() {
		return true, errStoppedEngine
	}

	var data []byte
	if err := msg.Decode(&data); err != nil {
		log.Info("Decode msg error", "error", err)
		return true, errDecodeFailed
	}

	pbftMsg := new(dbft.Message)
	if err := pbftMsg.FromPayload(data); err != nil {
		log.Info("PBFT Handle msg error", "err", err)
		return true, err
	}

	e.handleMsg(pbftMsg)
	return true, nil
}

func (e *engine) handleEvent(msgCh <-chan consensus.PbftMsg) {
	for {
		select {
		case ev := <-msgCh:
			message := new(dbft.Message)
			if err := message.FromPayload(ev.Payload); err != nil {
				log.Info("PBFT Handle msg error", "err", err)
			}
			if err := e.handleMsg(message); err != nil {
				log.Info("PBFT handleMsg error", "err", err)
			}
			break
		case <-e.msgSub.Err():
			return
		}
	}
}

func (e *engine) handleMsg(message *dbft.Message) error {
	var err error
	switch message.Code {
	case dbft.MsgPreprepare:
		err = e.PrePrepare(message)
		if err == nil {
			e.proecessPendingRequest(message.Address)
		}
	case dbft.MsgPrepare:
		err = e.Prepare(message)
	case dbft.MsgCommit:
		err = e.Commit(message)
	default:
		log.Error("Invalid pbft Message")
	}

	if err == errFutureMessage {
		e.storeFutureMsg(message)
	}

	if err == errOldMessage {
		// ignore old message
		return nil
	}
	return err
}

func (e *engine) storeFutureMsg(message *dbft.Message) {
	var subject dbft.Subject
	if err := message.Decode(&subject); err != nil {
		return
	}

	proposer := subject.View.Proposer

	e.pendingRequestsMu.Lock()
	defer e.pendingRequestsMu.Unlock()

	if e.pendingMsg[proposer] == nil {
		e.pendingMsg[proposer] = make([]*dbft.Message, 0, maxFutureMsgLen)
	}

	if len(e.pendingMsg[proposer]) >= maxFutureMsgLen {
		return
	}

	e.pendingMsg[proposer] = append(e.pendingMsg[proposer], message)
}

func (e *engine) proecessPendingRequest(proposer common.Address) {
	e.pendingRequestsMu.Lock()
	defer e.pendingRequestsMu.Unlock()

	if _, ok := e.pendingMsg[proposer]; !ok {
		return
	}

	for _, message := range e.pendingMsg[proposer] {
		switch message.Code {
		case dbft.MsgPrepare:
			e.Prepare(message)
		case dbft.MsgCommit:
			e.Commit(message)
		default:
			continue
		}
	}

	delete(e.pendingMsg, proposer)
}

func (e *engine) finalizeMessage(msg *dbft.Message, proposal dbft.Proposal) ([]byte, error) {
	var err error
	msg.Address = e.address

	// Add proof of consensus
	msg.CommittedSeal = []byte{}

	if msg.Code == dbft.MsgCommit {
		msg.CommittedSeal, err = e.backend.Sign(proposal.Hash().Bytes())
		if err != nil {
			return nil, err
		}
	}

	// Sign message
	data, err := msg.PayloadNoSig()
	if err != nil {
		return nil, err
	}
	msg.Signature, err = e.backend.Sign(data)
	if err != nil {
		return nil, err
	}

	// Convert to payload
	payload, err := msg.Payload()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (e *engine) isRunning() bool {
	return atomic.LoadInt32(&e.running) == 1
}

func (e *engine) setState(proposer common.Address, state *State) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.states[proposer] = state
}
