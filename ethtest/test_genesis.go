package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "os"
    "fmt"
)

/*
   TestTraverseGenesis
   TestGenesisMsg

   TestValidate 
   TestGenesisAccounts
*/

// this one will be in flux for a bit
// test the validation..
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
    }, 0)
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
            t.success = check_recovered(gen.String(), gen_tr.String())
        })
    }, 0)
}

// doesn't start up a node, just loads from db and traverses to genesis
func (t *Test) TestMaxGas(){
    t.tester("max gas", func(eth *EthChain){
        //eth.Start()
        v := ethchain.DougValue("maxgas", "values", eth.Ethereum.BlockChain().CurrentBlock.State())
        fmt.Println(v)
        os.Exit(0)
    }, 0)
}

// test sending a message to the genesis doug
func (t *Test) TestGenesisMsg(){
    //t.genesis = "lll/fake-doug-msg.lll"
    t.genesis = "tests/fake-doug-msg.lll"
    t.tester("genesis msg", func(eth *EthChain){
        eth.Start()
            key := "0x21"
            value := "0x400"
            eth.Msg(t.gendougaddr, []string{key, value})
            t.callback("get storage", eth, func(){
                recovered := "0x"+ eth.GetStorageAt(t.gendougaddr, key)
                t.success = check_recovered(value, recovered)
            })
    }, 0)
}

// print the genesis state
func (t *Test) TestGenesisAccounts(){
    t.tester("genesis contract", func(eth *EthChain){
        curchain := eth.Ethereum.BlockChain()
        block := curchain.CurrentBlock
        PrettyPrintBlockAccounts(block)
        os.Exit(0)
    }, 0)
}

// print the genesis state
func (t *Test) TestBlockNum(){

    t.tester("block num", func(eth *EthChain){
        curchain := eth.Ethereum.BlockChain()
        block := curchain.CurrentBlock
        fmt.Println(curchain.LastBlockNumber)
        fmt.Println(block.Number)
        fmt.Println(curchain.Genesis().Number)
        os.Exit(0)
    }, 0)
}


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

