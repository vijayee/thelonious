package ethtest

import (
    "github.com/eris-ltd/deCerver/decerver"
    "github.com/ethereum/eth-go"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/eth-go/ethpipe"
    "github.com/ethereum/eth-go/ethlog"
    "github.com/ethereum/eth-go/ethcrypto"
    "github.com/ethereum/eth-go/ethchain"
    "github.com/ethereum/eth-go/ethreact"
    "github.com/ethereum/go-ethereum/utils"
    "log"
    "bytes"
    "fmt"
    "os"
    "os/exec"
    "os/user"
    "strconv"
    "path"
    "errors"
    "io/ioutil"
)

// some global config vars
// to be set from above
// move these into default config
var (
    GoPath = os.Getenv("GOPATH")
    KeyFile = "keys.txt"
    EthDataDir = path.Join(homeDir(), ".eris-eth")
    EthPort = "30303"
    PathToLLL = "/Users/BatBuddha/Programming/goApps/src/github.com/project-douglas/cpp-ethereum/build/lllc/lllc"
    usr, _ = user.Current() // error?!
    ContractPath = path.Join(GoPath, "src", "github.com", "eris-ltd", "deCerver", "chain", "contracts")
)

// this is who we are
const (
    ClientIdentifier = "Ethereum(PD)"
    Version = "0.5.17"
    Identifier = ""
    KeyStore = "db"
)

//Logging
var logger *ethlog.Logger = ethlog.NewLogger("EthChain(PD)")

type ChainConfig struct{
    Port int        `json:"port"`
    Mining bool     `json:"mining"`
    MaxPeers int    `json:"max_peers"`
    ConfigFile string `json:"config_file"`
    RootDir string  `json:"root_dir"`
    Name string     `json:"name"`
    LogFile string  `json:"log_file"`
}

// set default config object
var DefaultConfig = &ChainConfig{ Port : 30303,
    Mining : false,
    MaxPeers : 10,
    RootDir : path.Join(usr.HomeDir, ".ethchain"),
    Name : "decerver-ethchain",
    ConfigFile : "config",
}


// implements decerver.Blockchain
// our window into eth-go
type EthChain struct{
    Config *ChainConfig
    Ethereum *eth.Ethereum
    Pipe *ethpipe.Pipe
    keyManager *ethcrypto.KeyManager
    reactor *ethreact.ReactorEngine
    chans map[string]chan ethreact.Event
}


// can these methods be functions in decerver that take the modules as argument?
func (e *EthChain) WriteConfig(config_file string){
}
func (e *EthChain) ReadConfig(config_file string){
}
func (e *EthChain) SetConfig(config interface{}) error{
    if s, ok := config.(string); ok{
        e.ReadConfig(s)
    } else if s, ok := config.(ChainConfig); ok{
        e.Config = &s
    } else {
        return errors.New("could not set config")
    }
    return nil
}


func NewEth() *EthChain{
    e := new(EthChain)
    e.Config = DefaultConfig
    return e
}

// initialize an ethchain
func (e *EthChain) Init() error{
    // if didn't call NewEth
    if e.Config == nil{
        e.Config = DefaultConfig
    }
    e.EthConfig()
    ethereum, pipe, keyManager := NewEthPEth()
    ethereum.Port = strconv.Itoa(e.Config.Port)
    ethereum.MaxPeers = e.Config.MaxPeers
    LoadKeys(KeyFile, keyManager)


    e.Ethereum = ethereum
    e.Pipe = pipe
    e.keyManager = keyManager
    e.reactor = ethereum.Reactor()

    log.Println(e.Ethereum.Port)
    
    return nil
}

// start the ethereum node
func (ethchain *EthChain) Start(){
    ethchain.Ethereum.Start(false) // ?
    if ethchain.Config.Mining{
        utils.StartMining(ethchain.Ethereum)
    }
}

// configure an ethereum node
func (e *EthChain) EthConfig() {
    ethutil.ReadConfig(path.Join(e.Config.RootDir, "config"), e.Config.RootDir, "ethchain")
    utils.InitLogging(e.Config.RootDir, e.Config.LogFile, 5, "")
}

// initialize a new ethereum, pipeereum, and keymanager object
func NewEthPEth() (*eth.Ethereum, *ethpipe.Pipe, *ethcrypto.KeyManager){
    // create a new ethereum node: init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
    db := utils.NewDatabase()

    keyManager := utils.NewKeyManager(KeyStore, EthDataDir, db)   
    keyManager.Init("", 0, true)

    clientIdentity := utils.NewClientIdentity(ClientIdentifier, Version, Identifier) 

    ethereum, err := eth.NewEris(db, clientIdentity, keyManager, eth.CapDefault, false, ethchain.GenesisPointer)

//    data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))   
 //   fmt.Println("last block data", data)
  //  os.Exit(0)

    //ethereum, err := eth.New(eth.CapDefault, false)
    if err != nil {
        log.Fatal("Could not start node: %s\n", err)
    }
    // initialize the public ethereum object. this is the interface QML gets, and it's mostly good enough for us to
    pipe := ethpipe.New(ethereum) 
    return ethereum, pipe, keyManager
}

// get=  contract storage
// everything should be in hex!
func (e EthChain) GetStorageAt(contract_addr string, storage_addr string) string{
    caddr := ethutil.Hex2Bytes(contract_addr)
    //saddr := ethutil.Hex2Bytes(storage_addr)
    w := e.Pipe.World()
    ret := w.SafeGet(caddr).GetStorage(ethutil.Big(storage_addr))
    //ret := e.Pipe.Storage(caddr, saddr) 
    //returns an ethValue
    // TODO: figure it out!
    //val := BigNumStrToHex(ret)
    return ret.String()
}

func (e EthChain) GetStorage(contract_addr string) map[string]*ethutil.Value{
    acct := e.Pipe.World().SafeGet(ethutil.Hex2Bytes(contract_addr)).StateObject
    m := make(map[string]*ethutil.Value)
    acct.EachStorage(func(k string, v *ethutil.Value){
            kk := ethutil.Bytes2Hex([]byte(k))
            fmt.Println("each storage", v)
            fmt.Println("each storage val", v.Val)
            m[kk] = v
        })
   return m 
}

// subscribe to an address (hex)
// returns a chanel that will fire when address is updated
func (e EthChain) Subscribe(addr, event string, ch chan decerver.Update){
    addr = string(ethutil.Hex2Bytes(addr))
    eth_ch := make(chan ethreact.Event, 1)
    e.reactor.Subscribe("object:"+addr, eth_ch)

    // since we cant cast to chan interface{}
    // we fire up a goroutine and broadcast on our main ch
    go func(eth_ch chan ethreact.Event, ch chan decerver.Update){
        for {
            r := <- eth_ch           
            log.Println(r)
            ch <- decerver.Update{Address:addr, Event:event}
        }
    }(eth_ch, ch)
}


func (e *EthChain) Msg(addr string, data []string){
    packed := PackTxDataArgs(data...)
    fmt.Println("packed", packed)
    keys := e.fetchKeyPair()
    byte_addr := ethutil.Hex2Bytes(addr)
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big("350")), ethutil.NewValue(ethutil.Big("20000")), ethutil.NewValue(ethutil.Big("1000000")), packed)
    if err != nil{
        log.Fatal("tx err", err)
    }
}

func (e *EthChain) Tx(addr, amt string){
    keys := e.fetchKeyPair()
    byte_addr := ethutil.Hex2Bytes(addr)
    fmt.Println("the amount:", amt, ethutil.Big(amt), ethutil.NewValue(amt), ethutil.NewValue(ethutil.Big(amt)))
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big(amt)), ethutil.NewValue(ethutil.Big("2000")), ethutil.NewValue(ethutil.Big("100000")), "")
    //_, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(amt), ethutil.NewValue("2000"), ethutil.NewValue("100000"), "")
    if err != nil{
        log.Fatal("tx err", err)
    }
}

func (e *EthChain) FetchAddr() string{
    keypair := e.keyManager.KeyPair()
    pub := ethutil.Bytes2Hex(keypair.Address())
    return pub
}

func (e *EthChain) fetchPriv() string{
    keypair := e.keyManager.KeyPair()
    priv := ethutil.Bytes2Hex(keypair.PrivateKey)
    return priv
}

func (e *EthChain) fetchKeyPair() *ethcrypto.KeyPair{
    return e.keyManager.KeyPair()
}

// this is bad but I need it for testing
func (e *EthChain) FetchPriv() string{
    return e.fetchPriv()
}

func (e *EthChain) SetCursor(n int){
    e.keyManager.SetCursor(n)
}

func (e EthChain) DeployContract(file, lang string) string{
    var script string
    if lang == "lll"{
        script = CompileLLL(file) // if lll, compile and pass along
    } else if lang == "mutan"{
        s, _ := ioutil.ReadFile(file) // if mutan, pass along and pipe will compile
        script = string(s)
    } else if lang == "serpent"{
    
    } else {
        script = file
    }
    log.Println("script:",script )
    // messy key system...
    // ethchain should have an 'active key'
    keys := e.fetchKeyPair()

    // well isn't this pretty! barf
    contract_addr, err := e.Pipe.Transact(keys, nil, ethutil.NewValue(ethutil.Big("271")), ethutil.NewValue(ethutil.Big("20000")), ethutil.NewValue(ethutil.Big("1000000")), script)
    if err != nil{
        log.Fatal("could not deploy contract", err)
    }
    return ethutil.Bytes2Hex(contract_addr)
}

func (e EthChain) Stop(){
}

// compile LLL file into evm bytecode 
func CompileLLL(filename string) string{
    cmd := exec.Command(PathToLLL, filename)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        logger.Infoln("Couldn't compile!!", err)
        return ""
    }
    //outstr := strings.Split(out.String(), "\n")
    outstr := out.String()
    for l:=len(outstr);outstr[l-1] == '\n';l--{
        outstr = outstr[:l-1]
    }
    return "0x"+outstr
    //return ethutil.Hex2Bytes(outstr)
}

// some convenience functions

// get users home directory
func homeDir() string{
    usr, _ := user.Current()
    return usr.HomeDir
}

// convert a big int from string to hex
func BigNumStrToHex(s string) string{
    bignum := ethutil.Big(s)
    bignum_bytes := ethutil.BigToBytes(bignum, 16)
    return ethutil.Bytes2Hex(bignum_bytes)
}

// takes a string, converts to bytes, returns hex
func SHA3(tohash string) string{
    h := ethcrypto.Sha3Bin([]byte(tohash))
    return ethutil.Bytes2Hex(h)
}

// pack data into acceptable format for transaction
// TODO: make sure this is ok ...
func PackTxDataArgs(args ... string) string{
    ret := *new([]byte)
    for _, s := range args{
        if s[:2] == "0x"{
            x := ethutil.Hex2Bytes(s[2:])
            l := len(x)
            ret = append(ret, ethutil.RightPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            ret = append(ret, ethutil.RightPadBytes(x, 32*((l + 31)/32))...)
        }
    }
    return "0x" + ethutil.Bytes2Hex(ret)
   // return ret
}

