package monkdoug

import (
    "fmt"
    "path"
    "os"
    "strconv"
    "io/ioutil"
    "encoding/json"
    "github.com/eris-ltd/thelonious/monkutil"    
    "github.com/eris-ltd/thelonious/monkcrypto"    
    "github.com/eris-ltd/thelonious/monkchain"
)

/*
    Configure a new genesis block from genesis.json
    Deploy the genesis block
*/

type Account struct{
    Address string `json:"address"`
    ByteAddr []byte  //convenience, but not from json
    Name string `json:"name"`
    Balance string  `json:"balance"`
    Permissions map[string]int `json:"permissions"`
    Stake int `json:"stake"`
}

type GenesisConfig struct{
    // MetaGenDoug
    Address string  `json:"address"` // bytes
    DougPath string `json:"doug"`
    ModelName string `json:"model"`
    NoGenDoug bool `json:"no-gendoug"`
    
    HexAddr string 
    ByteAddr []byte
    ContractPath string 

    // Global GenDoug Singles
    Consensus string `json:"consensus"`
    PublicMine int `json:"public:mine"`
    PublicCreate int `json:"public:create"`
    PublicTx int `json:"public:tx"`
    MaxGasTx string `json:"maxgastx"`
    BlockTime int `json:"blocktime"`

    // Accounts (permissions and stake)
    Accounts []*Account `json:"accounts"`

    Model monkchain.GenDougModel
}

// Load the genesis block info from genesis.json
func LoadGenesis(file string) *GenesisConfig{
    fmt.Println("reading ", file)
    b, err := ioutil.ReadFile(file)
    if err != nil{
        fmt.Println("err reading genesis.json", err)
        os.Exit(0)
    }

    g := new(GenesisConfig)
    err = json.Unmarshal(b, g)
    if err != nil{
        fmt.Println("error unmarshalling genesis.json", err)
        os.Exit(0)
    }

    // move address into accounts, in bytes
    for _, acc := range g.Accounts{
        acc.ByteAddr = monkutil.UserHex2Bytes(acc.Address)
    }

    g.ByteAddr = []byte(g.Address)
    g.HexAddr = monkutil.Bytes2Hex(g.ByteAddr)
    g.ContractPath = path.Join(ErisLtd, "eris-std-lib")

    /* TODO: deprecate
    // if the global DougPath is set, overwrite the config file
    if GenesisConfig.DougPath != ""{
        g.DougPath = DougPath
    }
    // if the global GenDougByteAddr is set, overwrite the config file
    if GenDougByteAddr != nil{
        g.Address = string(GenDougByteAddr)
    }
    
    if NoGenDoug{
        GenDougByteAddr = nil
        Model = nil
    } else{
        // check doug address validity (addr length is at least 20)
        if len(g.Address) >= 20 {
            if g.Address[:2] == "0x"{
                GenDougByteAddr = monkutil.Hex2Bytes(g.Address[2:])
            } else{
                GenDougByteAddr = []byte(g.Address)
            }
            GenDougByteAddr = GenDougByteAddr[:20]
        }
    }*/

    // set doug model
    g.Model = NewPermModel(g.ModelName, g.ByteAddr)

    return g
}

// Deploy the genesis block
// Converts the GenesisConfiginfo into a populated and functional doug contract in the genesis block
// if GenDougByteAddr is nil, simply bankroll the accounts (no doug)
func (g *GenesisConfig) Deploy(block *monkchain.Block){
    defer func(){
        block.State().Update()  
        block.State().Sync()  
    }()
    
    if g.NoGenDoug {
        // no genesis doug, deploy simple
        for _, account := range g.Accounts{
            // direct state modification to create accounts and balances
            AddAccount(account.ByteAddr, account.Balance, block)
        }
        return
    }

    fmt.Println("###DEPLOYING DOUG", g.Address, g.DougPath)

    // dummy keys for signing
    keys := monkcrypto.GenerateNewKeyPair() 

    // create the genesis doug
    codePath := path.Join(g.ContractPath, g.DougPath)
    genAddr := []byte(g.Address)
    MakeApplyTx(codePath, genAddr, nil, keys, block)

    // set the global vars
    g.Model.SetValue(genAddr, []string{"setvar", "consensus", g.Consensus}, keys, block)
    g.Model.SetValue(genAddr, []string{"setvar", "public:mine", "0x"+strconv.Itoa(g.PublicMine)}, keys, block)
    g.Model.SetValue(genAddr, []string{"setvar", "public:create", "0x"+strconv.Itoa(g.PublicCreate)}, keys, block)
    g.Model.SetValue(genAddr, []string{"setvar", "public:tx", "0x"+strconv.Itoa(g.PublicTx)}, keys, block)
    g.Model.SetValue(genAddr, []string{"setvar", "maxgastx", g.MaxGasTx}, keys, block)
    g.Model.SetValue(genAddr, []string{"setvar", "blocktime", "0x"+strconv.Itoa(g.BlockTime)}, keys, block)

    fmt.Println("done genesis. setting perms...")

    // set balances and permissions
    for _, account := range g.Accounts{
        // direct state modification to create accounts and balances
        AddAccount(account.ByteAddr, account.Balance, block)
        if g.Model != nil{
            // issue txs to set perms according to the model
            g.Model.SetPermissions(account.ByteAddr, account.Permissions, block, keys)

            g.Model.SetValue(g.ByteAddr, []string{"addminer", account.Name, account.Address, "0x"+strconv.Itoa(account.Stake)}, keys, block)
        }
    }
}

// set balance of an account (does not commit)
func AddAccount(addr []byte, balance string, block *monkchain.Block){
    account := block.State().GetAccount(addr)
    account.Balance = monkutil.Big(balance) //monkutil.BigPow(2, 200)
    block.State().UpdateStateObject(account)
}

// make and apply an administrative tx (simplified vm processing)
// addr is typically gendoug
func MakeApplyTx(codePath string, addr, data []byte, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    var tx *monkchain.Transaction
    if codePath != ""{
        tx = NewGenesisContract(codePath)        
    } else{
        tx = monkchain.NewTransactionMessage(addr, monkutil.Big("0"), monkutil.Big("10000"), monkutil.Big("10000"), data)
    }

    tx.Sign(keys.PrivateKey)
    //fmt.Println(tx.String())
    receipt := SimpleTransitionState(addr, block, tx)
    txs := append(block.Transactions(), tx)
    receipts := append(block.Receipts(), receipt)
    block.SetReceipts(receipts, txs)
    
    return tx, receipt
}




