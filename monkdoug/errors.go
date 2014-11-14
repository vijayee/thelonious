package monkdoug

import (
    "fmt"
    "github.com/eris-ltd/thelonious/monkutil"
)


type InvalidPermErr string

func InvalidPermError(addr []byte, role string) *InvalidPermErr{
    s := InvalidPermErr(fmt.Sprintf("Invalid permissions err on role %s for adddress %s", role, monkutil.Bytes2Hex(addr)))
    return &s
}

func (self *InvalidPermErr) Error() string{
    return string(*self)
}

type InvalidSigErr string

func InvalidSigError(signer, coinbase []byte) *InvalidSigErr{
    s := InvalidSigErr(fmt.Sprintf("Invalid signature err for coinbase %s signed by %s", monkutil.Bytes2Hex(coinbase), monkutil.Bytes2Hex(signer)))
    return &s
}

func (self *InvalidSigErr) Error() string{
    return string(*self)
}


