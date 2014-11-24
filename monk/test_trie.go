package monk

import (
    //"github.com/eris-ltd/decerver/chain"
    "github.com/eris-ltd/thelonious/monkutil"
    //"github.com/project-douglas/eth-go/monkchain"
    //"github.com/eris-ltd/thelonious/monktrie"
    "github.com/eris-ltd/thelonious/monkstate"
    "os"
    "fmt"
)

// traverse the trie
/*
func test_simple_trie(){
    eth := chain.NewEth()
    eth.Init() // necessary for config..

    // create a trie with root reference "abc"
    trie := monktrie.NewTrie(monkutil.Config.Db, []byte("abc"))
    cache := trie.Cache()
    // add a simple [key, value] entry. first nibble is 6
    trie.Update("aaa", "hello")
    // add a hash reference to a key-val. first nibble is 7
    trie.Update("zaa", "thisistheendoftheworldasweknowitandIfeeljustfine")
    root_node := cache.Get(trie.Root.([]byte)) // returns monkutil.Value
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

    gen := eth.Ethereum.ChainManager().Genesis()
    monkchain.AddTestNetFunds(gen)
    state := gen.State()
    trie := state.Trie
    chain.GetAddressList(*trie)
    os.Exit(0)

    /*
    obj := state.GetStateObject(monkutil.Hex2Bytes(("bbbd0256041f7aed3ce278c56ee61492de96d001")))
    fmt.Println("get state obj", obj)

    data := trie.Get(string(monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001")))
    obj = monkstate.NewStateObjectFromBytes(monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"), []byte(data))

    d, _ := monkutil.Decode([]byte(data), 0)
    dd := monkutil.NewValue(d)
    fmt.Println("d", d)
    fmt.Println("dd", dd)
    fmt.Println(dd.Get(0).Uint, dd.Get(1).BigInt)
    acct := state.GetAccount(monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"))
    fmt.Println(acct.Address(), acct.Amount, acct.Nonce, acct.State.Trie)
//    decoder := monkutil.NewValueFromBytes([]byte(data))
    
    //fmt.Println(data)
    //fmt.Println(obj)
}
*/


func test_trie(){

    eth := NewMonk(nil)
    eth.Init() // necessary for config..
    c := monkstate.NewStateObject([]byte("hithere"))
    trie := c.State.Trie //monktrie.New(monkutil.Config.Db, "")
    fmt.Println("trie root before", trie.Root)
    k := "000000000000000000000000000000000000000000000000000000000000000a"
    v := monkutil.NewValue(12)
    c.SetAddr(monkutil.Hex2Bytes(k), v)
    //trie.Update(string(monkutil.Hex2Bytes(k)), string(monkutil.Hex2Bytes(v)))
    fmt.Println("trie root after", trie.Root)

    n := monkutil.NewValue(trie.Root)
    fmt.Println("root node", n)
    fmt.Println("root node val", trie.Cache().Get(n.Bytes()))
    os.Exit(0)
    // if its an array, return n
    // string
        // if its empty, return
        // if its less than 32, return NewValueFromBytes
        // return t.cache.Get(n.Bytes())



    k = "0000000000000000000000000000000000000000000000000000000000000005"
    v = monkutil.NewValueFromBytes(monkutil.Hex2Bytes("a03535000000000000000000000000000000000000000000000000000000000000"))
    c.SetAddr(monkutil.Hex2Bytes(k), v)
    //fmt.Println(monkutil.Hex2Bytes(v))
    //trie.Update(string(monkutil.Hex2Bytes(k)), string(monkutil.Hex2Bytes(v)))
    fmt.Println("trie root after", trie.Root)

    it := trie.NewIterator()
    it.Each(func(key string, val *monkutil.Value) {
        fmt.Println(monkutil.Bytes2Hex([]byte(key)), val)
    })
    os.Exit(0)
}





