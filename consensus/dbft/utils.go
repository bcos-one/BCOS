package dbft

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/crypto"
	"github.com/bcos-one/BCOS/crypto/sha3"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/rlp"
)

func RLPHash(v interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, v)
	hw.Sum(h[:0])
	return h
}

// GetSignatureAddress gets the signer address from the signature
func GetSignatureAddress(data []byte, sig []byte) (common.Address, error) {
	// 1. Keccak data
	hashData := crypto.Keccak256([]byte(data))
	// 2. Recover public key
	pubkey, err := crypto.SigToPub(hashData, sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pubkey), nil
}

func CheckValidatorSignature(data []byte, sig []byte) (common.Address, error) {
	signer, err := GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("Failed to get signer address", "err", err)
		return common.Address{}, err
	}


	return signer, nil
}

