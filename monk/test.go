package monk

import (
    "fmt"
    "time"
    "github.com/eris-ltd/thelonious/monkchain"
    "github.com/eris-ltd/thelonious/monkreact"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkstate"
)   

// environment object for running custom tests (ie. not used in `go test`)
// one tester obj, will run many tests (sequentially)
type Test struct{
    genesis string
    blocks int
    mod *MonkModule
       
    // test specific 
    testerFunc string
    success bool
    err     error

    gendougaddr string //hex address

    reactor *monkreact.ReactorEngine

    failed []string // failed tests
}

func NewTester(tester, genesis string, blocks int) *Test{
    return &Test{testerFunc:tester, genesis:genesis, blocks:blocks, failed:[]string{}}
}

// for functions we cant use `go test` on
func (t *Test) Run(){
    switch(t.testerFunc){
        case "basic":
            t.TestBasic()
        case "run":
            t.TestRun()
        case "genesis":
            t.TestGenesisAccounts()
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
        case "maxgas":
            t.TestMaxGas()
        case "state":
            t.TestState()
        case "compress":
            t.TestCompression()
    }
    fmt.Println(t.success)
}

// general tester function on an eth node
// note, you ought to call eth.Start() somewhere in testing()!
func (t *Test) tester(name string, testing func(mod *MonkModule), end int){
    mod := t.mod
    if mod == nil{
        mod = NewMonk(nil) 
        t.mod = mod
    } 
    mod.ReadConfig("eth-config.json")
    mod.monk.Config.Mining = true
    mod.monk.Config.DbName = "tests/"+name
    monkchain.DougPath = t.genesis // overwrite whatever loads from genesis.json
    monkchain.GENDOUG = []byte("0000000000THISISDOUG") // similarly
    t.gendougaddr = monkutil.Bytes2Hex(monkchain.GENDOUG)
    mod.Init()

    t.reactor = mod.monk.Ethereum.Reactor()
    testing(mod)
    
    if end > 0{
        time.Sleep(time.Second*time.Duration(end))
    }
    mod.Shutdown()
    t.mod = nil
    time.Sleep(time.Second*3)
}

// called by `go test` functions
func tester(name string, testing func(mod *MonkModule), end int){
    mod := NewMonk(nil) 
    mod.ReadConfig("eth-config.json")
    mod.monk.Config.Mining = true
    mod.monk.Config.DbName = "tests/"+name
    //TODO: genesis
    //monkchain.DougPath = t.genesis // overwrite whatever loads from genesis.json
    monkchain.GENDOUG = []byte("0000000000THISISDOUG") // similarly
    mod.Init()

    testing(mod)
    
    if end > 0{
        time.Sleep(time.Second*time.Duration(end))
    }
    mod.Shutdown()
    time.Sleep(time.Second*3)
}

func callback(name string, mod *MonkModule, caller func()) {
    ch := make(chan monkreact.Event, 1)
    mod.monk.Ethereum.Reactor().Subscribe("newBlock", ch)
    _ = <- ch
    fmt.Println("####RESPONSE: "+ name +  " ####")
    caller()
} 

func (t *Test) callback(name string, mod *MonkModule, caller func()) {
    ch := make(chan monkreact.Event, 1)
    t.reactor.Subscribe("newBlock", ch)
    _ = <- ch
    fmt.Println("####RESPONSE: "+ name +  " ####")
    caller()
} 


func PrettyPrintAccount(obj *monkstate.StateObject){
    fmt.Println("Address", monkutil.Bytes2Hex(obj.Address())) //monkutil.Bytes2Hex([]byte(addr)))
    fmt.Println("\tNonce", obj.Nonce)
    fmt.Println("\tBalance", obj.Balance)
    if true { // only if contract, but how?!
        fmt.Println("\tInit", monkutil.Bytes2Hex(obj.InitCode))
        fmt.Println("\tCode", monkutil.Bytes2Hex(obj.Code))
        fmt.Println("\tStorage:")
        obj.EachStorage(func(key string, val *monkutil.Value){
            val.Decode()
            fmt.Println("\t\t", monkutil.Bytes2Hex([]byte(key)), "\t:\t", monkutil.Bytes2Hex([]byte(val.Str())))
        }) 
    }
}

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *monkchain.Block){
    state := block.State()
    it := state.Trie.NewIterator()   
    it.Each(func(key string, value *monkutil.Value) {  
        addr := monkutil.Address([]byte(key))
//        obj := monkstate.NewStateObjectFromBytes(addr, value.Bytes())
        obj := block.State().GetAccount(addr)
        PrettyPrintAccount(obj)
    })
}

// print all accounts and storage in the latest block
func PrettyPrintChainAccounts(mod *MonkModule){
    curchain := mod.monk.Ethereum.BlockChain()
    block := curchain.CurrentBlock
    PrettyPrintBlockAccounts(block)
}

// compare expected and recovered vals
func check_recovered(expected, recovered string) bool{
    if monkutil.Coerce2Hex(recovered) == monkutil.Coerce2Hex(expected){
        fmt.Println("Test passed")
        return true
    } else{
        fmt.Println("Test failed. Expected", expected, "Recovered", recovered)
        return false
    }
}

