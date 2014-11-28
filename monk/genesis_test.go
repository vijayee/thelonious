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
        mod.Init()
        mod.Start()
        callback("traverse_to_genesis", mod, func(){
            curchain := mod.monk.thelonious.ChainManager()
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
// TODO: this has gotten to complicated for this function
// to be able to handle. need to work on testing for this...
func _TestGenesisMsg(t *testing.T){
    mod := NewMonk(nil) 
    mod.ReadConfig("eth-config.json")
    mod.monk.config.Mining = false
    mod.monk.config.DbName = "tests/genesis-msg"

    g := mod.LoadGenesis(mod.Config.GenesisConfig)

    g.Difficulty = 3 // so we always mine quickly
    g.DougPath = "tests/fake-doug-msg.lll"
    g.ModelName = "yes"
    mod.SetGenesis(g)
    fmt.Println(mod.GenesisConfig.DougPath)
    mod.Init()
    mod.Start()

    key := "0x21"
    value := "0x400"
    gendoug := monkutil.Bytes2Hex([]byte(g.Address))
    mod.Msg(gendoug, []string{key, value})
    callback("genesis msg", mod, func(){
        recovered := "0x"+ mod.StorageAt(gendoug, key)
        if !check_recovered(value, recovered){
            t.Error("got:", recovered, "expected:", value)
        }
    })
}

// follow the prevhashes back to genesis
func traverse_to_genesis(curchain *monkchain.ChainManager, curblock *monkchain.Block) *monkchain.Block{
    prevhash := curblock.PrevHash
    prevblock := curchain.GetBlock(prevhash)
    fmt.Println("prevblock", prevblock)
    if prevblock == nil{
        return curblock
    }else{
        return traverse_to_genesis(curchain, prevblock)
    }
}

