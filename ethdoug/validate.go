package ethdoug

import (
    "github.com/ethereum/eth-go/ethcrypto"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/eth-go/ethstate"
    "fmt"

)

var (
    GENDOUG = ethcrypto.Sha3Bin([]byte("the genesis doug"))[12:] //[]byte("\x00"*16 + "DOUG")
    MINERS = "01"
    TXERS = "02"
)

func Validate(addr []byte, state *ethstate.State, role string) bool{
    fmt.Println("validating addr for role", role)
    genDoug := state.GetStateObject(GENDOUG)

    var N string
    switch(role){
        case "tx":
            N = TXERS
        case "miner":
            N = MINERS
        default:
            return false
    }

    caddr := genDoug.GetAddr(ethutil.Hex2Bytes(N))
    c := state.GetOrNewStateObject(caddr.Bytes())

    valid := c.GetAddr(addr)

    return !valid.IsNil()
}

