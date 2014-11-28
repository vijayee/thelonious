package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	//"github.com/ethereum/eth-go"
	//"github.com/ethereum/go-ethereum/utils"
	//"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monk"
	_ "net/http/pprof"
)

var (
	tester  = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage or all")
	genesis = flag.String("g", "", "pick a genesis contract:")
	blocks  = flag.Int("n", 10, "num blocks to wait before shutdown")
	pure    = flag.Bool("pure", false, "run a pure eth-go node")
)

/*
// this is for running a pure eth-go node
func NewThelonious() *eth.Thelonious{
    db := utils.NewDatabase()

    keyManager := utils.NewKeyManager("db", "./datadir", db)
    keyManager.Init("", 0, true)

    clientIdentity := utils.NewClientIdentity("fucker","0.3", "")

    // create the ethereum obj
    ethereum, err := eth.New(db, clientIdentity, keyManager, eth.CapDefault, false)

    if err != nil {
        log.Fatal("Could not start node: %s\n", err)
    }

    ethereum.Port = strconv.Itoa(30303)
    ethereum.MaxPeers = 10

    return ethereum
}

func Run(){
    monkutil.ReadConfig(path.Join("./datadir", "config"), "./datadir", "monkchain")
    // data dir, logfile, log level, debug file
    utils.InitLogging("./datadir", "", 5, "")
    e := NewThelonious()
    e.Start(false)
    utils.StartMining(e)
    e.WaitForShutdown()
}
*/

func main() {
	flag.Parse()
	/*
	   if *pure{
	       Run() //blocks until shutdown
	   }*/

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
