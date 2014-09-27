package main

import (
    "github.com/eris-ltd/eth-go-mods/ethtest"
    "flag"
    "os"
)   

var (
    tester = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage or all")
    genesis = flag.String("g", "", "pick a genesis functin:")
    blocks = flag.Int("n", 10, "num blocks to wait before shutdown")
)


func main(){
    flag.Parse()
    if *tester == ""{
        flag.Usage()
        os.Exit(0)
    }

    T := ethtest.NewTester(*tester, *genesis, *blocks)
    T.Run()
}

