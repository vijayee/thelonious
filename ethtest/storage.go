package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "fmt"
    "math/big"
    "path"
)


func TestMsgStorage(){
    tester("msg storage", func(eth *EthChain){
        eth.Start()
        contract_addr := eth.DeployContract(path.Join(ethchain.ContractPath, "genesis.mu"), "mutan")
        fmt.Println("contract addr", contract_addr)
        callback("get storage", eth, func(){
            fmt.Println("####RESPONSE####")
            fmt.Println(eth.GetStorageAt(contract_addr, "5"))
            storage := eth.GetStorage(contract_addr)
            fmt.Println(storage)
            eth.Msg(contract_addr, []string{"21"})
            callback("get storage", eth, func(){
                fmt.Println("####RESPONSE####")
                fmt.Println(eth.GetStorageAt(contract_addr, "5"))
                storage := eth.GetStorage(contract_addr)
                fmt.Println(storage)
                pretty_print_accounts_chain(eth)
            })
        })

    })
}

// add a contract account to the genesis block
func TestGetStorage(){
    tester("get storage", func(eth *EthChain){
        eth.Start()
        fmt.Println("addR", eth.FetchAddr())    
        //c := "./test.lll"
        contract_addr := eth.DeployContract("contract.storage[5]=21", "")
        //contract_addr := eth.DeployContract(c, "lll")
        fmt.Println("contract addr", contract_addr)
        callback("get storage", eth, func(){
            fmt.Println("####RESPONSE####")
            fmt.Println(eth.GetStorageAt(contract_addr, "5"))
            storage := eth.GetStorage(contract_addr)
            if len(storage) == 0{
                fmt.Println("Failed to store a value!")
            } else{
                fmt.Println("Value stored successfuly")
                fmt.Println(storage)
            }
            pretty_print_accounts_chain(eth)
        })
        //eth.Ethereum.WaitForShutdown()
    })
}


func TestTx(){
    tester("basic tx", func(eth *EthChain){
        eth.Start()
        addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
        addr_bytes := ethutil.Hex2Bytes(addr)
        amount := "567890"
        old_balance := eth.Pipe.Balance(addr_bytes)
        eth.SetCursor(0)
        eth.Tx(addr, amount)
        callback("get balance", eth, func(){
            fmt.Println("####RESPONSE####")
            new_balance := eth.Pipe.Balance(addr_bytes)
            fmt.Println("new balance", new_balance.BigInt())

            old := old_balance.BigInt()
            am := ethutil.Big(amount)
            fmt.Println("amount", am)
            fmt.Println("old", old)
            n := new(big.Int)
            n = n.Add(old, am) // TODO!!!!
            newb := ethutil.BigD(new_balance.Bytes())
            if n.Cmp(newb) != 0{ 
                fmt.Println("SIMPLE TX FAILED!")
                fmt.Println("expected", n, "got", newb)
            } else {
                fmt.Println("Simple tx passed")
            }
        })
        //eth.Ethereum.WaitForShutdown()
    })
}



func TestBasic(){
    tester("basic", func(eth *EthChain){
        eth.SetCursor(0)
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        fmt.Println("the node should be running and mining. if not, there are problems. it will stop in 10 seconds ...")
    })
}

func TestBig(){
    a := ethutil.NewValue("100000000000")
    fmt.Println("a, bigint", a, a.BigInt())
    // doesnt work! must do: 
    a = ethutil.NewValue(ethutil.Big("100000000000"))
    fmt.Println("a, bigint", a, a.BigInt())
}

