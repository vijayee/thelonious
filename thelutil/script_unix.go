// +build !windows

package thelutil

import "github.com/ethereum/serpent-go"

// General compile function
func Compile(script string, silent bool) (ret []byte, err error) {
	if len(script) > 2 {
		byteCode, err := serpent.Compile(script)
		if err != nil {
			return nil, err
		}

		return byteCode, nil
	}

	return nil, nil
}

// strings and hex only
func PackTxDataArgs2(args ...string) []byte {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if len(s) > 1 && s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return ret
}

// strings and hex only
func PackTxDataArgs(args ...string) []byte {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			ret = append(ret, RightPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return ret
}
