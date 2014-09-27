package ethdoug

import (
    "github.com/eris-ltd/eth-go-mods/ethstate"
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "fmt"
)


var (
    GENDOUG []byte = nil // dougs address
    MINERS = "01"
    TXERS = "02"
    CREATE = "03"
)


// use genesis block to validate addr's role
// TODO: bring up to date
func DougValidate(addr []byte, state *ethstate.State, role string) bool{
    if GENDOUG == nil{
        return true
    }
    //fmt.Println("validating addr for role", role)
    genDoug := state.GetStateObject(GENDOUG)

    var N string
    switch(role){
        case "tx":
            N = TXERS
        case "miner":
            N = MINERS
        case "create":
            N = CREATE
        default:
            return false
    }

    caddr := genDoug.GetStorage(ethutil.BigD(ethutil.Hex2Bytes(N)))
    c := state.GetOrNewStateObject(caddr.Bytes())

    valid := c.GetStorage(ethutil.BigD(addr))
    return !valid.IsNil()
}

type InvalidPermErr struct{
    Message string
    Addr string
}

func InvalidPermError(addr, role string) *InvalidPermErr{
    return &InvalidPermErr{Message: fmt.Sprintf("Invalid permissions err on role %s for adddress %s", role, addr), Addr:addr}
}

func (self *InvalidPermErr) Error() string{
    return self.Message
}
