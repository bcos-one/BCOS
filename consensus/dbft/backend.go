package dbft

import "github.com/bcos-one/BCOS/common"

type Backend interface {
	// Verify verifies the proposal.
	Verify(Proposal) (error)

	// Validators returns the validator set
	Validators(proposal Proposal) Validators

	// Sign signs input data with the backend's private key
	Sign([]byte) ([]byte, error)

	// CheckSignature verifies the signature by checking if it's signed by
	// the given validator
	CheckSignature(data []byte, addr common.Address, sig []byte) error

	// Commit delivers an approved proposal to backend.
	// The delivered proposal will be put into blockchain.
	Commit(proposal Proposal, seals [][]byte) error
}