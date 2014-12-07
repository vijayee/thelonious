package main

import (
	"flag"
	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monklog"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var logger *monklog.Logger = monklog.NewLogger("CLI")

var (
	ini     = flag.String("init", ".", "initialize a monkchain config")
	deploy  = flag.Bool("deploy", false, "deploy a monkchain")
	tester  = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage or all")
	genesis = flag.String("g", "genesis.json", "pick a genesis contract")
	name    = flag.String("n", "", "name the chain")
	config  = flag.String("c", "monk-config.json", "pick config file")
	blocks  = flag.Int("N", 10, "num blocks to wait before shutdown")
	pure    = flag.Bool("pure", false, "run a pure eth-go node")
)

func main() {
	flag.Parse()

	err := monk.InitChain(*ini)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize a chain by ensuring decerver dirs exist
	// and dropping a chain config and genesis.json in the
	// specified directory
	setflags := specifiedFlags()
	if _, ok := setflags["init"]; ok {
		os.Exit(0)
	}

	if *deploy {
		// deploy the genblock, copy into ~/.decerver, exit
		monk.DeploySequence(*name, *genesis, *config)
	}

	if *tester == "" {
		flag.Usage()
		os.Exit(0)
	}

	// run the pprof server for debug
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	T := monk.NewTester(*tester, *genesis, *blocks)
	T.Run()
}

func specifiedFlags() map[string]bool {
	// compute a map of the flags that have been set
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})
	return setFlags
}
