package monk

import (
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkchain"
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
    tester("traverse to genesis", func(mod *MonkModule){
        mod.Start()
        callback("traverse_to_genesis", mod, func(){
            curchain := mod.monk.ethereum.BlockChain()
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
    dp := monkchain.DougPath
    monkchain.DougPath = "tests/fake-doug-msg.lll"
    tester("genesis msg", func(mod *MonkModule){
        md := monkchain.Model
        monkchain.Model = nil // disable permissions model so we can transact
        mod.Start()
            key := "0x21"
            value := "0x400"
            gendoug := monkutil.Bytes2Hex(monkchain.GENDOUG)
            mod.Msg(gendoug, []string{key, value})
            callback("genesis msg", mod, func(){
                recovered := "0x"+ mod.StorageAt(gendoug, key)
                if !check_recovered(value, recovered){
                    fmt.Println("got:", recovered, "expected:", value)
                }
            })
            monkchain.Model = md
            monkchain.DougPath = dp
    }, 0)
}

// follow the prevhashes back to genesis
func traverse_to_genesis(curchain *monkchain.BlockChain, curblock *monkchain.Block) *monkchain.Block{
    prevhash := curblock.PrevHash
    prevblock := curchain.GetBlock(prevhash)
    fmt.Println("prevblock", prevblock)
    if prevblock == nil{
        return curblock
    }else{
        return traverse_to_genesis(curchain, prevblock)
    }
}

