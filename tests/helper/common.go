package helper

import "github.com/eris-ltd/new-thelonious/thelutil"

func FromHex(h string) []byte {
	if thelutil.IsHex(h) {
		h = h[2:]
	}

	return thelutil.Hex2Bytes(h)
}
