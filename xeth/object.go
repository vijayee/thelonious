package xeth

import (
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/state"
)

type Object struct {
	*state.StateObject
}

func (self *Object) StorageString(str string) *thelutil.Value {
	if thelutil.IsHex(str) {
		return self.Storage(thelutil.Hex2Bytes(str[2:]))
	} else {
		return self.Storage(thelutil.RightPadBytes([]byte(str), 32))
	}
}

func (self *Object) StorageValue(addr *thelutil.Value) *thelutil.Value {
	return self.Storage(addr.Bytes())
}

func (self *Object) Storage(addr []byte) *thelutil.Value {
	return self.StateObject.GetStorage(thelutil.BigD(addr))
}
