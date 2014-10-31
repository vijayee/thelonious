package ethchain

import (
    "fmt"
    "os"
    "math/big"
    "path"
    "io/ioutil"
    "github.com/eris-ltd/thelonious/ethutil"    
    "github.com/eris-ltd/thelonious/ethstate"    
    "github.com/eris-ltd/thelonious/ethtrie"    
)

var (

    GoPath = os.Getenv("GOPATH")

    DougPath = "" // lets us set the doug contract post config load. Convenient for testing

    // overwritten by monk/config.go
    DougDifficulty = ethutil.BigPow(2, 17)  // for mining speed
    ContractPath = path.Join(GoPath, "src", "github.com", "eris-ltd", "eris-std-lib")
    GenesisConfig = path.Join(GoPath, "src", "github.com", "eris-ltd", "thelonious", "monk", "genesis.json")
    // if GenesisConfig is invalid ...
    defaultGenesisConfig = path.Join(GoPath, "src", "github.com", "eris-ltd", "thelonious", "monk", "genesis.json")
)

// called by setLastBlock when a new blockchain is created
// ie. Load a genesis.json and deploy
// if GENDOUG is nil, simply bankroll the accounts (no doug)
func GenesisPointer(block *Block){
    g := LoadGenesis()

    fmt.Println("PRE DEPLOY")
    fmt.Println("GENDOUG", GENDOUG)
    if GENDOUG != nil{
        g.Deploy(block)
    } else{
        // no genesis doug, deploy simple
        for _, account := range g.Accounts{
            // direct state modification to create accounts and balances
            AddAccount(account.ByteAddr, account.Balance, block)
        }
        // update and commit state
        block.State().Update()  
        block.State().Sync()  
    }
}


/*
    Model is a global variable set at eth startup
    DougValidate and DougValue are our windows into the model
*/
func SetDougModel(model string){
    switch(model){
        case "fake":
            Model = NewFakeModel()
        case "dennis":
            Model = NewGenDougModel()
        case "std":
            Model = NewStdLibModel()
        default:
            Model = nil 
    }
}

// use gendoug and permissions model to validate addr's role
func DougValidate(addr []byte, state *ethstate.State, role string) bool{
    if GENDOUG == nil || Model == nil{
        return true
    }

    if Model == nil{
        return false
    }
    return Model.HasPermission(addr, role, state)
}

// look up a special doug param
func DougValue(key, namespace string, state *ethstate.State) []byte{
    if GENDOUG == nil{
        return nil 
    }
    return Model.GetValue(key, namespace, state)
}


/*
    Functions for setting for loading the genesis contract
    and processing the state changes
*/

// create a new tx from a script, with dummy keypair
// creates tx but does not sign!
func NewGenesisContract(scriptFile string) *Transaction{
    // if mutan, load the script. else, pass file name
    var s string
    if scriptFile[len(scriptFile)-3:] == ".mu"{
        r, err := ioutil.ReadFile(scriptFile)
        if err != nil{
            fmt.Println("could not load contract!", scriptFile, err)
            os.Exit(0)
        }
        s = string(r)
    } else{
        s = scriptFile
    }
    script, err := ethutil.Compile(string(s), false) 
    if err != nil{
        fmt.Println("failed compile", err)
        os.Exit(0)
    }
    //fmt.Println("script: ", script)

    // create tx
    tx := NewContractCreationTx(ethutil.Big("543"), ethutil.Big("10000"), ethutil.Big("10000"), script)
    //tx.Sign(keys.PrivateKey)

    return tx
}

// apply tx to genesis block
func SimpleTransitionState(addr []byte, block *Block, tx *Transaction) *Receipt{
    state := block.State()
    st := NewStateTransition(ethstate.NewStateObject(block.Coinbase), tx, state, block)
    st.AddGas(ethutil.Big("10000000000000000000000000000000000000000000000000000000000000000000000000000000000")) // gas is silly, but the vm needs it

    var script []byte
    receiver := state.GetOrNewStateObject(addr)
    if tx.CreatesContract(){    
        receiver.Balance = ethutil.Big("123456789098765432")
        receiver.InitCode = tx.Data
        receiver.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))
        script = receiver.Init()
    } else{
        script = receiver.Code
    }

    sender := state.GetOrNewStateObject(tx.Sender())  
    value := ethutil.Big("12342")

    msg := state.Manifest().AddMessage(&ethstate.Message{
        To: receiver.Address(), From: sender.Address(),
        Input:  tx.Data,
        Origin: sender.Address(),
        Block:  block.Hash(), Timestamp: block.Time, Coinbase: block.Coinbase, Number: block.Number,
        Value: value,
    })
    // TODO: this should switch on creates contract (init vs code) ?
    ret, err := st.Eval(msg, script, receiver, "init")
    if err != nil{
        fmt.Println("Eval error in simple transition state:", err)
        os.Exit(0)
    }
    if tx.CreatesContract(){
        receiver.Code = ret
    }
    msg.Output = ret

    rootI := state.Root()
    var root []byte
    if r, ok := rootI.([]byte); ok{
        root = r 
    } else if r, ok := rootI.(string); ok{
        root = []byte(r)
    }

    receipt := &Receipt{tx, ethutil.CopyBytes(root), new(big.Int)}
    // remove stateobject used to deploy gen doug
    state.DeleteStateObject(sender)    
    return receipt
}

/*
    sigh...
*/

func PrettyPrintAccount(obj *ethstate.StateObject){
    fmt.Println("Address", ethutil.Bytes2Hex(obj.Address())) //ethutil.Bytes2Hex([]byte(addr)))
    fmt.Println("\tNonce", obj.Nonce)
    fmt.Println("\tBalance", obj.Balance)
    if true { // only if contract, but how?!
        fmt.Println("\tInit", ethutil.Bytes2Hex(obj.InitCode))
        fmt.Println("\tCode", ethutil.Bytes2Hex(obj.Code))
        fmt.Println("\tStorage:")
        obj.EachStorage(func(key string, val *ethutil.Value){
            val.Decode()
            fmt.Println("\t\t", ethutil.Bytes2Hex([]byte(key)), "\t:\t", ethutil.Bytes2Hex([]byte(val.Str())))
        }) 
    }
}
/*

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *ethchain.Block){
    state := block.State()
    it := state.Trie.NewIterator()   
    it.Each(func(key string, value *ethutil.Value) {  
        addr := ethutil.Address([]byte(key))
//        obj := ethstate.NewStateObjectFromBytes(addr, value.Bytes())
        obj := block.State().GetAccount(addr)
        PrettyPrintAccount(obj)
    })
}

*/
