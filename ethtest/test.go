package ethtest

import (
    "fmt"
    "time"
    "os"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "github.com/eris-ltd/eth-go-mods/ethreact"
    "github.com/eris-ltd/eth-go-mods/ethutil"
)   

// general tester function on an eth node
// note, you ought to call eth.Start() somewhere in testing()!
func tester(name string, testing func(eth *EthChain), end int){
    eth := NewEth(nil) 
    eth.Config.Mining = true
    eth.Init()

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
func callback(name string, eth *EthChain, caller func()){
    reactor := eth.Ethereum.Reactor()
    ch := make(chan ethreact.Event, 1)
    reactor.Subscribe("newBlock", ch)
    _ = <- ch
    caller()
    os.Exit(0)
} 

func pretty_print_accounts_block(block *ethchain.Block){
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

func pretty_print_accounts_chain(eth *EthChain){
    curchain := eth.Ethereum.BlockChain()
    block := curchain.CurrentBlock
    pretty_print_accounts_block(block)
}
