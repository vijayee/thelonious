package ethchain

import (
    "fmt"
    "os"
    "math/big"
    "path"
    "io/ioutil"
    "github.com/eris-ltd/eth-go-mods/ethutil"    
    "github.com/eris-ltd/eth-go-mods/ethcrypto"    
    "github.com/eris-ltd/eth-go-mods/ethstate"    
    "github.com/eris-ltd/eth-go-mods/ethtrie"    
)

var (

    DougDifficulty = ethutil.BigPow(2, 12)  // for mining speed

    GENDOUG []byte = nil // dougs address
    MINERS = "01"
    TXERS = "02"

    SETDOUG = "" // lets us set the doug contract post config load
    SETDOUGADDR []byte = nil // lets us set the doug contract addr post config load

    GoPath = os.Getenv("GOPATH")
    ContractPath = path.Join(GoPath, "src", "github.com", "eris-ltd", "eth-go-mods", "ethtest", "contracts")
    GenesisConfig = path.Join(GoPath, "src", "github.com", "eris-ltd", "eth-go-mods", "ethtest", "genesis.json")
)

// this is what gets called by NewEris() to launch a genesis block
// do what you gotta do from here
// ie. Load a genesis.json and deploy
func GenesisPointer(block *Block, eth EthManager){
    g := LoadGenesis()

    // check doug address validity (addr length is at least 20)
    if len(g.Address) > 20 {
        if g.Address[:2] == "0x"{
            GENDOUG = ethutil.Hex2Bytes(g.Address[2:])
        } else{
            GENDOUG = []byte(g.Address)
        }
        GENDOUG = GENDOUG[:20]
    }

    // f might be useful if we want to do something special?
    g.Deploy(block, eth)
}

// use genesis block to validate addr's role
// TODO: bring up to date
func DougValidate(addr []byte, state *ethstate.State, role string) bool{
    if GENDOUG == nil{
        return true
    }
    //fmt.Println("validating addr for role", role)
    genDoug := state.GetStateObject(GENDOUG)

    var N string
    switch(role){
        case "tx":
            N = TXERS
        case "miner":
            N = MINERS
        default:
            return false
    }

    
    caddr := genDoug.GetStorage(ethutil.BigD(ethutil.Hex2Bytes(N)))
    c := state.GetOrNewStateObject(caddr.Bytes())

    valid := c.GetStorage(ethutil.BigD(addr))
    fmt.Println(valid)

    return !valid.IsNil()
}

// create a new tx from a script, with dummy keypair
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
    fmt.Println("script: ", script)
    // dummy keys for signing
    keys := ethcrypto.GenerateNewKeyPair() 

    // create tx
    tx := NewContractCreationTx(ethutil.Big("543"), ethutil.Big("10000"), ethutil.Big("10000"), script)
    tx.Sign(keys.PrivateKey)

    return tx
}

// apply tx to genesis block
func SimpleTransitionState(addr []byte, block *Block, tx *Transaction) *Receipt{
    state := block.State()
    st := NewStateTransition(ethstate.NewStateObject(block.Coinbase), tx, state, block)
    st.AddGas(ethutil.Big("10000000000000000000000000000000000000000000000000000000000000000000000000000000000")) // gas is silly, but the vm needs it

    fmt.Println("man oh man", ethutil.Bytes2Hex(addr))
    receiver := state.NewStateObject(addr)
    receiver.Balance = ethutil.Big("123456789098765432")
    receiver.InitCode = tx.Data
    receiver.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))
    sender := state.GetOrNewStateObject(tx.Sender())  
    value := ethutil.Big("12342")

    msg := state.Manifest().AddMessage(&ethstate.Message{
        To: receiver.Address(), From: sender.Address(),
        Input:  tx.Data,
        Origin: sender.Address(),
        Block:  block.Hash(), Timestamp: block.Time, Coinbase: block.Coinbase, Number: block.Number,
        Value: value,
    })
    code, err := st.Eval(msg, receiver.Init(), receiver, "init")
    if err != nil{
        fmt.Println("Eval error in simple transition state:", err)
        os.Exit(0)
    }
    receiver.Code = code
    msg.Output = code

    receipt := &Receipt{tx, ethutil.CopyBytes(state.Root().([]byte)), new(big.Int)}
    return receipt
}

