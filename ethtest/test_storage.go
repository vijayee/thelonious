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
)

/*
    TestSimpleStorage
    TestMsgStorage
    TestTx
*/


// contract that stores a single value during init
func (t *Test) TestSimpleStorage(){
    t.tester("simple storage", func(eth *EthChain){
        eth.Start()
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
        c := "lll/simple-storage.lll"
        p := path.Join(eth.Config.ContractPath, c)
        err := ioutil.WriteFile(p, []byte(code), 0644)
        if err != nil{
            fmt.Println("write file failed", err)
            os.Exit(0)
        }
        contract_addr := eth.DeployContract(p, "lll")
        // callback when block is mined
        t.callback("simple storage", eth, func(){
            recovered := "0x" + eth.GetStorageAt(contract_addr, key)
            t.success = check_recovered(value, recovered)
        })
    }, 0)
}

// test a simple key-value store contract
func (t *Test) TestMsgStorage(){
    t.tester("msg storage", func(eth *EthChain){
        eth.Start()
        contract_addr := eth.DeployContract(path.Join(ethchain.ContractPath, "lll/keyval.lll"), "lll")
        t.callback("deploy key-value", eth, func(){
            key := "0x21"
            value := "0x400"
            time.Sleep(time.Nanosecond) // needed or else subscribe channels block and are skipped ... TODO: why?!
            eth.Msg(contract_addr, []string{key, value})
            t.callback("test key-value", eth, func(){
                recovered := "0x"+eth.GetStorageAt(contract_addr, key)
                t.success = check_recovered(value, recovered)
            })
        })

    }, 0)
}

// test simple tx
func (t *Test) TestTx(){
    t.tester("basic tx", func(eth *EthChain){
        eth.Start()
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := ethutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := eth.Pipe.Balance(addr_bytes)
        //eth.SetCursor(0)
        eth.Tx(addr, amount)
        t.callback("get balance", eth, func(){
            new_balance := eth.Pipe.Balance(addr_bytes)
            old := old_balance.BigInt()
            am := ethutil.Big(amount)
            n := new(big.Int)
            n.Add(old, am)
            newb := ethutil.BigD(new_balance.Bytes())
            t.success = check_recovered(n.String(), newb.String())
        })
        //eth.Ethereum.WaitForShutdown()
    }, 0)
}
