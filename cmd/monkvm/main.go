package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"time"
    "strings"

    "github.com/eris-ltd/thelonious/monk"
    "github.com/eris-ltd/thelonious/monkvm"
    "github.com/eris-ltd/thelonious/monkdb"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monklog"
    "github.com/eris-ltd/thelonious/monkstate"
    "github.com/eris-ltd/thelonious/monktrie"
)

var (
	code     = flag.String("code", "", "evm code")
	loglevel = flag.Int("log", 5, "log level")
	gas      = flag.String("gas", "1000000", "gas amount")
	price    = flag.String("price", "0", "gas price")
	dump     = flag.Bool("dump", false, "dump state after run")
	data     = flag.String("data", "", "data")
)

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		logger.Errorln("Fatal: ", err)
		status = 1
	}
	monklog.Flush()
	os.Exit(status)
}

var logger = monklog.NewLogger("VM")


func main(){
	flag.Parse()

	monklog.AddLogSystem(monklog.NewStdLogSystem(os.Stdout, log.LstdFlags, monklog.LogLevel(*loglevel)))
	monkutil.ReadConfig("/tmp/evmtest", "/tmp/evm", "")

    // compile lll
    var err error
    if strings.HasSuffix(*code, ".lll"){
       *code, err = monk.CompileLLL(*code, false)
       if err != nil{
            exit(err)
       }
    }

    *code = (*code)[2:]

    msg := &monkstate.Message{                         
    }

	tstart := time.Now()

	env := NewVmEnv()
    vm := monkvm.New(env)
    vm.Verbose = true

	stateObject := env.state.NewStateObject([]byte("evmuser"))

	closure := monkvm.NewClosure(msg, stateObject, stateObject, monkutil.Hex2Bytes(*code), monkutil.Big(*gas), monkutil.Big(*price))

	ret, _, e := closure.Call(vm, monkutil.Hex2Bytes(*data))

	monklog.Flush()
	if e != nil {
        fmt.Println(e)
	}

    env.state.UpdateStateObject(stateObject)
    env.state.Update()
    env.state.Sync()

	if *dump {
		//fmt.Println(string(env.state.Dump()))
        dumpState(env.state)
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("vm took %v\n", time.Since(tstart))
	fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)

	fmt.Printf("%x\n", ret)
}

type VmEnv struct {
	state *monkstate.State
}

func NewVmEnv() *VmEnv {
	db, _ := monkdb.NewMemDatabase()
    monkutil.Config.Db = db
	return &VmEnv{monkstate.New(monktrie.New(db, ""))}
}

func (VmEnv) Origin() []byte            { return nil }
func (VmEnv) BlockNumber() *big.Int     { return nil }
func (VmEnv) BlockHash() []byte         { return nil }
func (VmEnv) PrevHash() []byte          { return nil }
func (VmEnv) Coinbase() []byte          { return nil }
func (VmEnv) Time() int64               { return 0 }
func (VmEnv) GasLimit() *big.Int        { return nil }
func (VmEnv) Difficulty() *big.Int      { return nil }
func (VmEnv) Value() *big.Int           { return nil }
func (self *VmEnv) State() *monkstate.State { return self.state }
func (self *VmEnv) Doug() []byte            { return nil}
func (self *VmEnv) DougValidate(addr []byte, role string, state *monkstate.State) error {
    return nil
}

func dumpState(state *monkstate.State){
    fmt.Println("State dump!")
    it := state.Trie.NewIterator()
    it.Each(func(addr string, acct *monkutil.Value){
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
        fmt.Println(hexAddr)
    
        obj := state.GetOrNewStateObject([]byte(addr))
        obj.EachStorage(func(k string, v *monkutil.Value){
            kk := monkutil.Bytes2Hex([]byte(k))
            v.Decode()
            vv := monkutil.Bytes2Hex(v.Bytes())
            fmt.Printf("\t%s : %s\n", kk, vv)
        })
    })

}
