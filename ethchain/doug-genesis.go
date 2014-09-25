package ethchain

import (
    "fmt"
    "path"
    "os"
    "io/ioutil"
    "github.com/eris-ltd/eth-go-mods/ethutil"    
    "github.com/eris-ltd/eth-go-mods/ethcrypto"    
)

/*
    functions for setting the genesis block
*/

// for testing. the keys are in keys.txt.
var ADDRS = []string{
        "bbbd0256041f7aed3ce278c56ee61492de96d001",                                  
        "b9398794cafb108622b07d9a01ecbed3857592d5", 
    }

type Permission{
    Tx int  `json:"tx"`
    Mining int  `json:"mining"`
    Create int  `json:"create"`
}

type Account struct{
    Address string  `json:"addr"`
    Balance string  `json:"balance"`
    Permissions Permission  `json:"permissions"`
}

type Genesis struct{
    Address string  `json:"address"`
    Accounts map[string]Account `json:"accounts"`
    Doug string `json:"doug"`

    f string // other things to do

    block *Block
    eth EthManager
}

// load the genesis block info from genesis.json
func LoadGenesis() *Genesis{
    b, err := ioutil.ReadFile(GenesisConfig)
    if err != nil{
        fmt.Println("err reading genesis.json", err)
        os.Exit(0)
    }

    g := new(Genesis)
    err := json.Unmarshall(b, g)
    if err != nil{
        fmt.Println("error unmarshalling genesis.json", err)
        os.Exit(0)
    }

    return g
}

// deploy the genesis block
func (g *Genesis) Deploy(block *Block, eth EthManager, f string){
    for addr, account := range g.Accounts{
        AddAccount(addr, account.Balance, block)
    }

    txs := Transactions{}
    receipts := []*Receipt{}

    GENDOUG = g.Address 
    tx := NewGenesisContract(path.Join(ContractPath, g.Doug))
    fmt.Println(tx.String())
    receipt := SimpleTransitionState(GENDOUG, block, tx)

    txs = append(txs, tx) 
    receipts = append(receipts, receipt)

    block.SetReceipts(receipts, txs)
    block.State().Update()  
    block.State().Sync()  
}


// add addresses
func GenesisSimple(block *Block, eth EthManager){
    // private keys for these are stored in keys.txt
	for _, addr := range ADDRS{
        AddAccount(addr, "1606938044258990275541962092341162602522202993782792835301376", block)
	}
    block.State().Update()  
    block.State().Sync()  
}

func GenesisInit(block *Block, eth EthManager){
	for _, addr := range ADDRS{
        AddAccount(addr, "1606938044258990275541962092341162602522202993782792835301376", block)
	}
    txs := Transactions{}
    receipts := []*Receipt{}

    addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
    GENDOUG = addr[12:] 
    tx := NewGenesisContract(path.Join(ContractPath, "lll/keyvalue.lll"))
    receipt := SimpleTransitionState(addr, block, tx)

    txs = append(txs, tx) 
    receipts = append(receipts, receipt)

    block.SetReceipts(receipts, txs)
    block.State().Update()  
    block.State().Sync()  
}


// add addresses and a simple contract
func GenesisKeyVal(block *Block, eth EthManager){
    // private keys for these are stored in keys.txt
	for _, addr := range ADDRS{
        AddAccount(addr, "1606938044258990275541962092341162602522202993782792835301376", block)
	}
    txs := Transactions{}
    receipts := []*Receipt{}

    addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
    GENDOUG = addr[12:] 
    tx := NewGenesisContract(path.Join(ContractPath, "lll/keyvalue.lll"))
    receipt := SimpleTransitionState(addr, block, tx)

    txs = append(txs, tx) 
    receipts = append(receipts, receipt)

    block.SetReceipts(receipts, txs)
    block.State().Update()  
    block.State().Sync()  
}


func AddAccount(addr, balance string, block *Block){
    codedAddr := ethutil.Hex2Bytes(addr)
    account := block.State().GetAccount(codedAddr)
    account.Balance = ethutil.Big(balance) //ethutil.BigPow(2, 200)
    block.State().UpdateStateObject(account)
}

// doug and lists of valid miners/txers
func Valids(block *Block, eth EthManager){
    addrs := []string{
        "bbbd0256041f7aed3ce278c56ee61492de96d001",
        "b9398794cafb108622b07d9a01ecbed3857592d5",
    }
    // private keys for these are stored in keys.txt
	for _, addr := range addrs{
        AddAccount(addr, "1606938044258990275541962092341162602522202993782792835301376", block)
	}
  
    // set up main contract addrs
    doug := ethcrypto.Sha3Bin([]byte("the genesis doug"))[12:]
    GENDOUG = doug 
    txers := ethcrypto.Sha3Bin([]byte("txers"))[12:]
    miners := ethcrypto.Sha3Bin([]byte("miners"))[12:]
    // create accounts
    Doug := block.State().GetOrNewStateObject(doug)
    Txers := block.State().GetOrNewStateObject(txers)
    Miners := block.State().GetOrNewStateObject(miners)
    // add addresses into DOUG
    Doug.SetAddr([]byte("\x00"), doug)
    Doug.SetAddr([]byte("\x01"), txers)
    Doug.SetAddr([]byte("\x02"), miners)
    // add permitted transactors to txers contract 
    for _, a := range addrs{
        Txers.SetAddr(ethutil.Hex2Bytes(a), 1)
    }
    // add permitted miners to miners contract 
    Miners.SetAddr(ethutil.Hex2Bytes(addrs[0]), 1)

    block.State().Update()  
    block.State().Sync()
}

// add addresses and a simple contract
func GenesisTxsByDoug(block *Block, eth EthManager){
    // private keys for these are stored in keys.txt
	for _, addr := range []string{
        "bbbd0256041f7aed3ce278c56ee61492de96d001",
        "b9398794cafb108622b07d9a01ecbed3857592d5",
	} {
        AddAccount(addr, "1606938044258990275541962092341162602522202993782792835301376", block)
	}

    fmt.Println("TXS BY DOUG!!")

    txs := Transactions{}
    receipts := []*Receipt{}

    addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
    GENDOUG = addr[12:] //[]byte("\x00"*16 + "DOUG")
    tx := NewGenesisContract(path.Join(ContractPath, "lll/fake-doug1.lll"))
    fmt.Println(tx.String())
    receipt := SimpleTransitionState(addr, block, tx)

    txs = append(txs, tx) 
    receipts = append(receipts, receipt)

    block.SetReceipts(receipts, txs)
    block.State().Update()  
    block.State().Sync()  
}

