package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "fmt"
    "math/big"
    "path"
    "io/ioutil"
    "os"
    "time"
    "testing"
)

/*
    TestSimpleStorage
    TestMsgStorage
    TestTx
    TestManyTx
*/


// contract that stores a single value during init
func TestSimpleStorage(t *testing.T){
    tester2("simple storage", func(eth *EthChain){
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
        p := path.Join(eth.Config.ContractPath, c)
        err := ioutil.WriteFile(p, []byte(code), 0644)
        if err != nil{
            fmt.Println("write file failed", err)
            os.Exit(0)
        }
        contract_addr := eth.DeployContract(p, "lll")
        eth.Start()
        // callback when block is mined
        callback2("simple storage", eth, func(){
            recovered := "0x" + eth.GetStorageAt(contract_addr, key)
            result := check_recovered(value, recovered)
            if !result{
                t.Error("got:", recovered, "expected:", value)
            }
        })
    }, 0)
}

// test a simple key-value store contract
func TestMsgStorage(t *testing.T){
    tester2("msg storage", func(eth *EthChain){
        contract_addr := eth.DeployContract(path.Join(ethchain.ContractPath, "tests/keyval.lll"), "lll")
        eth.Start()
        callback2("deploy key-value", eth, func(){
            key := "0x21"
            value := "0x400"
            time.Sleep(time.Nanosecond) // needed or else subscribe channels block and are skipped ... TODO: why?!
            eth.Msg(contract_addr, []string{key, value})
            callback2("test key-value", eth, func(){
                start := time.Now()
                recovered := "0x"+eth.GetStorageAt(contract_addr, key)
                dif := time.Since(start)
                fmt.Println("get storage took", dif)
                result := check_recovered(value, recovered)
                if !result{
                    t.Error("got:", value, "expected:", recovered)
                }
            })
        })

    }, 0)
}

// test simple tx
func TestTx(t *testing.T){
    tester2("basic tx", func(eth *EthChain){
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := ethutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := eth.Pipe.Balance(addr_bytes)
        //eth.SetCursor(0)
        start := time.Now()
        eth.Tx(addr, amount)
        dif := time.Since(start)
        fmt.Println("sending one tx took", dif)
        eth.Start()
        callback2("get balance", eth, func(){
            new_balance := eth.Pipe.Balance(addr_bytes)
            old := old_balance.BigInt()
            am := ethutil.Big(amount)
            n := new(big.Int)
            n.Add(old, am)
            newb := ethutil.BigD(new_balance.Bytes())
            //t.success = check_recovered(n.String(), newb.String())
            result := check_recovered(n.String(), newb.String())
            if !result{
                t.Error("got:", newb.String(), "expected:", n.String())
            }
        })
        //eth.Ethereum.WaitForShutdown()
    }, 0)
}

func TestManyTx(t *testing.T){
    tester2("many tx", func(eth *EthChain){
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := ethutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := eth.Pipe.Balance(addr_bytes)
        N := 1000
        //eth.SetCursor(0)
        start := time.Now()
        for i:=0; i<N; i++{
            eth.Tx(addr, amount)
        }
        end := time.Since(start)
        fmt.Printf("sending %d txs took %s\n", N, end)
        eth.Start()
        callback2("get balance", eth, func(){
            new_balance := eth.Pipe.Balance(addr_bytes)
            old := old_balance.BigInt()
            am := ethutil.Big(amount)
            mult := big.NewInt(int64(N))
            n := new(big.Int)
            n.Add(old, n.Mul(mult, am))
            newb := ethutil.BigD(new_balance.Bytes())
            results := check_recovered(n.String(), newb.String())
            if !results{
                t.Error("got:", newb.String(), "expected:", n.String())
            }
        })
        //eth.Ethereum.WaitForShutdown()
    }, 0)
    
}
