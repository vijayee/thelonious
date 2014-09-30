package ethtest

import (
    "fmt"
    "time"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "github.com/eris-ltd/eth-go-mods/ethreact"
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethstate"
    "os"
)   

// environment object for running tests
// one tester obj, will run many tests (sequentially)
type Test struct{
    genesis string
    blocks int
    eth *EthChain
       
    // test specific 
    testerFunc string
    success bool
    err     error

    gendougaddr string //hex address

    reactor *ethreact.ReactorEngine

    failed []string // failed tests
}

func NewTester(tester, genesis string, blocks int) *Test{
    return &Test{testerFunc:tester, genesis:genesis, blocks:blocks, failed:[]string{}}
}

var options = []string{"tx", "genesis-msg", "simple-storage", "msg-storage"} //"traverse"} //, mining, validate

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
        case "listening":
            t.TestStopListening()
        case "blocknum":
            t.TestBlockNum()
        case "restart":
            t.TestRestart()
        case "callstack":
            t.TestCallStack()
        case "all":
            t.eth = NewEth(nil)
            rootdir := t.eth.Config.RootDir
            for _, testf := range options{
                fmt.Println("running next test function: ", testf)
                fmt.Println("delete ~/.ethchain", rootdir)
                os.Remove(rootdir)
                time.Sleep(time.Second*2)
                t.testerFunc = testf
                t.success = false
                t.Run()
                if !t.success{
                    t.failed = append(t.failed, t.testerFunc)
                }
            }
            fmt.Println("failed tests:", len(t.failed), "/", len(options))
            fmt.Println("failed:", t.failed)
    }
    fmt.Println(t.success)
}

// general tester function on an eth node
// note, you ought to call eth.Start() somewhere in testing()!
func (t *Test) tester(name string, testing func(eth *EthChain), end int){
    eth := t.eth
    if eth == nil{
        eth = NewEth(nil) 
        t.eth = eth
    } 

    eth.Config.Mining = true
    eth.Config.GenesisPointer = t.genesis
    ethchain.DougPath = t.genesis // overwrite whatever loads from genesis.json
    ethchain.GENDOUG = []byte("0000000000THISISDOUG") // similarly
    t.gendougaddr = ethutil.Bytes2Hex(ethchain.GENDOUG)
    eth.Init()

    t.reactor = eth.Ethereum.Reactor()

    testing(eth)
    
    if end > 0{
        time.Sleep(time.Second*time.Duration(end))
    }
    fmt.Println("Stopping...")
    eth.Stop()
    t.eth = nil
    time.Sleep(time.Second*3)
}

func (t *Test) callback(name string, eth *EthChain, caller func()) {
    ch := make(chan ethreact.Event, 1)
    t.reactor.Subscribe("newBlock", ch)
    _ = <- ch
    fmt.Println("####RESPONSE: "+ name +  " ####")
    caller()
} 


func PrettyPrintAccount(obj *ethstate.StateObject){
    fmt.Println("Address", ethutil.Bytes2Hex(obj.Address())) //ethutil.Bytes2Hex([]byte(addr)))
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
}

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *ethchain.Block){
    state := block.State()
    it := state.Trie.NewIterator()   
    it.Each(func(key string, value *ethutil.Value) {  
        addr := ethutil.Address([]byte(key))
//        obj := ethstate.NewStateObjectFromBytes(addr, value.Bytes())
        obj := block.State().GetAccount(addr)
        PrettyPrintAccount(obj)
    })
}

// print all accounts and storage in the latest block
func PrettyPrintChainAccounts(eth *EthChain){
    curchain := eth.Ethereum.BlockChain()
    block := curchain.CurrentBlock
    PrettyPrintBlockAccounts(block)
}

// compare expected and recovered vals
func check_recovered(expected, recovered string) bool{
    if recovered == expected{
        fmt.Println("Test passed")
        return true
    } else{
        fmt.Println("Test failed. Expected", expected, "Recovered", recovered)
        return false
    }
}

