package ethtest

import (
    "strings"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/eth-go/ethtrie"
    //"reflect"
    "log"
    "fmt"
)

func Hex2Nibbles(h string) []int{
    base := "0123456789abcdef"
    nibs := make([]int, 0)
    for _, v := range h{
        nibs = append(nibs, strings.IndexByte(base, byte(v)))
    }
    return nibs
}

func Nibbles2Hex(nibs []int) string{
    base := "0123456789abcdef"
    s := ""
    for _, k := range nibs{
        h := base[k]
        s += string(h)
    }
    return s
}

func StripHP(k string) string{
    h := ethutil.Bytes2Hex([]byte(k))
    nibs := Hex2Nibbles(h)
    var n []int
    if nibs[0] == 1 || nibs[0] == 3{
        n = nibs[1:]
    } else{
        n = nibs[2:]
    }
    r := Nibbles2Hex(n)
    //fmt.Println("strip hp initial hex:", h, "nibs:",  nibs, "new nibs", n, "final hex", r)
    return r

}

func IsTerminator(k string) bool{
    h := ethutil.Bytes2Hex([]byte(k))
    nibs := Hex2Nibbles(h)
    //fmt.Println("termination input:", k, "byte input:", []byte(k), "hex:",  h, "nibs:", nibs)
    if nibs[0] >= 2{
        return true
    }
    return false
}

// traverse the trie, accumulating keys
func ListKeysRecursive(trie ethtrie.Trie, node interface{}, keys *[]string, prefix []byte){
    cache := trie.Cache()
    //typ := reflect.TypeOf(node)
    // either string or list
    if _, ok := node.(string); ok{
        s := node.(string)
        if s != ""{
            log.Fatal("impossible! a string thats not empty?!", s)
        }
    } else if b, ok := node.([]byte); ok{
        //fmt.Println("byte node:", node)
        new_node := cache.Get(b)
        ListKeysRecursive(trie, new_node, keys, prefix)

    } else if _, ok := node.([]interface{}); ok {
        //fmt.Println("node:", node, typ)
        n, _ := node.([]interface{})
        //fmt.Println("length node array", len(n))
        // either len 2 or 17
        if len(n) == 2{
            k := ""

            // awful...
            if k, ok = n[0].(string); !ok{
                if kk, ok := n[0].([]uint8); !ok{
                    k = string(n[0].(uint8))
                } else{
                    k = string(kk)
                }
            }
            if IsTerminator(k){
                // this is a key-val where we have the rest of the key and the actual value
                kk := StripHP(k)
                key := append(prefix, []byte(kk)...)
                //fmt.Println("FOUND TERMINATOR!!!!", ethutil.Bytes2Hex([]byte(k)), kk, string(prefix), string(key))
                *keys = append(*keys, string(key))
            } else {
                // key-val where the val is a hash to lookup
                kk := StripHP(k)
                prefix = append(prefix, []byte(kk)...)
                v := n[1].([]byte)
                //fmt.Println("MOVING ON TO HASH", "prefix:", string(prefix), "value:", ethutil.Bytes2Hex(v))
                new_node := cache.Get(v)
                ListKeysRecursive(trie, new_node, keys, prefix)
            }
            
        } else if len(n) == 17{
            for i, _ := range n{
                new_node := n[i]
                if i < 16{
                    p := append(prefix, "0123456789abcdef"[i])
                    ListKeysRecursive(trie, new_node, keys, p)
                } else{
                    ListKeysRecursive(trie, new_node, keys, prefix)
                }
            }

        } else {
            log.Fatal("impossible node array size")
        }

    } else if n, ok := node.(*ethutil.Value); ok{
            ListKeysRecursive(trie, n.Val, keys, prefix)
    } else{
        //log.Fatal("what type?", typ)
        return
    }

}

// get all keys in the trie
func GetAddressList(trie ethtrie.Trie) []string{
    addrs := new([]string)
    cache := trie.Cache()
    node := cache.Get(trie.Root.([]byte)) // returns ethutil.Value
    ListKeysRecursive(trie, node.Val, addrs, []byte{})
    fmt.Println(*addrs)
    return *addrs

}

