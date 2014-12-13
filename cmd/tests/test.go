package main

import (
	"fmt"
	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/thelonious/monkutil"
	"time"
)

// environment object for running custom tests (ie. not used in `go test`)
// one tester obj, will run many tests (sequentially)
type Test struct {
	genesis string
	blocks  int
	mine    bool
	log     int
	mod     *monk.MonkModule

	// test specific
	testerFunc string
	success    bool
	err        error

	gendougaddr string //hex address

	reactor *monkreact.ReactorEngine

	failed []string // failed tests
}

func NewTester(tester, genesis string, mine bool, log int, blocks int) *Test {
	return &Test{testerFunc: tester, genesis: genesis, mine: mine, log: log, blocks: blocks, failed: []string{}}
}

// for functions we cant use `go test` on
func (t *Test) Run() {
	switch t.testerFunc {
	case "basic":
		t.TestBasic()
	case "run":
		t.TestRun()
	case "load":
		t.TestRunLoad()
	case "event":
		t.TestRunEvent()
	//case "genesis":
	//t.TestGenesisAccounts()
	case "mining":
		t.TestStopMining()
	case "listening":
		t.TestStopListening()
	case "restart":
		t.TestRestart()
	case "callstack":
		t.TestCallStack()
	//case "maxgas":
	//t.TestMaxGas()
	case "state":
		t.TestState()
	case "compress":
		t.TestCompression()
	}
	fmt.Println(t.success)
}

// general tester function on a thelonious node
// note, you ought to call th.Start() somewhere in testing()!
func (t *Test) tester(name string, testing func(mod *monk.MonkModule), end int) {
	mod := t.mod
	if mod == nil {
		mod = monk.NewMonk(nil)
		t.mod = mod
	}
	mod.Config.Mining = mod.Config.Mining
	if t.mine {
		mod.Config.Mining = true
	}
	mod.Config.DbMem = true

	if mod.Config.LogLevel != t.log {
		mod.Config.LogLevel = t.log
	}

	mod.Init()

	testing(mod)

	if end > 0 {
		time.Sleep(time.Second * time.Duration(end))
	}
	mod.Shutdown()
	t.mod = nil
	time.Sleep(time.Second * 3)
}

// compare expected and recovered vals
func check_recovered(expected, recovered string) bool {
	if monkutil.Coerce2Hex(recovered) == monkutil.Coerce2Hex(expected) {
		fmt.Println("Test passed")
		return true
	} else {
		fmt.Println("Test failed. Expected", expected, "Recovered", recovered)
		return false
	}
}
