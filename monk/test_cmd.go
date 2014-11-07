package monk

import (
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkchain"
    "os"
    "path"
    "fmt"
    "log"
    "time"
    "strconv"
)

// start the node, start mining, quit
func (t *Test) TestBasic(){
    t.tester("basic", func(mod *MonkModule){
        // mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", mod.monk.FetchAddr())
        mod.Start()
        fmt.Println("the node should be running and mining. if not, there are problems. it will stop in 10 seconds ...")
    }, 10)
}

// run a node
func (t* Test) TestRun(){
    t.tester("basic", func(mod *MonkModule){
        // mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
        fmt.Println("mining addresS", mod.monk.FetchAddr())
        mod.Start()
        mod.monk.ethereum.WaitForShutdown()
    }, 0)
}

func (t *Test) TestState(){
    t.tester("state", func(mod *MonkModule){
        state := mod.GetState()
        fmt.Println(state)
    }, 0)
}

// mine, stop mining, start mining
func (t *Test) TestStopMining(){
    t.tester("mining", func(mod *MonkModule){
        fmt.Println("mining addresS", mod.monk.FetchAddr())
        mod.Start()
        time.Sleep(time.Second*10)
        fmt.Println("stopping mining")
        mod.monk.StopMining()
        time.Sleep(time.Second*10)
        fmt.Println("starting mining again")
        mod.monk.StartMining()        
    }, 5)
}

// mine, stop mining, start mining
func (t *Test) TestStopListening(){
    t.tester("mining", func(mod *MonkModule){
        mod.monk.config.Mining = false
        mod.Start()
        time.Sleep(time.Second*1)
        fmt.Println("stopping listening")
        mod.monk.StopListening()
        time.Sleep(time.Second*1)
        fmt.Println("starting listening again")
        mod.monk.StartListening()
    }, 3)
}

func (t *Test) TestRestart(){
    mod := NewMonk(nil)
    mod.monk.config.Mining = true
    mod.Init()
    mod.Start()
    time.Sleep(time.Second*5)
    mod.Shutdown()
    time.Sleep(time.Second*5)
    mod = NewMonk(nil)
    mod.monk.config.Mining = true
    mod.Init()
    mod.Start()
    time.Sleep(time.Second*5)
}

// note about big nums and values...
func (t *Test) TestBig(){
    a := monkutil.NewValue("100000000000")
    fmt.Println("a, bigint", a, a.BigInt())
    // doesnt work! must do: 
    a = monkutil.NewValue(monkutil.Big("100000000000"))
    fmt.Println("a, bigint", a, a.BigInt())
}

// doesn't start up a node, just loads from db and traverses to genesis
func (t *Test) TestMaxGas(){
    t.tester("max gas", func(mod *MonkModule){
        //mod.Start()
        v := monkchain.DougValue("maxgas", "values", mod.monk.ethereum.BlockChain().CurrentBlock.State())
        fmt.Println(v)
        os.Exit(0)
    }, 0)
}

// this one will be in flux for a bit
// test the validation..
func (t *Test) TestValidate(){
    t.tester("validate", func(mod *MonkModule){
        PrettyPrintChainAccounts(mod)
        gen := mod.monk.ethereum.BlockChain().Genesis()
        a1 := monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001")
        a2 := monkutil.Hex2Bytes("b9398794cafb108622b07d9a01ecbed3857592d5")
        a3 := monkutil.Hex2Bytes("cced0756041f7aed3ce278c56ee638bade96d001")
        fmt.Println(monkchain.DougValidate(a1, gen.State(), "tx"))
        fmt.Println(monkchain.DougValidate(a2, gen.State(), "tx"))
        fmt.Println(monkchain.DougValidate(a3, gen.State(), "tx"))
        fmt.Println(monkchain.DougValidate(a1, gen.State(), "miner"))
        fmt.Println(monkchain.DougValidate(a2, gen.State(), "miner"))
        fmt.Println(monkchain.DougValidate(a3, gen.State(), "miner"))
    }, 0)
}

// print the genesis state
func (t *Test) TestGenesisAccounts(){
    t.tester("genesis contract", func(mod *MonkModule){
        curchain := mod.monk.ethereum.BlockChain()
        block := curchain.CurrentBlock
        PrettyPrintBlockAccounts(block)
        os.Exit(0)
    }, 0)
}

// print the genesis state
func (t *Test) TestBlockNum(){

    t.tester("block num", func(mod *MonkModule){
        curchain := mod.monk.ethereum.BlockChain()
        block := curchain.CurrentBlock
        fmt.Println(curchain.LastBlockNumber)
        fmt.Println(block.Number)
        fmt.Println(curchain.Genesis().Number)
        os.Exit(0)
    }, 0)
}

func (t *Test) TestCallStack(){
    t.tester("callstack", func(mod *MonkModule){
        mod.Start()
        mod.Script(path.Join(monkchain.ContractPath, "lll/callstack.lll"), "lll")
        t.callback("op: callstack", mod, func(){
            PrettyPrintChainAccounts(mod)
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
        monkutil.COMPRESS = compress
        fmt.Println("compress:", monkutil.COMPRESS)
        t.tester(name, func(mod *MonkModule){
            contract_addr, err := mod.Script(path.Join(monkchain.ContractPath, "tests/lots-of-stuff.lll"), "lll")
            if err != nil{
                log.Fatal(err)
            }
            // send many msgs
            start := time.Now()
            for i := 0; i < 10000; i++{
                key := monkutil.Bytes2Hex([]byte(strconv.Itoa(i)))
                value := "x0001200003400021000500555000000008"
                mod.Msg(contract_addr, []string{key, value})
                fmt.Println(i)
            }
            results_time[name] = time.Since(start)
            root = mod.monk.config.RootDir
            db = mod.monk.config.DbName
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


