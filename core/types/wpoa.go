package types

import (
	"errors"
	"io"
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/rlp"
)

var (
	WPoaDigest = common.BytesToHash([]byte("bcos proof-of-authority consensus"))

	WPoaExtraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	WPoaExtraSeal   = 65 // Fixed number of extra-data bytes reserved for signer seal

	ErrInvalidWPoaHeaderExtra = errors.New("invalid wpoa header extra-data")
)

// the extraData ,if it is wpoa's mode
type WPoaExtra struct {
	Managers           []common.Address
	Signers            []common.Address
	DiscardManagers    []common.Address
	DiscardSigners     []common.Address
}

// EncodeRLP serializes wpoa into the Ethereum RLP format.
func (wpoa *WPoaExtra) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		wpoa.Managers,
		wpoa.Signers,
		wpoa.DiscardManagers,
		wpoa.DiscardSigners,
	})
}

// DecodeRLP implements rlp.Decoder, and load the wpoa fields from a RLP stream.
func (wpoa *WPoaExtra) DecodeRLP(s *rlp.Stream) error {
	var extra struct {
		Manager            []common.Address
		Signer             []common.Address
		DiscardManagers    []common.Address
		DiscardSigners     []common.Address
	}

	if err := s.Decode(&extra); err != nil {
		return err
	}

	wpoa.Managers, wpoa.Signers, wpoa.DiscardManagers, wpoa. DiscardSigners  = extra.Manager, extra.Signer, extra.DiscardManagers, extra.DiscardSigners
	return nil
}

// ExtractWPoaExtra extracts all values of the WPoaExtra from the header. It returns an
// error if the length of the given extra-data is less than 32 bytes or the extra-data can not
// be decoded.
func ExtractWPoaExtra(h *Header) (*WPoaExtra, error) {
	if len(h.Extra) < WPoaExtraVanity {
		return nil, ErrInvalidWPoaHeaderExtra
	}

	var wpoaExtra *WPoaExtra
	err := rlp.DecodeBytes(h.Extra[WPoaExtraVanity:len(h.Extra) - WPoaExtraSeal], &wpoaExtra)
	if err != nil {
		return nil, err
	}
	return wpoaExtra, nil
}