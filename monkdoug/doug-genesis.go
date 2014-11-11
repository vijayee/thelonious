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
    configure a new genesis block from genesis.json
    deploy the genesis block from json 
    something of a shell for epm
*/

type Account struct{
    Address string `json:"address"`
    ByteAddr []byte  //convenience, but not from json
    Name string `json:"name"`
    Balance string  `json:"balance"`
    Permissions map[string]int `json:"permissions"`
    Stake int `json:"stake"`
}

type GenesisJSON struct{
    // MetaGenDoug
    Address string  `json:"address"`
    DougPath string `json:"doug"`
    Model string `json:"model"`

    // Global GenDoug Singles
    Consensus string `json:"consensus"`
    PublicMine int `json:"public:mine"`
    PublicCreate int `json:"public:create"`
    PublicTx int `json:"public:tx"`
    MaxGasTx string `json:"maxgastx"`
    BlockTime int `json:"blocktime"`

    // Accounts (permissions and stake)
    Accounts []*Account `json:"accounts"`

    f string // other things to do
}

// load the genesis block info from genesis.json
func LoadGenesis() *GenesisJSON{
    _, err := os.Stat(GenesisConfig)
    if err != nil{
        fmt.Println("No genesis.json file found. Resorting to default")
        GenesisConfig = defaultGenesisConfig
    }

    fmt.Println("reading ", GenesisConfig)
    b, err := ioutil.ReadFile(GenesisConfig)
    if err != nil{
        fmt.Println("err reading genesis.json", err)
        os.Exit(0)
    }

    g := new(GenesisJSON)
    err = json.Unmarshal(b, g)
    if err != nil{
        fmt.Println("error unmarshalling genesis.json", err)
        os.Exit(0)
    }

    // move address into accounts, in bytes
    for _, acc := range g.Accounts{
        acc.ByteAddr = monkutil.UserHex2Bytes(acc.Address)
    }
    // if the global DougPath is set, overwrite the config file
    if DougPath != ""{
        g.DougPath = DougPath
    }
    // if the global GENDOUG is set, overwrite the config file
    if GENDOUG != nil{
        g.Address = string(GENDOUG)
    }
    
    if NoGenDoug{
        GENDOUG = nil
        Model = nil
    } else{
        // check doug address validity (addr length is at least 20)
        if len(g.Address) >= 20 {
            if g.Address[:2] == "0x"{
                GENDOUG = monkutil.Hex2Bytes(g.Address[2:])
            } else{
                GENDOUG = []byte(g.Address)
            }
            GENDOUG = GENDOUG[:20]
        }
    }

    // set doug model
    SetDougModel(g.Model)

    return g
}

// deploy the genesis block
// converts the genesisJSON info into a populated and functional doug contract in the genesis block
func (g *GenesisJSON) Deploy(block *monkchain.Block){
    fmt.Println("###DEPLOYING DOUG", g.Address, g.DougPath)

    // dummy keys for signing
    keys := monkcrypto.GenerateNewKeyPair() 

    // create the genesis doug
    codePath := path.Join(ContractPath, g.DougPath)
    genAddr := []byte(g.Address)
    MakeApplyTx(codePath, genAddr, nil, keys, block)

    // set the global vars
    Model.SetValue(genAddr, []string{"setvar", "consensus", g.Consensus}, keys, block)
    Model.SetValue(genAddr, []string{"setvar", "public:mine", "0x"+strconv.Itoa(g.PublicMine)}, keys, block)
    Model.SetValue(genAddr, []string{"setvar", "public:create", "0x"+strconv.Itoa(g.PublicCreate)}, keys, block)
    Model.SetValue(genAddr, []string{"setvar", "public:tx", "0x"+strconv.Itoa(g.PublicTx)}, keys, block)
    Model.SetValue(genAddr, []string{"setvar", "maxgastx", g.MaxGasTx}, keys, block)
    Model.SetValue(genAddr, []string{"setvar", "blocktime", "0x"+strconv.Itoa(g.BlockTime)}, keys, block)

    fmt.Println("done genesis. setting perms...")

    // set balances and permissions
    for _, account := range g.Accounts{
        // direct state modification to create accounts and balances
        AddAccount(account.ByteAddr, account.Balance, block)
        if Model != nil{
            // issue txs to set perms according to the model
            Model.SetPermissions(account.ByteAddr, account.Permissions, block, keys)

            Model.SetValue(GENDOUG, []string{"addminer", account.Name, account.Address, "0x"+strconv.Itoa(account.Stake)}, keys, block)
        }
    }
    block.State().Update()  
    block.State().Sync()  
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




