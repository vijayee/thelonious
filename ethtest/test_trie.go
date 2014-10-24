package ethtest

import (
    //"github.com/eris-ltd/deCerver/chain"
    "github.com/eris-ltd/thelonious/ethutil"
    //"github.com/project-douglas/eth-go/ethchain"
    //"github.com/eris-ltd/thelonious/ethtrie"
    "github.com/eris-ltd/thelonious/ethstate"
    "os"
    "fmt"
)

// traverse the trie
/*
func test_simple_trie(){
    eth := chain.NewEth()
    eth.Init() // necessary for config..

    // create a trie with root reference "abc"
    trie := ethtrie.NewTrie(ethutil.Config.Db, []byte("abc"))
    cache := trie.Cache()
    // add a simple [key, value] entry. first nibble is 6
    trie.Update("aaa", "hello")
    // add a hash reference to a key-val. first nibble is 7
    trie.Update("zaa", "thisistheendoftheworldasweknowitandIfeeljustfine")
    root_node := cache.Get(trie.Root.([]byte)) // returns ethutil.Value
    r, _ := root_node.Val.([]interface{}) // since this is a trie, itll always be a list of 2 or 17, or empty string
    fmt.Println("length of node", len(r)) // 17
    fmt.Println("rootnode", r)
    fmt.Println("6th spot", r[6])
    fmt.Println("7th spot", r[7])
    // now look up the hash value
    node := cache.Get(r[7].([]byte))
    fmt.Println(node.Val)
    n := node.Val.([]interface{})
    fmt.Println("key-value node refered by hash", n)
}


func test_state_trie(){
    eth := chain.NewEth(nil)
    eth.Init()

    gen := eth.Ethereum.BlockChain().Genesis()
    ethchain.AddTestNetFunds(gen)
    state := gen.State()
    trie := state.Trie
    chain.GetAddressList(*trie)
    os.Exit(0)

    /*
    obj := state.GetStateObject(ethutil.Hex2Bytes(("bbbd0256041f7aed3ce278c56ee61492de96d001")))
    fmt.Println("get state obj", obj)

    data := trie.Get(string(ethutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001")))
    obj = ethstate.NewStateObjectFromBytes(ethutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"), []byte(data))

    d, _ := ethutil.Decode([]byte(data), 0)
    dd := ethutil.NewValue(d)
    fmt.Println("d", d)
    fmt.Println("dd", dd)
    fmt.Println(dd.Get(0).Uint, dd.Get(1).BigInt)
    acct := state.GetAccount(ethutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"))
    fmt.Println(acct.Address(), acct.Amount, acct.Nonce, acct.State.Trie)
//    decoder := ethutil.NewValueFromBytes([]byte(data))
    
    //fmt.Println(data)
    //fmt.Println(obj)
}
*/


func test_trie(){

    eth := NewEth(nil)
    eth.Init() // necessary for config..
    c := ethstate.NewStateObject([]byte("hithere"))
    trie := c.State.Trie //ethtrie.New(ethutil.Config.Db, "")
    fmt.Println("trie root before", trie.Root)
    k := "000000000000000000000000000000000000000000000000000000000000000a"
    v := ethutil.NewValue(12)
    c.SetAddr(ethutil.Hex2Bytes(k), v)
    //trie.Update(string(ethutil.Hex2Bytes(k)), string(ethutil.Hex2Bytes(v)))
    fmt.Println("trie root after", trie.Root)

    n := ethutil.NewValue(trie.Root)
    fmt.Println("root node", n)
    fmt.Println("root node val", trie.Cache().Get(n.Bytes()))
    os.Exit(0)
    // if its an array, return n
    // string
        // if its empty, return
        // if its less than 32, return NewValueFromBytes
        // return t.cache.Get(n.Bytes())



    k = "0000000000000000000000000000000000000000000000000000000000000005"
    v = ethutil.NewValueFromBytes(ethutil.Hex2Bytes("a03535000000000000000000000000000000000000000000000000000000000000"))
    c.SetAddr(ethutil.Hex2Bytes(k), v)
    //fmt.Println(ethutil.Hex2Bytes(v))
    //trie.Update(string(ethutil.Hex2Bytes(k)), string(ethutil.Hex2Bytes(v)))
    fmt.Println("trie root after", trie.Root)

    it := trie.NewIterator()
    it.Each(func(key string, val *ethutil.Value) {
        fmt.Println(ethutil.Bytes2Hex([]byte(key)), val)
    })
    os.Exit(0)
}





