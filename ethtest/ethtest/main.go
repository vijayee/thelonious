package main

import (
    "path"
    "log"
    "strconv"
    "flag"
    "os"
    "github.com/ethereum/eth-go"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/go-ethereum/utils"
    "github.com/eris-ltd/eth-go-mods/ethtest"
)   

var (
    tester = flag.String("t", "", "pick a test: basic, tx, traverse, genesis, genesis-msg, get-storage, msg-storage or all")
    genesis = flag.String("g", "", "pick a genesis functin:")
    blocks = flag.Int("n", 10, "num blocks to wait before shutdown")
    pure = flag.Bool("pure", false, "run a pure eth-go node")
)

// this is for running a pure eth-go node
func NewEthereum() *eth.Ethereum{
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
    ethutil.ReadConfig(path.Join("./datadir", "config"), "./datadir", "ethchain")
    // data dir, logfile, log level, debug file
    utils.InitLogging("./datadir", "", 5, "")
    e := NewEthereum()
    e.Start(false)
    utils.StartMining(e)
    e.WaitForShutdown()
}



func main(){
    flag.Parse()

    if *pure{
        Run() //blocks until shutdown
    }

    if *tester == ""{
        flag.Usage()
        os.Exit(0)
    }

    T := ethtest.NewTester(*tester, *genesis, *blocks)
    T.Run()
}

