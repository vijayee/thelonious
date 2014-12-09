package main

import (
	"fmt"
	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monkutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"
)

// start the node, start mining, quit
func (t *Test) TestBasic() {
	t.tester("basic", func(mod *monk.MonkModule) {
		// mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
		fmt.Println("mining addresS", mod.ActiveAddress())
		mod.Start()
		fmt.Println("the node should be running and mining. if not, there are problems. it will stop in 10 seconds ...")
	}, 10)
}

// run a node
func (t *Test) TestRun() {
	t.tester("basic", func(mod *monk.MonkModule) {
		// mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
		fmt.Println("mining addresS", mod.ActiveAddress())
		mod.Start()
		mod.WaitForShutdown()
	}, 0)
}

// run a node under load
func (t *Test) TestRunLoad() {
	t.tester("basic", func(mod *monk.MonkModule) {
		// mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
		fmt.Println("mining addresS", mod.ActiveAddress())
		mod.Start()
		go func() {
			tick := time.Tick(1000 * time.Millisecond)
			addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
			amount := "567890"
			for _ = range tick {
				mod.Tx(addr, amount)
			}
		}()
		mod.WaitForShutdown()
	}, 0)
}

// run a node under load
func (t *Test) TestRunEvent() {
	t.tester("basic", func(mod *monk.MonkModule) {
		// mod.SetCursor(0) // setting this will invalidate you since this addr isnt in the genesis
		fmt.Println("mining addresS", mod.ActiveAddress())
		mod.Start()
		ch := mod.Subscribe("testchannel", "newBlock", "")
		ctr := 0
		for evt := range ch {
			if ctr > 50 {
				return
			}
			fmt.Println("Received: " + evt.Event)
			mod.State()
			ctr++
		}
		mod.WaitForShutdown()
	}, 0)
}

func (t *Test) TestState() {
	t.tester("state", func(mod *monk.MonkModule) {
		state := mod.State()
		fmt.Println(state)
	}, 0)
}

// mine, stop mining, start mining
func (t *Test) TestStopMining() {
	t.tester("mining", func(mod *monk.MonkModule) {
		fmt.Println("mining addresS", mod.ActiveAddress())
		mod.Start()
		time.Sleep(time.Second * 10)
		fmt.Println("stopping mining")
		mod.AutoCommit(false)
		time.Sleep(time.Second * 10)
		fmt.Println("starting mining again")
		mod.AutoCommit(true)
	}, 5)
}

// mine, stop mining, start mining
func (t *Test) TestStopListening() {
	t.tester("mining", func(mod *monk.MonkModule) {
		mod.Config.Mining = false
		mod.Start()
		time.Sleep(time.Second * 1)
		fmt.Println("stopping listening")
		mod.Listen(false)
		time.Sleep(time.Second * 1)
		fmt.Println("starting listening again")
		mod.Listen(true)
	}, 3)
}

func (t *Test) TestRestart() {
	mod := monk.NewMonk(nil)
	mod.Config.Mining = true
	mod.Init()
	mod.Start()
	time.Sleep(time.Second * 5)
	mod.Shutdown()
	time.Sleep(time.Second * 5)
	mod = monk.NewMonk(nil)
	mod.Config.Mining = true
	mod.Init()
	mod.Start()
	time.Sleep(time.Second * 5)
}

// note about big nums and values...
func (t *Test) TestBig() {
	a := monkutil.NewValue("100000000000")
	fmt.Println("a, bigint", a, a.BigInt())
	// doesnt work! must do:
	a = monkutil.NewValue(monkutil.Big("100000000000"))
	fmt.Println("a, bigint", a, a.BigInt())
}

// doesn't start up a node, just loads from db and traverses to genesis
/*
func (t *Test) TestMaxGas(){
    t.tester("max gas", func(mod *monk.MonkModule){
        //mod.Start()
        v := monkchain.DougValue("maxgas", "values", mod.monk.thelonious.ChainManager().CurrentBlock.State())
        fmt.Println(v)
        os.Exit(0)
    }, 0)
}*/

// print the genesis state
/*
//TODO: fix this...
func (t *Test) TestGenesisAccounts() {
	t.tester("genesis contract", func(mod *monk.MonkModule) {
		curchain := mod.monk.thelonious.ChainManager()
		block := curchain.CurrentBlock()
		monk.PrettyPrintBlockAccounts(block)
		os.Exit(0)
	}, 0)
}*/

/*
func (t *Test) TestBlockNum() {
	t.tester("block num", func(mod *monk.MonkModule) {
		curchain := mod.monk.thelonious.ChainManager()
		block := curchain.CurrentBlock()
		fmt.Println(curchain.CurrentBlockNumber())
		fmt.Println(block.Number)
		fmt.Println(curchain.Genesis().Number)
		os.Exit(0)
	}, 0)
}*/

func (t *Test) TestCallStack() {
	t.tester("callstack", func(mod *monk.MonkModule) {
		mod.Start()
		mod.Script(path.Join(t.mod.Config.ContractPath, "lll/callstack.lll"), "lll")
		mod.Commit()
		monk.PrettyPrintChainAccounts(mod)
	}, 0)
}

func (t *Test) TestCompression() {
	m := map[bool]string{false: "compression-without", true: "compression-with"}
	root := ""
	db := ""
	results_size := make(map[string]int64)
	results_time := make(map[string]time.Duration)
	for compress, name := range m {
		monkutil.COMPRESS = compress
		fmt.Println("compress:", monkutil.COMPRESS)
		t.tester(name, func(mod *monk.MonkModule) {
			contract_addr, err := mod.Script(path.Join(t.mod.Config.ContractPath, "tests/lots-of-stuff.lll"), "lll")
			if err != nil {
				log.Fatal(err)
			}
			// send many msgs
			start := time.Now()
			for i := 0; i < 10000; i++ {
				key := monkutil.Bytes2Hex([]byte(strconv.Itoa(i)))
				value := "x0001200003400021000500555000000008"
				mod.Msg(contract_addr, []string{key, value})
				fmt.Println(i)
			}
			results_time[name] = time.Since(start)
			root = mod.Config.RootDir
			db = mod.Config.DbName
			f := path.Join(root, db)
			fi, err := os.Stat(f)
			if err != nil {
				fmt.Println("cma!", err)
				os.Exit(0)
			}
			results_size[name] = fi.Size()

		}, 0)
	}
	for i, v := range results_size {
		fmt.Println(i, v, results_time[i])
	}
}
