package monk

import (
    "github.com/eris-ltd/decerver-interfaces/events"
    "github.com/eris-ltd/decerver-interfaces/core"
    "github.com/eris-ltd/decerver-interfaces/api"
    "github.com/eris-ltd/thelonious"
    "github.com/eris-ltd/thelonious/ethutil"
    "github.com/eris-ltd/thelonious/ethpipe"
    "github.com/eris-ltd/thelonious/ethlog"
    "github.com/eris-ltd/thelonious/ethcrypto"
    "github.com/eris-ltd/thelonious/ethreact"
    "github.com/eris-ltd/thelonious/ethstate"
    "github.com/eris-ltd/thelonious/ethchain"
    "log"
    "fmt"
    "os"
    "os/user"
    "strconv"
    "io/ioutil"
    "math/big"
    "time"
)

var (
    GoPath = os.Getenv("GOPATH")
    usr, _ = user.Current() // error?!
)

//Logging
var logger *ethlog.Logger = ethlog.NewLogger("EthChain(deCerver)")

// implements decerver Module
// our window into eth-go
type EthChain struct{
    Config *ChainConfig
    Ethereum *eth.Ethereum
    Pipe *ethpipe.Pipe
    keyManager *ethcrypto.KeyManager
    reactor *ethreact.ReactorEngine
    started bool
    Chans map[string]chan events.Event
}

// new ethchain with default config
// it allows you to pass in an etheruem instance
// btu it will not start a new one otherwise
// this gives you a chance to set config options after
//      creating the EthChain
func NewEth(ethereum *eth.Ethereum) *EthChain{
    e := new(EthChain)
    // here we load default config and leave it to caller
    // to read a config file to overwrite
    e.Config = DefaultConfig
    if ethereum != nil{
        e.Ethereum = ethereum
    }
    e.started = false
    return e
}

// register the module with the decerver javascript vm
func (e *EthChain) RegisterModule(registry api.ApiRegistry, logger core.LogSystem) error{
    return nil
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

    // subscribe to the new block
    e.Chans = make(map[string]chan events.Event)
    e.Subscribe("newBlock", "newBlock", "")

    log.Println(e.Ethereum.Port)
    
    return nil
}

// start the ethereum node
func (ec *EthChain) Start() error{
    ec.Ethereum.Start(true) // peer seed
    ec.started = true

    if ec.Config.Mining{
        StartMining(ec.Ethereum)
    }
    return nil
}

// create a new ethereum instance
// expects EthConfig to already have been called!
// init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
func (e *EthChain) NewEthereum(){
    db := NewDatabase(e.Config.DbName)

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

// returns hex addr of gendoug
func (e EthChain) GenDoug() string{
    return ethutil.Bytes2Hex(ethchain.GENDOUG)
}


// TODO: return hex string
func (e EthChain) _GetStorage(contract_addr string) map[string]*ethutil.Value{
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

func (e EthChain) State() core.State{
    return e.GetState()
}

func (e EthChain) Storage(addr string) core.Storage{
    return e.GetStorage(addr)
}

func (e EthChain) GetStorage(addr string) core.Storage{
    w := e.Pipe.World()
    obj := w.SafeGet(ethutil.UserHex2Bytes(addr)).StateObject
    ret := core.Storage{make(map[string]interface{}), []string{}}
    obj.EachStorage(func(k string, v *ethutil.Value){
        kk := ethutil.Bytes2Hex([]byte(k))
        vv := ethutil.Bytes2Hex(v.Bytes())
        ret.Order = append(ret.Order, kk)
        ret.Storage[kk] = vv 
    })
    return ret
}

func (e EthChain) GetState() core.State{
    state := e.Pipe.World().State()
    stateMap := core.State{make(map[string]core.Storage), []string{}}

    trieIterator := state.Trie.NewIterator()
    trieIterator.Each(func (addr string, acct *ethutil.Value){
        hexAddr := ethutil.Bytes2Hex([]byte(addr))
        stateMap.Order = append(stateMap.Order, hexAddr)
        stateMap.State[hexAddr] = core.Storage{make(map[string]interface{}), []string{}}

        acctObj := ethstate.NewStateObjectFromBytes([]byte(addr), acct.Bytes())
        acctObj.EachStorage(func (storage string, value *ethutil.Value){
            value.Decode()
            hexStorage := ethutil.Bytes2Hex([]byte(storage))
            storageState := stateMap.State[hexAddr]
            storageState.Order = append(stateMap.State[hexAddr].Order, hexStorage)
            storageState.Storage[hexStorage] = ethutil.Bytes2Hex(value.Bytes())
            stateMap.State[hexAddr] = storageState
        })
    })
    return stateMap
}

// subscribe to an address (hex)
// returns a chanel that will fire when address is updated
func (e EthChain) Subscribe(name, event, target string){
    eth_ch := make(chan ethreact.Event, 1)
    if target != ""{
        addr := string(ethutil.Hex2Bytes(target))
        e.reactor.Subscribe("object:"+addr, eth_ch)
    } else{
        e.reactor.Subscribe(event, eth_ch)
    }

    e.Chans[name] = make(chan events.Event)
    ch := e.Chans[name]

    // fire up a goroutine and broadcast module specific chan on our main chan
    go func(){
        for {
            r := <- eth_ch           
            log.Println(r)
            ch <- events.Event{
                         Event:event,
                         Target:target,
                         Source:"monk",
                         TimeStamp:time.Now(),
                    }
        }
    }()
}

// Mine a block
func (e EthChain) Commit(){
    e.StartMining()
    _ =<- e.Chans["newBlock"]
    v := false
    for !v{
        v = e.StopMining()
    }
}

// start and stop continuous mining
func (e EthChain) AutoCommit(toggle bool){
    if toggle{
        e.StartMining()
    } else{
        e.StopMining()
    }
}

func (e EthChain) IsAutocommit() bool{
    return e.Ethereum.IsMining()
}

// send a message to a contract
func (e *EthChain) Msg(addr string, data []string){
    packed := PackTxDataArgs(data...)
    keys := e.fetchKeyPair()
    addr = ethutil.StripHex(addr)
    byte_addr := ethutil.Hex2Bytes(addr)
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big("350")), ethutil.NewValue(ethutil.Big("200000000000")), ethutil.NewValue(ethutil.Big("1000000")), packed)
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
    //fmt.Println("the amount:", amt, ethutil.Big(amt), ethutil.NewValue(amt), ethutil.NewValue(ethutil.Big(amt)))
    // note, NewValue will not turn a string int into a big int..
    start := time.Now()
    _, err := e.Pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big(amt)), ethutil.NewValue(ethutil.Big("20000000000")), ethutil.NewValue(ethutil.Big("100000")), "")
    dif := time.Since(start)
    fmt.Println("pipe tx took ", dif)
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
    // messy key system...
    // ethchain should have an 'active key'
    keys := e.fetchKeyPair()

    // well isn't this pretty! barf
    contract_addr, err := e.Pipe.Transact(keys, nil, ethutil.NewValue(ethutil.Big("271")), ethutil.NewValue(ethutil.Big("2000000000000")), ethutil.NewValue(ethutil.Big("1000000")), script)
    if err != nil{
        log.Fatal("could not deploy contract", err)
    }
    return ethutil.Bytes2Hex(contract_addr)
}

func (e *EthChain) Shutdown() error{
    e.Stop()
    return nil
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

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (e *EthChain) Name() string{
    return "monk"
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
// TODO: this is in two places, clean it up you putz
func PackTxDataArgs(args ... string) string{
    //fmt.Println("pack data:", args)
    ret := *new([]byte)
    for _, s := range args{
        if s[:2] == "0x"{
            t := s[2:]
            if len(t) % 2 == 1{
                t = "0"+t
            }
            x := ethutil.Hex2Bytes(t)
            //fmt.Println(x)
            l := len(x)
            ret = append(ret, ethutil.LeftPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            // TODO: just changed from right to left. yabadabadoooooo take care!
            ret = append(ret, ethutil.LeftPadBytes(x, 32*((l + 31)/32))...)
        }
    }
    return "0x" + ethutil.Bytes2Hex(ret)
   // return ret
}


