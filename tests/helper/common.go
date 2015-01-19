package helper

import "github.com/eris-ltd/new-thelonious/monkutil"

func FromHex(h string) []byte {
	if monkutil.IsHex(h) {
		h = h[2:]
	}

	return monkutil.Hex2Bytes(h)
}
