package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    //"github.com/eris-ltd/eth-go-mods/ethstate"
    "os"
    "fmt"
)

// compare the genesis block from following prevhash with that from calling blockchain.Genesis()
// they are different, since Genesis() does not include updates to state due to testnets (inital allocations)
// Genesis() is strictly the genesis block from the whitepaper

// follow the prevhashes back to genesis
func traverse_to_genesis(curchain *ethchain.BlockChain, curblock *ethchain.Block) *ethchain.Block{
    prevhash := curblock.PrevHash
    prevblock := curchain.GetBlock(prevhash)
    fmt.Println("prevblock", prevblock)
    if prevblock == nil{
        return curblock
    }else{
        return traverse_to_genesis(curchain, prevblock)
    }
}

func (t *Test) TestValidate(){
    t.tester("validate", func(eth *EthChain){
        PrettyPrintChainAccounts(eth)
        gen := eth.Ethereum.BlockChain().Genesis()
        a1 := ethutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001")
        a2 := ethutil.Hex2Bytes("b9398794cafb108622b07d9a01ecbed3857592d5")
        a3 := ethutil.Hex2Bytes("cced0756041f7aed3ce278c56ee638bade96d001")
        fmt.Println(ethchain.DougValidate(a1, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a2, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a3, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a1, gen.State(), "miner"))
        fmt.Println(ethchain.DougValidate(a2, gen.State(), "miner"))
        fmt.Println(ethchain.DougValidate(a3, gen.State(), "miner"))
    }, 10)
}

// doesn't start up a node, just loads from db and traverses to genesis
func (t *Test) TestTraverseGenesis(){
    t.tester("traverse to genesis", func(eth *EthChain){
        eth.Start()
        t.callback("traverse_to_genesis", eth, func(){
            curchain := eth.Ethereum.BlockChain()
            curblock := curchain.CurrentBlock
            gen_tr := traverse_to_genesis(curchain, curblock)
            gen := curchain.Genesis()
            check_recovered(gen.String(), gen_tr.String())
        })
    }, 10)
}

// test sending a message to the genesis doug
func (t *Test) TestGenesisMsg(){
    t.tester("genesis msg", func(eth *EthChain){
        eth.Start()
            addr := ethchain.GENDOUG //"2b36a39892af8e0b63042d8cead877517cd62c48"
            //eth.SetCursor(2)
            eth.Msg(addr, []string{"55"})
            t.callback("get storage", eth, func(){
                fmt.Println("####RESPONSE####")
                fmt.Println(eth.GetStorageAt(addr, "5"))
                storage := eth.GetStorage(addr)
                fmt.Println(storage)
                PrettyPrintChainAccounts(eth)
            })
            os.Exit(0)
    }, 10)

}


// add a contract account to the genesis block
func (t *Test) TestGenesisAccounts(){
    t.tester("genesis contract", func(eth *EthChain){
        //eth.Start()
        //t.callback"genesis", eth, func(){
            curchain := eth.Ethereum.BlockChain()
            block := curchain.CurrentBlock
            //block = curchain.Genesis()
            PrettyPrintBlockAccounts(block)
            os.Exit(0)
            fmt.Println(eth.GetStorageAt("a92e33077d317c1d838f7270d9fe3e1c4399f997", "7b"))
            latest := eth.Ethereum.BlockChain().CurrentBlock
            acct := latest.State().GetAccount(ethutil.Hex2Bytes("a92e33077d317c1d838f7270d9fe3e1c4399f997"))
            fmt.Println(acct)
            os.Exit(0)
         //})
    }, 10)

    /*
        //gen := traverse_to_genesis(*(eth.Ethereum.BlockChain()), *latest)
       // fmt.Println(&gen)
        //gen = *(eth.Ethereum.BlockChain().Genesis())
        //ethchain.AddTestNetFunds(&gen)
        //fmt.Println(&gen)
        state := latest.State()
        trie := state.Trie
        addrs := (chain.GetAddressList(*trie))
        fmt.Println(addrs)
        for _, ac := range addrs{
            account := state.GetAccount(ethutil.Hex2Bytes(ac))
            fmt.Println("account!", account.Address(), account.Amount)
            //account.EachStorage(func(key string, val *ethutil.Value){ fmt.Printf("key: %x, \t val %x", key, val)})
            fmt.Println(ethutil.Bytes2Hex(account.Address()), account.GetStorage(ethutil.Big("123")))
        }

    //fmt.Println(eth.Peth.GetStateObject("a92e33077d317c1d838f7270d9fe3e1c4399f997").GetStorage("123"))

        account := state.GetAccount(ethutil.Hex2Bytes("a92e33077d317c1d838f7270d9fe3e1c4399f997"))
        fmt.Println(account.GetStorage(ethutil.Big("123")))

        acc := state.GetStateObject(ethutil.Hex2Bytes("a92e33077d317c1d838f7270d9fe3e1c4399f997"))
        fmt.Println(acc)
        fmt.Println(acc.GetStorage(ethutil.Big("0x7b")))
        */
        //fmt.Println(eth.GetStorageAt("a92e33077d317c1d838f7270d9fe3e1c4399f997", "7b"))
        os.Exit(0)
        

}

/*

    value := ethutil.Big("100")
    gas := ethutil.Big("10000")
    price := ethutil.Big("10000")
    script := []byte("")
    tx := ethchain.NewContractCreationTx(value, gas, price, script)
    priv := eth.FetchPriv()
    tx.Sign([]byte(priv))
    addr := tx.CreationAddress()
    fmt.Println(addr)

    tx := ethchain.NewContractCreationTx(value, gas, price, script)
    priv := eth.FetchPriv()
    tx.Sign([]byte(priv))
    addr := tx.CreationAddress()
    fmt.Println(addr)

*/

