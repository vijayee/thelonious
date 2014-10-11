package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "os"
    "fmt"
    "testing"
)

/*
   TestTraverseGenesis
   TestGenesisMsg

   TestValidate 
   TestGenesisAccounts
*/

// doesn't start up a node, just loads from db and traverses to genesis
func TestTraverseGenesis(t *testing.T){
    tester2("traverse to genesis", func(eth *EthChain){
        eth.Start()
        callback2("traverse_to_genesis", eth, func(){
            curchain := eth.Ethereum.BlockChain()
            curblock := curchain.CurrentBlock
            gen_tr := traverse_to_genesis(curchain, curblock)
            gen := curchain.Genesis()
            if !check_recovered(gen.String(), gen_tr.String()){
                t.Error("got:", gen_tr.String(), "expected:", gen.String())
            }
        })
    }, 0)
}


// test sending a message to the genesis doug
func TestGenesisMsg(t *testing.T){
    //t.genesis = "lll/fake-doug-msg.lll"
    ethchain.DougPath = "tests/fake-doug-msg.lll"
    tester2("genesis msg", func(eth *EthChain){
        ethchain.Model = nil // disable permissions model so we can transact
        eth.Start()
            key := "0x21"
            value := "0x400"
            gendoug := ethutil.Bytes2Hex(ethchain.GENDOUG)
            eth.Msg(gendoug, []string{key, value})
            callback2("get storage", eth, func(){
                recovered := "0x"+ eth.GetStorageAt(gendoug, key)
                if !check_recovered(value, recovered){
                    fmt.Println("got:", recovered, "expected:", value)
                }
            })
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

