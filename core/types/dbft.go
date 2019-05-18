// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package types

import (
	"errors"
	"io"

	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/rlp"
)

var (
	// DbftDigest represents a hash of "DBFT practical byzantine fault tolerance"
	// to identify whether the block is from DBFT consensus engine
	// "dpos-bft consensus engine"
	DbftDigest = common.HexToHash("0x64706f732d62667420636f6e73656e73757320656e67696e6500000000000000")

	DbftExtraVanity = 32 // Fixed number of extra-data bytes reserved for validator vanity
	DbftExtraSeal   = 65 // Fixed number of extra-data bytes reserved for validator seal

	// ErrInvalidDbftHeaderExtra is returned if the length of extra-data is less than 32 bytes
	ErrInvalidDbftHeaderExtra = errors.New("invalid dbft header extra-data")

	VoteContract = common.HexToAddress("0x0000000000000000000000000000000000000020")

)
// DbftExtra the extraData ,if it is dbft's mode
type DbftExtra struct {
	Seal          []byte
	CommittedSeal [][]byte
}

// EncodeRLP serializes ist into the Ethereum RLP format.
func (dbft *DbftExtra) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		dbft.Seal,
		dbft.CommittedSeal,
	})
}

// DecodeRLP implements rlp.Decoder, and load the dbft fields from a RLP stream.
func (dbft *DbftExtra) DecodeRLP(s *rlp.Stream) error {
	var dbftExtra struct {
		Seal          []byte
		CommittedSeal [][]byte
	}
	if err := s.Decode(&dbftExtra); err != nil {
		return err
	}
	dbft.Seal, dbft.CommittedSeal = dbftExtra.Seal, dbftExtra.CommittedSeal
	return nil
}

// ExtractDbftExtra extracts all values of the DbftExtra from the header. It returns an
// error if the length of the given extra-data is less than 32 bytes or the extra-data can not
// be decoded.
func ExtractDbftExtra(h *Header) (*DbftExtra, error) {
	if len(h.Extra) < DbftExtraVanity {
		return nil, ErrInvalidDbftHeaderExtra
	}

	var dbftExtra *DbftExtra
	err := rlp.DecodeBytes(h.Extra[DbftExtraVanity:], &dbftExtra)
	if err != nil {
		return nil, err
	}
	return dbftExtra, nil
}

// DbftFilteredHeader returns a filtered header which some information (like seal, committed seals)
// are clean to fulfill the DBFT hash rules. It returns nil if the extra-data cannot be
// decoded/encoded by rlp.
func DbftFilteredHeader(h *Header, keepSeal bool) *Header {
	newHeader := CopyHeader(h)
	dbftExtra, err := ExtractDbftExtra(newHeader)
	if err != nil {
		return nil
	}

	if !keepSeal {
		dbftExtra.Seal = []byte{}
	}
	dbftExtra.CommittedSeal = [][]byte{}

	payload, err := rlp.EncodeToBytes(&dbftExtra)
	if err != nil {
		return nil
	}

	newHeader.Extra = append(newHeader.Extra[:DbftExtraVanity], payload...)

	return newHeader
}
