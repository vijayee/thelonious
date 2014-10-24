
package ethtest

import (
    "github.com/eris-ltd/thelonious/ethutil"
    "github.com/eris-ltd/thelonious/ethchain"
    "os"
    "path"
    "fmt"
    "time"
    "strconv"
)

// start the node, start mining, quit
func (t *Test) TestBasic(){
    t.tester("basic", func(eth *EthChain){
        // eth.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        fmt.Println("the node should be running and mining. if not, there are problems. it will stop in 10 seconds ...")
    }, 10)
}

// run a node
func (t* Test) TestRun(){
    t.tester("basic", func(eth *EthChain){
        // eth.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        eth.Ethereum.WaitForShutdown()
    }, 0)
}

func (t *Test) TestState(){
    t.tester("state", func(eth *EthChain){
        state := eth.GetState()
        fmt.Println(state)
    }, 0)
}

// mine, stop mining, start mining
func (t *Test) TestStopMining(){
    t.tester("mining", func(eth *EthChain){
        fmt.Println("mining addresS", eth.FetchAddr())
        eth.Start()
        time.Sleep(time.Second*10)
        fmt.Println("stopping mining")
        eth.StopMining()
        time.Sleep(time.Second*10)
        fmt.Println("starting mining again")
        eth.StartMining()        
    }, 5)
}

// mine, stop mining, start mining
func (t *Test) TestStopListening(){
    t.tester("mining", func(eth *EthChain){
        eth.Config.Mining = false
        eth.Start()
        time.Sleep(time.Second*1)
        fmt.Println("stopping listening")
        eth.StopListening()
        time.Sleep(time.Second*1)
        fmt.Println("starting listening again")
        eth.StartListening()
    }, 3)
}

func (t *Test) TestRestart(){
    eth := NewEth(nil)
    eth.Config.Mining = true
    eth.Init()
    eth.Start()
    time.Sleep(time.Second*5)
    eth.Stop()
    time.Sleep(time.Second*5)
    eth = NewEth(nil)
    eth.Config.Mining = true
    eth.Init()
    eth.Start()
    time.Sleep(time.Second*5)
}

// note about big nums and values...
func (t *Test) TestBig(){
    a := ethutil.NewValue("100000000000")
    fmt.Println("a, bigint", a, a.BigInt())
    // doesnt work! must do: 
    a = ethutil.NewValue(ethutil.Big("100000000000"))
    fmt.Println("a, bigint", a, a.BigInt())
}

// doesn't start up a node, just loads from db and traverses to genesis
func (t *Test) TestMaxGas(){
    t.tester("max gas", func(eth *EthChain){
        //eth.Start()
        v := ethchain.DougValue("maxgas", "values", eth.Ethereum.BlockChain().CurrentBlock.State())
        fmt.Println(v)
        os.Exit(0)
    }, 0)
}

// this one will be in flux for a bit
// test the validation..
func (t *Test) TestValidate(){
    t.tester("validate", func(eth *EthChain){
        PrettyPrintChainAccounts(eth)
        gen := eth.Ethereum.BlockChain().Genesis()
        a1 := ethutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001")
        a2 := ethutil.Hex2Bytes("b9398794cafb108622b07d9a01ecbed3857592d5")
        a3 := ethutil.Hex2Bytes("cced0756041f7aed3ce278c56ee638bade96d001")
        fmt.Println(ethchain.DougValidate(a1, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a2, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a3, gen.State(), "tx"))
        fmt.Println(ethchain.DougValidate(a1, gen.State(), "miner"))
        fmt.Println(ethchain.DougValidate(a2, gen.State(), "miner"))
        fmt.Println(ethchain.DougValidate(a3, gen.State(), "miner"))
    }, 0)
}

// print the genesis state
func (t *Test) TestGenesisAccounts(){
    t.tester("genesis contract", func(eth *EthChain){
        curchain := eth.Ethereum.BlockChain()
        block := curchain.CurrentBlock
        PrettyPrintBlockAccounts(block)
        os.Exit(0)
    }, 0)
}

// print the genesis state
func (t *Test) TestBlockNum(){

    t.tester("block num", func(eth *EthChain){
        curchain := eth.Ethereum.BlockChain()
        block := curchain.CurrentBlock
        fmt.Println(curchain.LastBlockNumber)
        fmt.Println(block.Number)
        fmt.Println(curchain.Genesis().Number)
        os.Exit(0)
    }, 0)
}

func (t *Test) TestCallStack(){
    t.tester("callstack", func(eth *EthChain){
        eth.Start()
        eth.DeployContract(path.Join(ethchain.ContractPath, "lll/callstack.lll"), "lll")
        t.callback("op: callstack", eth, func(){
            PrettyPrintChainAccounts(eth)
        })

    }, 0)
}

func (t *Test) TestCompression(){
    m := map[bool]string{false:"compression-without", true:"compression-with"}
    root := ""
    db := ""
    results_size := make(map[string]int64)
    results_time := make(map[string]time.Duration)
    for compress, name := range m{
        ethutil.COMPRESS = compress
        fmt.Println("compress:", ethutil.COMPRESS)
        t.tester(name, func(eth *EthChain){
            contract_addr := eth.DeployContract(path.Join(ethchain.ContractPath, "tests/lots-of-stuff.lll"), "lll")
            // send many msgs
            start := time.Now()
            for i := 0; i < 10000; i++{
                key := ethutil.Bytes2Hex([]byte(strconv.Itoa(i)))
                value := "x0001200003400021000500555000000008"
                eth.Msg(contract_addr, []string{key, value})
                fmt.Println(i)
            }
            results_time[name] = time.Since(start)
            root = eth.Config.RootDir
            db = eth.Config.DbName
            f := path.Join(root, db)
            fi, err := os.Stat(f)
            if err != nil{
                fmt.Println("cma!", err)
                os.Exit(0)
            }
            results_size[name] = fi.Size()
            
        }, 0)
   }
   for i, v := range results_size{
        fmt.Println(i, v, results_time[i])
   }
}


