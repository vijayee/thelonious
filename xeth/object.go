package xeth

import (
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/state"
)

type Object struct {
	*state.StateObject
}

func (self *Object) StorageString(str string) *monkutil.Value {
	if monkutil.IsHex(str) {
		return self.Storage(monkutil.Hex2Bytes(str[2:]))
	} else {
		return self.Storage(monkutil.RightPadBytes([]byte(str), 32))
	}
}

func (self *Object) StorageValue(addr *monkutil.Value) *monkutil.Value {
	return self.Storage(addr.Bytes())
}

func (self *Object) Storage(addr []byte) *monkutil.Value {
	return self.StateObject.GetStorage(monkutil.BigD(addr))
}
