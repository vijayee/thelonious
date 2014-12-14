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
	ini      = flag.Bool("init", false, "initialize a monkchain config")
	deploy   = flag.Bool("deploy", false, "deploy a monkchain")
	genesis  = flag.String("g", "genesis.json", "pick a genesis contract")
	name     = flag.String("n", "", "name the chain")
	config   = flag.String("c", "monk-config.json", "pick config file")
	mine     = flag.Bool("mine", false, "mine blocks")
	loglevel = flag.Int("log", 5, "log level")

	tester = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage or all")
	blocks = flag.Int("N", 10, "num blocks to wait before shutdown")
)

func main() {
	flag.Parse()

	// Initialize a chain by ensuring decerver dirs exist
	// and dropping a chain config and genesis.json in the
	// specified directory
	if *ini {
		args := flag.Args()
		var p string
		if len(args) == 0 {
			p = "."
		} else {
			p = args[0]
		}
		err := monk.InitChain(p)
		if err != nil {
			log.Fatal(err)
		}
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

	T := NewTester(*tester, *genesis, *mine, *loglevel, *blocks)
	T.Run()
}
