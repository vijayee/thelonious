
package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "fmt"
    "time"
)


func TestBasic(){
    tester("basic", func(eth *EthChain){
        // eth.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        fmt.Println("the node should be running and mining. if not, there are problems. it will stop in 10 seconds ...")
    }, 10)
}

func TestBig(){
    a := ethutil.NewValue("100000000000")
    fmt.Println("a, bigint", a, a.BigInt())
    // doesnt work! must do: 
    a = ethutil.NewValue(ethutil.Big("100000000000"))
    fmt.Println("a, bigint", a, a.BigInt())
}

func Run(){
    tester("basic", func(eth *EthChain){
        // eth.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
    }, 0)
}

func TestStopMining(){
    tester("mining", func(eth *EthChain){
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        time.Sleep(time.Second*10)
        fmt.Println("stopping mining")
        eth.StopMining()
        time.Sleep(time.Second*10)
        fmt.Println("starting mining again")
        eth.StartMining()        
    }, 30)
}
