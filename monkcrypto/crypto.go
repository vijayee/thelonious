package monkcrypto

import (
	//"code.google.com/p/go.crypto/sha3"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/obscuren/sha3"
)

func Sha3Bin(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b []byte, nonce uint64) []byte {
	return Sha3Bin(monkutil.NewValue([]interface{}{b, nonce}).Encode())[12:]
}