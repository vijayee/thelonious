package ethtest

import (
    "fmt"
    "time"
    "os"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "github.com/eris-ltd/eth-go-mods/ethreact"
    "github.com/eris-ltd/eth-go-mods/ethutil"
)   

// environment object for running tests
type Test struct{
    testerFunc string
    genesis string
    blocks int

    reactor *ethreact.ReactorEngine
}

func NewTester(tester, genesis string, blocks int) *Test{
    return &Test{tester, genesis, blocks, nil}
}

func (t *Test) Run(){
    switch(t.testerFunc){
        case "basic":
            t.TestBasic()
        case "run":
            t.Run()
        case "tx":
            t.TestTx()
        case "traverse":
            t.TestTraverseGenesis()
        case "genesis":
            t.TestGenesisAccounts()
        case "genesis-msg":
            t.TestGenesisMsg()
        case "simple-storage":
            t.TestSimpleStorage()
        case "msg-storage":
            t.TestMsgStorage()
        case "validate":
            t.TestValidate()
        case "mining":
            t.TestStopMining()
    }
}

// general tester function on an eth node
// note, you ought to call eth.Start() somewhere in testing()!
func (t *Test) tester(name string, testing func(eth *EthChain), end int){
    eth := NewEth(nil) 
    eth.Config.Mining = true
    eth.Config.GenesisPointer = t.genesis
    eth.Init()

    t.reactor = eth.Ethereum.Reactor()

    testing(eth)
    
    if end > 0{
        time.Sleep(time.Second*time.Duration(end))
        fmt.Println("Stopping...")
        os.Exit(0)
    }
    eth.Ethereum.WaitForShutdown()
}

// general callback function after a block is mined
// fires once an exits
func (t *Test) callback(name string, eth *EthChain, caller func()){
    ch := make(chan ethreact.Event, 1)
    t.reactor.Subscribe("newBlock", ch)
    _ = <- ch
    fmt.Println("####RESPONSE: "+ name +  " ####")
    caller()
    os.Exit(0)
} 

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *ethchain.Block){
    state := block.State()
    it := state.Trie.NewIterator()   
    it.Each(func(key string, value *ethutil.Value) {  
        addr := ethutil.Address([]byte(key))
//        obj := ethstate.NewStateObjectFromBytes(addr, value.Bytes())
        obj := block.State().GetAccount(addr)
        fmt.Println("Address", ethutil.Bytes2Hex([]byte(addr)))
        fmt.Println("\tNonce", obj.Nonce)
        fmt.Println("\tBalance", obj.Balance)
        if true { // only if contract, but how?!
            fmt.Println("\tInit", ethutil.Bytes2Hex(obj.InitCode))
            fmt.Println("\tCode", ethutil.Bytes2Hex(obj.Code))
            fmt.Println("\tStorage:")
            obj.EachStorage(func(key string, val *ethutil.Value){
                val.Decode()
                fmt.Println("\t\t", ethutil.Bytes2Hex([]byte(key)), "\t:\t", ethutil.Bytes2Hex([]byte(val.Str())))
            }) 
        }
    })
}

// print all accounts and storage in the latest block
func PrettyPrintChainAccounts(eth *EthChain){
    curchain := eth.Ethereum.BlockChain()
    block := curchain.CurrentBlock
    PrettyPrintBlockAccounts(block)
}

// compare expected and recovered vals
func check_recovered(expected, recovered string){
    if recovered == expected{
        fmt.Println("Test passed")
    } else{
        fmt.Println("Test failed. Expected", expected, "Recovered", recovered)
    }
}

