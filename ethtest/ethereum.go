package ethtest

import (
    "github.com/eris-ltd/deCerver/decerver"
    "github.com/eris-ltd/eth-go-mods"
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethpipe"
    "github.com/eris-ltd/eth-go-mods/ethlog"
    "github.com/eris-ltd/eth-go-mods/ethcrypto"
    "github.com/eris-ltd/eth-go-mods/ethreact"
    "github.com/eris-ltd/eth-go-mods/ethstate"
    "log"
    "fmt"
    "os"
    "os/user"
    "strconv"
    "io/ioutil"
    "math/big"
)

var (
    GoPath = os.Getenv("GOPATH")
    usr, _ = user.Current() // error?!
)

//Logging
var logger *ethlog.Logger = ethlog.NewLogger("EthChain(deCerver)")


// implements decerver.Blockchain
// our window into eth-go
type EthChain struct{
    Config *ChainConfig
    Ethereum *eth.Ethereum
    Pipe *ethpipe.Pipe
    keyManager *ethcrypto.KeyManager
    reactor *ethreact.ReactorEngine
    started bool
    //chans map[string]chan ethreact.Event
}

// new ethchain with default config
// it allows you to pass in an etheruem instance
// btu it will not start a new one otherwise
// this gives you a chance to set config options after
//      creating the EthChain
func NewEth(ethereum *eth.Ethereum) *EthChain{
    e := new(EthChain)
    e.Config = DefaultConfig
    if ethereum != nil{
        e.Ethereum = ethereum
    }
    e.started = false
    return e
}

// initialize an ethchain
// it may or may not already have an ethereum instance
// basically gives you a pipe, local keyMang, and reactor
func (e *EthChain) Init() error{
    // if didn't call NewEth
    if e.Config == nil{
        e.Config = DefaultConfig
    }
    // if no ethereum instance
    if e.Ethereum == nil{
        e.EthConfig()
        e.NewEthereum()
    }

    // public interface
    pipe := ethpipe.New(e.Ethereum) 
    // load keys from file. genesis block keys. convenient for testing
    LoadKeys(e.Config.KeyFile, e.Ethereum.KeyManager())

    e.Pipe = pipe
    e.keyManager = e.Ethereum.KeyManager()
    e.reactor = e.Ethereum.Reactor()

    log.Println(e.Ethereum.Port)
    
    return nil
}

// start the ethereum node
func (ethchain *EthChain) Start(){
    ethchain.Ethereum.Start(true) // peer seed
    ethchain.started = true

    if ethchain.Config.Mining{
        StartMining(ethchain.Ethereum)
    }
}

// create a new ethereum instance
// expects EthConfig to already have been called!
// init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
func (e *EthChain) NewEthereum(){
    db := NewDatabase()

    keyManager := NewKeyManager(e.Config.KeyStore, e.Config.DataDir, db)
    keyManager.Init("", 0, true)
    e.keyManager = keyManager

    clientIdentity := NewClientIdentity(e.Config.ClientIdentifier, e.Config.Version, e.Config.Identifier) 

    // create the ethereum obj
    ethereum, err := eth.New(db, clientIdentity, e.keyManager, eth.CapDefault, false)

    if err != nil {
        log.Fatal("Could not start node: %s\n", err)
    }

    ethereum.Port = strconv.Itoa(e.Config.Port)
    ethereum.MaxPeers = e.Config.MaxPeers

    e.Ethereum = ethereum
}

/*
    Request Functions
*/


// get=  contract storage
// everything should be in hex!
func (e EthChain) GetStorageAt(contract_addr string, storage_addr string) string{
    var saddr *big.Int
    if ethutil.IsHex(storage_addr){
        saddr = ethutil.BigD(ethutil.Hex2Bytes(ethutil.StripHex(storage_addr)))
    } else {
        saddr = ethutil.Big(storage_addr)
    }

    contract_addr = ethutil.StripHex(contract_addr)
    caddr := ethutil.Hex2Bytes(contract_addr)
    //saddr := ethutil.Hex2Bytes(storage_addr)
    w := e.Pipe.World()
    ret := w.SafeGet(caddr).GetStorage(saddr)
    //ret := e.Pipe.Storage(caddr, saddr) 
    //returns an ethValue
    // TODO: figure it out!
    //val := BigNumStrToHex(ret)
    if ret.IsNil(){
        return "0x"
    }
    return ethutil.Bytes2Hex(ret.Bytes())
}

// TODO: return hex string
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

func (e EthChain) GetState() map[string]map[string]string{
    state := e.Pipe.World().State()
    stateMap := make(map[string]map[string]string)
    trieIterator := state.Trie.NewIterator()
    trieIterator.Each(func (addr string, acct *ethutil.Value){
        stateMap[ethutil.Bytes2Hex([]byte(addr))] = make(map[string]string)
        acctObj := ethstate.NewStateObjectFromBytes([]byte(addr), acct.Bytes())
        acctObj.EachStorage(func (storage string, value *ethutil.Value){
            stateMap[ethutil.Bytes2Hex([]byte(addr))][ethutil.Bytes2Hex([]byte(storage))] = ethutil.Bytes2Hex(value.Bytes())
        })
    })
    return stateMap
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

// send a message to a contract
func (e *EthChain) Msg(addr string, data []string){
    packed := PackTxDataArgs(data...)
    keys := e.fetchKeyPair()
    addr = ethutil.StripHex(addr)
    byte_addr := ethutil.Hex2Bytes(addr)
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big("350")), ethutil.NewValue(ethutil.Big("20000")), ethutil.NewValue(ethutil.Big("1000000")), packed)
    if err != nil{
        //TODO: don't be so mean
        log.Fatal("tx err", err)
    }
}

// send a tx
func (e *EthChain) Tx(addr, amt string){
    keys := e.fetchKeyPair()
    addr = ethutil.StripHex(addr)
    if addr[:2] == "0x"{
        addr = addr[2:]
    }
    byte_addr := ethutil.Hex2Bytes(addr)
    fmt.Println("the amount:", amt, ethutil.Big(amt), ethutil.NewValue(amt), ethutil.NewValue(ethutil.Big(amt)))
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big(amt)), ethutil.NewValue(ethutil.Big("2000")), ethutil.NewValue(ethutil.Big("100000")), "")
    //_, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(amt), ethutil.NewValue("2000"), ethutil.NewValue("100000"), "")
    if err != nil{
        log.Fatal("tx err", err)
    }
}

/*
    daemon stuff
*/

func (e *EthChain) StartMining() bool{
    return StartMining(e.Ethereum)
}

func (e *EthChain) StopMining() bool{
    return StopMining(e.Ethereum)
}

func (e *EthChain) StartListening(){
    e.Ethereum.StartListening()
}

func (e *EthChain) StopListening() {
    e.Ethereum.StopListening()
}

/*
    some key management stuff
*/

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

// switch current key
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

func (e *EthChain) Stop(){
    if !e.started{
        fmt.Println("can't stop: haven't even started...")
        return
    }
    e.StopMining()
    fmt.Println("stopped mining")
    e.Ethereum.Stop()
    fmt.Println("stopped ethereum")
    e = &EthChain{Config: e.Config}
    ethlog.Reset()
}

// compile LLL file into evm bytecode 
// returns hex
func CompileLLL(filename string) string{
    code, err := ethutil.CompileLLL(filename)
    if err != nil{
        fmt.Println("error compiling lll!", err)
        return ""
    }
    return "0x"+ethutil.Bytes2Hex(code)
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
    fmt.Println("pack data:", args)
    ret := *new([]byte)
    for _, s := range args{
        if s[:2] == "0x"{
            t := s[2:]
            if len(t) % 2 == 1{
                t = "0"+t
            }
            x := ethutil.Hex2Bytes(t)
            fmt.Println(x)
            l := len(x)
            ret = append(ret, ethutil.LeftPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            ret = append(ret, ethutil.RightPadBytes(x, 32*((l + 31)/32))...)
        }
    }
    return "0x" + ethutil.Bytes2Hex(ret)
   // return ret
}

