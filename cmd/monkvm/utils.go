package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkvm"
)

func resolveCode(code *string) {
	// compile lll
	var err error
	if strings.HasSuffix(*code, ".lll") {
		*code, err = monk.CompileLLL(*code, false)
		if err != nil {
			exit(err)
		}
	}

	// strip hex
	if len(*code) > 2 && (*code)[:2] == "0x" {
		*code = (*code)[2:]
	}
	fmt.Println("code:", *code)
}

func exec(env *VmEnv, code, data []byte) []byte {
	vm := monkvm.New(env)
	vm.Verbose = true
	//vm.Dump = true
	stateObject := env.state.NewStateObject([]byte("evmuser"))

	// the vm calls functions on this without checking nil
	// like in SSTORE
	msg := &monkstate.Message{}

	closure := monkvm.NewClosure(msg, stateObject, stateObject, code, monkutil.Big(*gas), monkutil.Big(*price))
	ret, _, e := closure.Call(vm, data)

	if e != nil {
		fmt.Println(e)
	}

	env.state.UpdateStateObject(stateObject)
	env.state.Update()
	env.state.Sync()

	return ret
}

func printMemStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	//fmt.Printf("vm took %v\n", time.Since(tstart))
	fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)
}

func dumpState(state *monkstate.State) {
	fmt.Println("State dump!")
	it := state.Trie.NewIterator()
	it.Each(func(addr string, acct *monkutil.Value) {
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
		fmt.Println(hexAddr)

		obj := state.GetOrNewStateObject([]byte(addr))
		obj.EachStorage(func(k string, v *monkutil.Value) {
			kk := monkutil.Bytes2Hex([]byte(k))
			v.Decode()
			vv := monkutil.Bytes2Hex(v.Bytes())
			fmt.Printf("\t%s : %s\n", kk, vv)
		})
	})

}

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
