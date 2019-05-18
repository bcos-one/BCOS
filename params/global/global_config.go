package global

import "github.com/bcos-one/BCOS/params"

/**
 * clientIdentifier
 */
var ClientIdentifier string

func init() {
	ClientIdentifier = params.ClientIdentifier
}

// isGeth
func IsGeth() bool {
	if ClientIdentifier == "geth" {
		return true
	}
	return false
}
