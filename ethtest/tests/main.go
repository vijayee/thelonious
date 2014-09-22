package main

import (
    "github.com/eris-ltd/eth-go-mods/ethtest"
    "flag"
    "os"
)   

var (
    tester = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage")
)


// due to instability in Ethereum.Stop(), must run these one at a time
// for now ...
func main(){
    flag.Parse()
    if *tester == ""{
        flag.Usage()
        os.Exit(0)
    }

    switch(*tester){
        case "basic":
            ethtest.TestBasic()
        case "run":
            ethtest.Run()
        case "tx":
            ethtest.TestTx()
        case "traverse":
            ethtest.TestTraverseGenesis()
        case "genesis":
            ethtest.TestGenesisAccounts()
        case "genesis-msg":
            ethtest.TestGenesisMsg()
        case "get-storage":
            ethtest.TestGetStorage()
        case "msg-storage":
            ethtest.TestMsgStorage()
        case "validate":
            ethtest.TestValidate()
        case "mining":
            ethtest.TestStopMining()
    }
}
