package main

import (
	"flag"
	"log"
	"math/big"
	"os"

	"github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/thelonious/monkutil"
)

var (
	script   = flag.String("script", "", "evm code or lll file")
	loglevel = flag.Int("log", 5, "log level")
	gas      = flag.String("gas", "1000000", "gas amount")
	price    = flag.String("price", "0", "gas price")
	dump     = flag.Bool("dump", false, "dump state after run")
	data     = flag.String("data", "", "data")

	test = flag.String("t", "", "test to run")
)

var logger = monklog.NewLogger("VM")

func main() {
	flag.Parse()

	monklog.AddLogSystem(monklog.NewStdLogSystem(os.Stdout, log.LstdFlags, monklog.LogLevel(*loglevel)))
	monkutil.ReadConfig("/tmp/evmtest", "/tmp/evm", "")

	if *test != "" {
		// run a test and quit
		runTest(*test)
		exit(nil)
	}

	env := NewVmEnv()

	resolveCode(script)

	ret := exec(env, monkutil.Hex2Bytes(*script), monkutil.Hex2Bytes(*data))
	logger.Infof("return: %x\n", ret)

	monklog.Flush()

	if *dump {
		dumpState(env.state)
	}
}

type VmEnv struct {
	state *monkstate.State
}

func NewVmEnv() *VmEnv {
	db, _ := monkdb.NewMemDatabase()
	monkutil.Config.Db = db
	return &VmEnv{monkstate.New(monktrie.New(db, ""))}
}

func (VmEnv) Origin() []byte                { return nil }
func (VmEnv) BlockNumber() *big.Int         { return nil }
func (VmEnv) BlockHash() []byte             { return nil }
func (VmEnv) PrevHash() []byte              { return nil }
func (VmEnv) Coinbase() []byte              { return nil }
func (VmEnv) Time() int64                   { return 0 }
func (VmEnv) GasLimit() *big.Int            { return nil }
func (VmEnv) Difficulty() *big.Int          { return nil }
func (VmEnv) Value() *big.Int               { return nil }
func (self *VmEnv) State() *monkstate.State { return self.state }
func (self *VmEnv) Doug() []byte            { return nil }
func (self *VmEnv) DougValidate(addr []byte, role string, state *monkstate.State) error {
	return nil
}
