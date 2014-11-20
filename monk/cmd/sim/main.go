package main

import (
    "os"
    "flag"
    "os/user"
    "os/exec"
    "path"
    "strconv"
    "time"
    "fmt"
    "bytes"
    "io/ioutil"
    "encoding/json"
    "github.com/eris-ltd/thelonious/monk"
    "github.com/eris-ltd/thelonious/monkcrypto"
    "github.com/eris-ltd/thelonious/monkdoug"
    "github.com/eris-ltd/thelonious/monkutil"
)

var (
    N = 10
    usr, _ = user.Current()
    rootDir = path.Join(usr.HomeDir, "monksim")

    index = flag.Int("index", -1, "port")
    client = flag.Bool("client", false, "client")
    env = flag.Bool("env", false, "env..")
)


func runMaster(){
    mm := monk.NewMonk(nil)
    mm.Config.RootDir = path.Join(rootDir, strconv.Itoa(*index))
    mm.Config.Port = 30304 + *index
    mm.Config.GenesisConfig = path.Join(rootDir, "genesis.json")
    mm.Config.Mining = true
    mm.Config.LogLevel = 3
    mm.Config.KeySession = "keys"
    //mm.Config.Adversary = 1

    if *index < 0{
        mm.Config.RootDir = path.Join(rootDir, "seed")
        mm.Config.LogLevel = 0
        mm.Config.Mining = true
        mm.Config.Adversary = 0

        go func(){
            for i:=0; i<1; i++{
                cmd := exec.Command("sim", "-client", "-index", strconv.Itoa(i))
                cmd.Stdout = os.Stdout
                go cmd.Run()
            }
        }()
    }

    mm.Init()
    mm.Start()

    time.Sleep(100*time.Second)
}

func runClient(){
    mm := monk.NewMonk(nil)
    mm.Config.RootDir = path.Join(rootDir, strconv.Itoa(*index))
    mm.Config.Port = 30303 + *index
    mm.Config.GenesisConfig = path.Join(rootDir, "genesis.json")
    mm.Config.Mining = true
    if *index >= 0{
        mm.Config.Adversary = 1
    }
    mm.Config.LogLevel = 3
    mm.Config.KeySession = "keys"
    //mm.Config.Adversary = 1

    mm.Init()
    mm.Start()

    time.Sleep(100*time.Second)

    // tally chain
    latest := mm.LatestBlock()
    for {
        b := mm.Block(latest)
        if b == nil{
            break
        }
        fmt.Println(b.Coinbase, b.Hash)
        latest = b.PrevHash
    }

}

func main(){
    flag.Parse()

    if *env{
        setupSimEnv(N)
        return
    }

    if !*client{
        runMaster()
    } else{
        runClient()
    }

}

func setupSimEnv(n int){
    _, err := os.Stat(rootDir)
    if err != nil{
        os.Mkdir(rootDir, 0777)
    }

    keyPairs := []*monkcrypto.KeyPair{}

    // load genesis json, append new keys. allocations.
    g := monkdoug.LoadGenesis(path.Join(monk.ErisLtd, "thelonious", "monk", "genesis-std.json"))

    g.Difficulty = 12

    // generate N keys and N dirs,
    // put a key in each dir 
    // create account object and append to genesis config
    for i:=0;i<n;i++{
        kP := monkcrypto.GenerateNewKeyPair()
        keyPairs = append(keyPairs, kP)

        d := path.Join(rootDir, strconv.Itoa(i))
        os.Mkdir(d, 0777)
        ioutil.WriteFile(path.Join(d, "keys.prv"), []byte(monkutil.Bytes2Hex(kP.PrivateKey)), 0600)

        acc := monkdoug.Account{
            Address: monkutil.Bytes2Hex(kP.Address()),
            Name: strconv.Itoa(i),
            Balance: "12345678900000",
            Permissions: map[string]int{"mine":0, "transact":1, "create":0},
        }
        g.Accounts = append(g.Accounts, &acc)
    }

    b, err := json.Marshal(g)
    if err != nil{
        fmt.Println("err on marshal!", err)
    }
    var out bytes.Buffer
    json.Indent(&out, b, "", "\t")
    ioutil.WriteFile(path.Join(rootDir, "genesis.json"), out.Bytes(), 0600)
}

