package monk

import (
    "fmt"
    "math/big"
    "path"
    "io/ioutil"
    "os"
    "time"
    "testing"
    "github.com/eris-ltd/thelonious/monkutil"
)

/*
    TestSimpleStorage
    TestMsgStorage
    TestTx
    TestManyTx
*/


// contract that stores a single value during init
func TestSimpleStorage(t *testing.T){
    tester("simple storage", func(mod *MonkModule){
        mod.Init()
        // set up test parameters and code
        key := "0x5"
        value := "0x400"
        code := fmt.Sprintf(`
            {
                ;; store a value
                [[%s]]%s
            }
        `, key, value)
        fmt.Println("Code:\n", code)
        // write code to file and deploy
        c := "tests/simple-storage.lll"
        p := path.Join(mod.monk.config.ContractPath, c)
        err := ioutil.WriteFile(p, []byte(code), 0644)
        if err != nil{
            fmt.Println("write file failed", err)
            os.Exit(0)
        }
        contract_addr, err := mod.Script(p, "lll")
        if err != nil{
            t.Fatal(err)
        }
        mod.Start()
        // callback when block is mined
        callback("simple storage", mod, func(){
            recovered := "0x" + mod.StorageAt(contract_addr, key)
            result := check_recovered(value, recovered)
            if !result{
                t.Error("got:", recovered, "expected:", value)
            }
        })
    }, 0)
}

// test a simple key-value store contract
func TestMsgStorage(t *testing.T){
    tester("msg storage", func(mod *MonkModule){
        mod.Init()
        contract_addr, err := mod.Script(path.Join(mod.Config.ContractPath, "tests/keyval.lll"), "lll")
        if err != nil{
            t.Fatal(err)
        }
        mod.Start()
        callback("deploy key-value", mod, func(){
            key := "0x21"
            value := "0x400"
            time.Sleep(time.Nanosecond) // needed or else subscribe channels block and are skipped ... TODO: why?!
            fmt.Println("contract account:", mod.Account(contract_addr))
            fmt.Println("my account:", mod.Account(mod.ActiveAddress()))

            mod.Msg(contract_addr, []string{key, value})
            callback("test key-value", mod, func(){
                start := time.Now()
                recovered := "0x"+mod.StorageAt(contract_addr, key)
                dif := time.Since(start)
                fmt.Println("get storage took", dif)
                result := check_recovered(value, recovered)
                if !result{
                    t.Error("got:", recovered, "expected:", value)
                }
            })
        })

    }, 0)
}

// test simple tx
func TestTx(t *testing.T){
    tester("basic tx", func(mod *MonkModule){
        mod.Init()
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := monkutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := mod.monk.pipe.Balance(addr_bytes)
        //mod.SetCursor(0)
        start := time.Now()
        mod.Tx(addr, amount)
        dif := time.Since(start)
        fmt.Println("sending one tx took", dif)
        mod.Start()
        callback("get balance", mod, func(){
            new_balance := mod.monk.pipe.Balance(addr_bytes)
            old := old_balance.BigInt()
            am := monkutil.Big(amount)
            n := new(big.Int)
            n.Add(old, am)
            newb := monkutil.BigD(new_balance.Bytes())
            //t.success = check_recovered(n.String(), newb.String())
            result := check_recovered(n.String(), newb.String())
            if !result{
                t.Error("got:", newb.String(), "expected:", n.String())
            }
        })
        //mod.Thelonious.WaitForShutdown()
    }, 0)
}

func TestManyTx(t *testing.T){
    tester("many tx", func(mod *MonkModule){
        mod.Init()
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := monkutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := mod.monk.pipe.Balance(addr_bytes)
        N := 1000
        //mod.SetCursor(0)
        start := time.Now()
        for i:=0; i<N; i++{
            mod.Tx(addr, amount)
        }
        end := time.Since(start)
        fmt.Printf("sending %d txs took %s\n", N, end)
        mod.Start()
        callback("get balance", mod, func(){
            new_balance := mod.monk.pipe.Balance(addr_bytes)
            old := old_balance.BigInt()
            am := monkutil.Big(amount)
            mult := big.NewInt(int64(N))
            n := new(big.Int)
            n.Add(old, n.Mul(mult, am))
            newb := monkutil.BigD(new_balance.Bytes())
            results := check_recovered(n.String(), newb.String())
            if !results{
                t.Error("got:", newb.String(), "expected:", n.String())
            }
        })
        //mod.Thelonious.WaitForShutdown()
    }, 0)
}

