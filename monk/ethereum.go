package monk

import (
    "log"
    "fmt"
    "os"
    "os/user"
    "strconv"
    "io/ioutil"
    "math/big"
    "time"

    "github.com/eris-ltd/deCerver-interfaces/events"
    "github.com/eris-ltd/deCerver-interfaces/core"
    "github.com/eris-ltd/deCerver-interfaces/api"
    "github.com/eris-ltd/deCerver-interfaces/modules"

    "github.com/eris-ltd/thelonious"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkpipe"
    "github.com/eris-ltd/thelonious/monklog"
    "github.com/eris-ltd/thelonious/monkcrypto"
    "github.com/eris-ltd/thelonious/monkreact"
    "github.com/eris-ltd/thelonious/monkstate"
    "github.com/eris-ltd/thelonious/monkchain"
)

var (
    GoPath = os.Getenv("GOPATH")
    usr, _ = user.Current() // error?!
)

//Logging
var logger *monklog.Logger = monklog.NewLogger("EthChain(deCerver)")

// implements decerver-interfaces Module
type MonkModule struct{
    monk *Monk

	wsAPIServiceFactory api.WsAPIServiceFactory
	httpAPIService      interface{}
	eventReg events.EventRegistry
}

// implements decerver-interfaces Blockchain 
// TODO: privatize everything bitch
type Monk struct{
    Config *ChainConfig
    Ethereum *eth.Ethereum
    Pipe *monkpipe.Pipe
    keyManager *monkcrypto.KeyManager
    reactor *monkreact.ReactorEngine
    started bool
    Chans map[string]chan events.Event
}

/*
    First, the functions to satisfy Module
*/

// new monkchain with default config
// it allows you to pass in an etheruem instance
// btu it will not start a new one otherwise
// this gives you a chance to set config options after
//      creating the EthChain
func NewMonk(ethereum *eth.Ethereum) *MonkModule{
    e := new(MonkModule)
    m := new(Monk)
    // here we load default config and leave it to caller
    // to read a config file to overwrite
    m.Config = DefaultConfig
    if ethereum != nil{
        m.Ethereum = ethereum
    }
    m.started = false
    e.monk = m
    return e
}

// register the module with the decerver javascript vm
func (mod *MonkModule) Register(fileIO core.FileIO, registry api.ApiRegistry, runtime core.Runtime, eReg events.EventRegistry) error{
    return nil
}

// initialize an monkchain
// it may or may not already have an ethereum instance
// basically gives you a pipe, local keyMang, and reactor
func (mod *MonkModule) Init() error{
    m := mod.monk
    // if didn't call NewEth
    if m.Config == nil{
        m.Config = DefaultConfig
    }
    // if no ethereum instance
    if m.Ethereum == nil{
        m.EthConfig()
        m.NewEthereum()
    }

    // public interface
    pipe := monkpipe.New(m.Ethereum) 
    // load keys from file. genesis block keys. convenient for testing
    LoadKeys(m.Config.KeyFile, m.Ethereum.KeyManager())

    m.Pipe = pipe
    m.keyManager = m.Ethereum.KeyManager()
    m.reactor = m.Ethereum.Reactor()

    // subscribe to the new block
    m.Chans = make(map[string]chan events.Event)
    m.Subscribe("newBlock", "newBlock", "")

    log.Println(m.Ethereum.Port)
    
    return nil
}

// start the ethereum node
func (mod *MonkModule) Start() error{
    m := mod.monk
    m.Ethereum.Start(true) // peer seed
    m.started = true

    if m.Config.Mining{
        StartMining(m.Ethereum)
    }
    return nil
}

func (mod *MonkModule) Shutdown() error{
    mod.monk.Stop()
    return nil
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *MonkModule) Name() string{
    return "monk"
}

/*
    Wrapper so module satisfies Blockchain
*/

func (mod *MonkModule) GetWorldState() modules.WorldState{
    return mod.monk.GetWorldState()
}

func (mod *MonkModule) GetState() modules.State{
    return mod.monk.GetState()
}

func (mod *MonkModule) GetStorage(target string) modules.Storage{
    return mod.monk.GetStorage(target)
}

func (mod *MonkModule) GetAccount(target string) modules.Account{
    return mod.monk.GetAccount(target)
}

func (mod *MonkModule) GetStorageAt(target, storage string) string{
    return mod.monk.GetStorageAt(target, storage)
}

func (mod *MonkModule) GetBlockCount() int{
    return mod.monk.GetBlockCount()
}

func (mod *MonkModule) GetLatestBlock() string{
    return mod.monk.GetLatestBlock()
}

func (mod *MonkModule) GetBlock(hash string) Block{
    return mod.monk.GetBlock(hash)
}

func (mod *MonkModule) IsScript(target string) bool{
    return mod.monk.IsScript(target)
}

func (mod *MonkModule) Tx(addr, amt string){
    mod.monk.Tx(addr, amt)
}

func (mod *MonkModule) Msg(addr string, data []string){
    mod.monk.Msg(addr, data)
}

func (mod *MonkModule) Script(file, lang string) string { 
    return mod.monk.Script(file, lang)
}

func (mod *MonkModule) Commit(){
    mod.monk.Commit()
}

func (mod *MonkModule) AutoCommit(toggle bool){
    mod.monk.AutoCommit(toggle)
}

func (mod *MonkModule) IsAutocommit() bool{
    return mod.monk.IsAutocommit()
}

/*
    Module should also satisfy KeyManager
*/

func (mod *MonkModule) GetActiveAddress() string{
    return mod.monk.GetActiveAddress()
}

func (mod *MonkModule) GetAddress(n int) (string, error){
    return mod.monk.GetAddress(n)
}

func (mod *MonkModule) SetAddress(addr string) error{
    return mod.monk.SetAddress(addr)
}

func (mod *MonkModule) SetAddressN(n int) error{
    return mod.monk.SetAddressN(n)
}

func (mod *MonkModule) NewAddress(set bool) string{
    return mod.monk.NewAddress(set)
}

func (mod *MonkModule) AddressCount() int{
    return mod.monk.AddressCount()
}


/*
    Implement Blockchain
*/

func (monk *Monk) GetWorldState() WorldState{
    state := monk.Pipe.World().State()
    stateMap := modules.State{make(map[string]modules.Account), []string{}}

    trieIterator := state.Trie.NewIterator()
    trieIterator.Each(func (addr string, acct *monkutil.Value){
        hexAddr := monkutil.Bytes2Hex([]byte(addr))
        stateMap.Order = append(stateMap.Order, hexAddr)
        stateMap.Accounts[hexAddr] = monk.GetAccount(hexAddr)

    })
    return stateMap
}

func (monk *Monk) GetState() modules.State{
    state := monk.Pipe.World().State()
    stateMap := modules.State{make(map[string]modules.Storage), []string{}}

    trieIterator := state.Trie.NewIterator()
    trieIterator.Each(func (addr string, acct *monkutil.Value){
        hexAddr := monkutil.Bytes2Hex([]byte(addr))
        stateMap.Order = append(stateMap.Order, hexAddr)
        stateMap.State[hexAddr] = monk.GetStorage(hexAddr)

    })
    return stateMap
}

func (monk *Monk) GetStorage(addr string) modules.Storage{
    w := monk.Pipe.World()
    obj := w.SafeGet(monkutil.UserHex2Bytes(addr)).StateObject
    ret := modules.Storage{make(map[string]string), []string{}}
    obj.EachStorage(func(k string, v *monkutil.Value){
        kk := monkutil.Bytes2Hex([]byte(k))
        vv := monkutil.Bytes2Hex(v.Bytes())
        ret.Order = append(ret.Order, kk)
        ret.Storage[kk] = vv 
    })
    return ret
}

func (monk *Monk) GetAccount(target string) Account{
    w := monk.Pipe.World()
    obj := w.SafeGet(monkutil.UserHex2Bytes(addr)).StateObject

    bal := obj.Balance.String()
    nonce := strconv.Itoa(obj.Nonce)
    script := monkutil.Bytes2Hex(obj.Code)
    storage := monk.GetStorage(target)
    isscript := len(storage.Order) > 0 || len(script) > 0

    return modules.Account{
        Address: target,
        Balance: bal,
        Nonce: nonce,
        Script: script,
        Storage: storage,
        IsScript: isscript,
    }
}

func (monk *Monk) GetStorageAt(contract_addr string, storage_addr string) string{
    var saddr *big.Int
    if monkutil.IsHex(storage_addr){
        saddr = monkutil.BigD(monkutil.Hex2Bytes(monkutil.StripHex(storage_addr)))
    } else {
        saddr = monkutil.Big(storage_addr)
    }

    contract_addr = monkutil.StripHex(contract_addr)
    caddr := monkutil.Hex2Bytes(contract_addr)
    //saddr := monkutil.Hex2Bytes(storage_addr)
    w := monk.Pipe.World()
    ret := w.SafeGet(caddr).GetStorage(saddr)
    //ret := e.Pipe.Storage(caddr, saddr) 
    //returns an ethValue
    // TODO: figure it out!
    //val := BigNumStrToHex(ret)
    if ret.IsNil(){
        return "0x"
    }
    return monkutil.Bytes2Hex(ret.Bytes())
}

func (monk *Monk) GetBlockCount() int{
    return monk.Ethereum.Blockchain().LastBlockNumber
}

func (monk *Monk) GetLatestBlock() string{
   return monkutil.Bytes2Hex(monk.Ethereum.Blockchain().LastBlockHash)
}

func (monk *Monk) GetBlock(hash string) Block{
   hashBytes := monkutil.Hex2Bytes(hash) 
   block := monk.Ethereum.Blockchain().GetBlock(hashBytes)
    
   // TODO: convert eth block to module block

}

func (monk *Monk) IsScript(target string) bool{
    // is contract if storage is empty and no bytecode
    obj := monk.GetAccount(target)
    storage := obj.Storage 
    if len(storage.Order) == 0 && obj.Script == ""{
        return false
    }
    return true
}

// send a tx
func (monk *Monk) Tx(addr, amt string) (string, error){
    keys := monk.fetchKeyPair()
    addr = monkutil.StripHex(addr)
    if addr[:2] == "0x"{
        addr = addr[2:]
    }
    byte_addr := monkutil.Hex2Bytes(addr)
    // note, NewValue will not turn a string int into a big int..
    start := time.Now()
    hash, err := monk.Pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big(amt)), monkutil.NewValue(monkutil.Big("20000000000")), monkutil.NewValue(monkutil.Big("100000")), "")
    dif := time.Since(start)
    fmt.Println("pipe tx took ", dif)
    if err != nil{
        return "", err
    }
    return monkutil.Bytes2Hex(hash), nil
}

// send a message to a contract
func (monk *Monk) Msg(addr string, data []string) (string, error){
    packed := PackTxDataArgs(data...)
    keys := monk.fetchKeyPair()
    addr = monkutil.StripHex(addr)
    byte_addr := monkutil.Hex2Bytes(addr)
    hash, err := monk.Pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big("350")), monkutil.NewValue(monkutil.Big("200000000000")), monkutil.NewValue(monkutil.Big("1000000")), packed)
    if err != nil{
        return "", err
    }
    return monkutil.Bytes2Hex(hash), nil
}

func (monk *Monk) Script(file, lang string) (string, error){
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
    // monkchain should have an 'active key'
    keys := monk.fetchKeyPair()

    // well isn't this pretty! barf
    contract_addr, err := monk.Pipe.Transact(keys, nil, monkutil.NewValue(monkutil.Big("271")), monkutil.NewValue(monkutil.Big("2000000000000")), monkutil.NewValue(monkutil.Big("1000000")), script)
    if err != nil{
        return "", err
    }
    return monkutil.Bytes2Hex(contract_addr), nil
}



// subscribe to an address (hex)
// returns a chanel that will fire when address is updated
// TODO: cast resource to proper interfaces
func (monk *Monk) Subscribe(name, event, target string) chan events.Event{
    eth_ch := make(chan monkreact.Event, 1)
    if target != ""{
        addr := string(monkutil.Hex2Bytes(target))
        monk.reactor.Subscribe("object:"+addr, eth_ch)
    } else{
        monk.reactor.Subscribe(event, eth_ch)
    }

    monk.Chans[name] = make(chan events.Event)
    ch := monk.Chans[name]

    // fire up a goroutine and broadcast module specific chan on our main chan
    go func(){
        for {
            r := <- eth_ch           
            ch <- events.Event{
                         Event:event,
                         Target:target,
                         Source:"monk",
                         Resource: r,
                         TimeStamp:time.Now(),
                    }
        }
    }()
    return ch
}


// Mine a block
func (m *Monk) Commit(){
    m.StartMining()
    _ =<- m.Chans["newBlock"]
    v := false
    for !v{
        v = m.StopMining()
    }
}

// start and stop continuous mining
func (m *Monk) AutoCommit(toggle bool){
    if toggle{
        m.StartMining()
    } else{
        m.StopMining()
    }
}

func (m *Monk) IsAutocommit() bool{
    return m.Ethereum.IsMining()
}

/*
    Blockchain interface should also satisfy KeyManager
*/

func (monk *Monk) GetActiveAddress() string{
    keypair := monk.keyManager.KeyPair()
    addr := monkutil.Bytes2Hex(keypair.Address())
    return addr
}

func (monk *Monk) GetAddress(n int) (string, error){
    ring := monk.keyManager.KeyRing()
    if n >= ring.Len(){
	    return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
    }
    pair := ring.GetKeyPair(n)
    addr := monkutil.Bytes2Hex(pair.Address())
    return addr, nil
}

func (monk *Monk) SetAddress(addr string) error{
    n := -1
    i := 0
    ring := monk.keyManager.KeyRing()
    ring.Each(func(kp *monkcrypto.KeyPair){
        a := monkutil.Bytes2Hex(kp.Address())
        if a == addr{
            n = i
        }
        i += 1
    })
    if n == -1{
        return fmt.Errorf("Address %s not found in keyring", addr)
    }
    return monk.SetAddressN(n)
}

func (monk *Monk) SetAddressN(n int) error{
    return monk.keyManager.SetCursor(n)
}

func (monk *Monk) NewAddress(set bool) string{
    newpair := monkcrypto.GenerateNewKeyPair()
    addr := monkutil.Bytes2Hex(newpair.Address())
    ring := monk.keyManager.KeyRing()
    ring.AddKeyPair(newpair)
    if set{
        monk.SetAddressN(ring.Len()-1)
    }
    return addr
}

func (monk *Monk) AddressCount() int{
    return monk.keyManager.KeyRing().Len()
}


/*
    Helper functions
*/

// create a new ethereum instance
// expects EthConfig to already have been called!
// init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
func (m *Monk) NewEthereum(){
    db := NewDatabase(m.Config.DbName)

    keyManager := NewKeyManager(m.Config.KeyStore, m.Config.DataDir, db)
    keyManager.Init("", 0, true)
    m.keyManager = keyManager

    clientIdentity := NewClientIdentity(m.Config.ClientIdentifier, m.Config.Version, m.Config.Identifier) 

    // create the ethereum obj
    ethereum, err := eth.New(db, clientIdentity, m.keyManager, eth.CapDefault, false)

    if err != nil {
        log.Fatal("Could not start node: %s\n", err)
    }

    ethereum.Port = strconv.Itoa(m.Config.Port)
    ethereum.MaxPeers = m.Config.MaxPeers

    m.Ethereum = ethereum
}

// returns hex addr of gendoug
func (monk *Monk) GenDoug() string{
    return monkutil.Bytes2Hex(monkchain.GENDOUG)
}

// TODO: return hex string
func (monk *Monk) _GetStorage(contract_addr string) map[string]*monkutil.Value{
    acct := monk.Pipe.World().SafeGet(monkutil.Hex2Bytes(contract_addr)).StateObject
    m := make(map[string]*monkutil.Value)
    acct.EachStorage(func(k string, v *monkutil.Value){
            kk := monkutil.Bytes2Hex([]byte(k))
            fmt.Println("each storage", v)
            fmt.Println("each storage val", v.Val)
            m[kk] = v
        })
   return m 
}

func (monk *Monk) StartMining() bool{
    return StartMining(monk.Ethereum)
}

func (monk *Monk) StopMining() bool{
    return StopMining(monk.Ethereum)
}

func (monk *Monk) StartListening(){
    monk.Ethereum.StartListening()
}

func (monk *Monk) StopListening() {
    monk.Ethereum.StopListening()
}

/*
    some key management stuff
*/

//TODO: deprecate (now GetActiveAddress)
func (monk *Monk) FetchAddr() string{
    keypair := monk.keyManager.KeyPair()
    pub := monkutil.Bytes2Hex(keypair.Address())
    return pub
}

func (monk *Monk) fetchPriv() string{
    keypair := monk.keyManager.KeyPair()
    priv := monkutil.Bytes2Hex(keypair.PrivateKey)
    return priv
}

func (monk *Monk) fetchKeyPair() *monkcrypto.KeyPair{
    return monk.keyManager.KeyPair()
}

// this is bad but I need it for testing
// TODO: deprecate!
func (monk *Monk) FetchPriv() string{
    return monk.fetchPriv()
}

// switch current key
// TODO: deprecate (SetAddressN)
func (monk *Monk) SetCursor(n int){
    monk.keyManager.SetCursor(n)
}


func (monk *Monk) Stop(){
    if !monk.started{
        fmt.Println("can't stop: haven't even started...")
        return
    }
    monk.StopMining()
    fmt.Println("stopped mining")
    monk.Ethereum.Stop()
    fmt.Println("stopped ethereum")
    monk = &Monk{Config: monk.Config}
    monklog.Reset()
}


// compile LLL file into evm bytecode 
// returns hex
func CompileLLL(filename string) string{
    code, err := monkutil.CompileLLL(filename)
    if err != nil{
        fmt.Println("error compiling lll!", err)
        return ""
    }
    return "0x"+monkutil.Bytes2Hex(code)
}

// some convenience functions

// get users home directory
func homeDir() string{
    usr, _ := user.Current()
    return usr.HomeDir
}

// convert a big int from string to hex
func BigNumStrToHex(s string) string{
    bignum := monkutil.Big(s)
    bignum_bytes := monkutil.BigToBytes(bignum, 16)
    return monkutil.Bytes2Hex(bignum_bytes)
}

// takes a string, converts to bytes, returns hex
func SHA3(tohash string) string{
    h := monkcrypto.Sha3Bin([]byte(tohash))
    return monkutil.Bytes2Hex(h)
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
            x := monkutil.Hex2Bytes(t)
            //fmt.Println(x)
            l := len(x)
            ret = append(ret, monkutil.LeftPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            // TODO: just changed from right to left. yabadabadoooooo take care!
            ret = append(ret, monkutil.LeftPadBytes(x, 32*((l + 31)/32))...)
        }
    }
    return "0x" + monkutil.Bytes2Hex(ret)
   // return ret
}


