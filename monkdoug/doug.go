package monkdoug

import (
    "fmt"
    "os"
    "math/big"
    "path"
    "io/ioutil"
    "github.com/eris-ltd/thelonious/monkutil"    
    "github.com/eris-ltd/thelonious/monkstate"    
    "github.com/eris-ltd/thelonious/monktrie"    
    "github.com/eris-ltd/thelonious/monkchain"
)

var (
    GoPath = os.Getenv("GOPATH")
    ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")
)

// Load a genesis.json and deploy
// Called from monkchain by setLastBlock when a new blockchain is created
// If it can't find the file, fail
//TODO: deprecate
func GenesisPointer(jsonFile string, block *monkchain.Block){
    _, err := os.Stat(jsonFile)
    if err != nil{
        fmt.Printf("Genesis.json file %s could not be found")
        os.Exit(0)
    }
    gson := LoadGenesis(jsonFile)
    gson.Deploy(block)
}


/*
   Model is a global variable set at eth startup
    DougValidate and DougValue are our windows into the model
*/
func NewPermModel(modelName string, dougAddr []byte) (model monkchain.GenDougModel){
    switch(modelName){
        case "fake":
            model = NewFakeModel(dougAddr)
        case "dennis":
            model = NewGenDougModel(dougAddr)
        case "std":
            model = NewStdLibModel(dougAddr)
        case "yes":
            model = NewYesModel()
        case "no":
            model = NewNoModel()
        default:
            fmt.Println("shitty default")
            model = NewYesModel()
    }
    return 
}


/*
    TODO: deprecate
type GenDoug struct{

}

// use gendoug and permissions model to validate addr's role
func (g *GenDoug) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    if GenDougByteAddr == nil || Model == nil{
        return true
    }

    if Model == nil{
        return false
    }
    return Model.HasPermission(addr, role, state)
}

// look up a special doug param
func (g *GenDoug) ValidateValue(name string, value interface{}, state *monkstate.State) bool { //[]byte{
    return true
    if GENDOUG == nil{
        return nil 
    }
    return Model.GetValue(key, namespace, state)
}
*/

/*
    Functions for setting for loading the genesis contract
    and processing the state changes
*/

// create a new tx from a script, with dummy keypair
// creates tx but does not sign!
func NewGenesisContract(scriptFile string) *monkchain.Transaction{
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
    script, err := monkutil.Compile(string(s), false) 
    if err != nil{
        fmt.Println("failed compile", err)
        os.Exit(0)
    }
    //fmt.Println("script: ", script)

    // create tx
    tx := monkchain.NewContractCreationTx(monkutil.Big("543"), monkutil.Big("10000"), monkutil.Big("10000"), script)
    //tx.Sign(keys.PrivateKey)

    return tx
}

// apply tx to genesis block
func SimpleTransitionState(addr []byte, block *monkchain.Block, tx *monkchain.Transaction) *monkchain.Receipt{
    state := block.State()
    st := monkchain.NewStateTransition(monkstate.NewStateObject(block.Coinbase), tx, state, block)
    st.AddGas(monkutil.Big("10000000000000000000000000000000000000000000000000000000000000000000000000000000000")) // gas is silly, but the vm needs it

    var script []byte
    receiver := state.GetOrNewStateObject(addr)
    if tx.CreatesContract(){    
        receiver.Balance = monkutil.Big("123456789098765432")
        receiver.InitCode = tx.Data
        receiver.State = monkstate.New(monktrie.New(monkutil.Config.Db, ""))
        script = receiver.Init()
    } else{
        script = receiver.Code
    }

    sender := state.GetOrNewStateObject(tx.Sender())  
    value := monkutil.Big("12342")

    msg := state.Manifest().AddMessage(&monkstate.Message{
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

    receipt := &monkchain.Receipt{tx, monkutil.CopyBytes(root), new(big.Int)}
    // remove stateobject used to deploy gen doug
    state.DeleteStateObject(sender)    
    return receipt
}

/*
    sigh...
*/

func PrettyPrintAccount(obj *monkstate.StateObject){
    fmt.Println("Address", monkutil.Bytes2Hex(obj.Address())) //monkutil.Bytes2Hex([]byte(addr)))
    fmt.Println("\tNonce", obj.Nonce)
    fmt.Println("\tBalance", obj.Balance)
    if true { // only if contract, but how?!
        fmt.Println("\tInit", monkutil.Bytes2Hex(obj.InitCode))
        fmt.Println("\tCode", monkutil.Bytes2Hex(obj.Code))
        fmt.Println("\tStorage:")
        obj.EachStorage(func(key string, val *monkutil.Value){
            val.Decode()
            fmt.Println("\t\t", monkutil.Bytes2Hex([]byte(key)), "\t:\t", monkutil.Bytes2Hex([]byte(val.Str())))
        }) 
    }
}
/*

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *monkchain.Block){
    state := block.State()
    it := state.Trie.NewIterator()   
    it.Each(func(key string, value *monkutil.Value) {  
        addr := monkutil.Address([]byte(key))
//        obj := monkstate.NewStateObjectFromBytes(addr, value.Bytes())
        obj := block.State().GetAccount(addr)
        PrettyPrintAccount(obj)
    })
}

*/
