package ethchain

import (
    "fmt"
    "path"
    "os"
    "io/ioutil"
    "encoding/json"
    "github.com/eris-ltd/eth-go-mods/ethutil"    
    "github.com/eris-ltd/eth-go-mods/ethcrypto"    
    "time"
    "strconv"
)

/*
    configure a new genesis block from genesis.json
    deploy the genesis
    something of a shell for epm
*/

type Permission struct{
    Tx int  `json:"tx"`
    Mining int  `json:"mining"`
    Create int  `json:"create"`
}

type Account struct{
    Address string  `json:"addr"`
    Balance string  `json:"balance"`
    Permissions Permission  `json:"permissions"`
}

type GenesisJSON struct{
    Address string  `json:"address"`
    Accounts map[string]Account `json:"accounts"`
    Doug string `json:"doug"`

    f string // other things to do
}

// load the genesis block info from genesis.json
func LoadGenesis() *GenesisJSON{
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

    if SETDOUG != ""{
        g.Doug = SETDOUG
    }
    if GENDOUG != nil{
        g.Address = string(GENDOUG)
    }

    return g
}

// deploy the genesis block
func (g *GenesisJSON) Deploy(block *Block, eth EthManager){
    fmt.Println("###DEPLOYING DOUG", ethutil.Bytes2Hex(GENDOUG), g.Doug)
    time.Sleep(time.Second*2)

    // dummy keys for signing
    keys := ethcrypto.GenerateNewKeyPair() 

    txs := Transactions{}
    receipts := []*Receipt{}

    // create the genesis doug
    tx := NewGenesisContract(path.Join(ContractPath, g.Doug))
    tx.Sign(keys.PrivateKey)
    receipt := SimpleTransitionState(GENDOUG, block, tx)
    txs = append(txs, tx) 
    receipts = append(receipts, receipt)

    chainlogger.Debugln("done genesis. setting perms...")
    txs = Transactions{}
    receipts = []*Receipt{}


    time.Sleep(time.Second*4)
    // set balances and permissions
    for addr, account := range g.Accounts{
        // direct state modification to create accounts and balances
        AddAccount(addr, account.Balance, block)
        // issue txs to set perms
        ts, rs := SetPerms(addr, account, block, keys)
        txs = append(txs, ts...)
        receipts = append(receipts, rs...)
    }
    // update and commit state
    block.SetReceipts(receipts, txs)
    block.State().Update()  
    block.State().Sync()  
}

// set balance of an account (does not commit)
func AddAccount(addr, balance string, block *Block){
    codedAddr := ethutil.Hex2Bytes(addr)
    account := block.State().GetAccount(codedAddr)
    account.Balance = ethutil.Big(balance) //ethutil.BigPow(2, 200)
    block.State().UpdateStateObject(account)
}

// make and apply an administrative tx (simplified vm processing)
// addr is typically gendoug
func MakeApplyTx(codePath string, addr, data []byte, keys *ethcrypto.KeyPair, block *Block) (*Transaction, *Receipt){
    var tx *Transaction
    if codePath != ""{
        tx = NewGenesisContract(codePath)        
    } else{
        fmt.Println("new msg!")
        tx = NewTransactionMessage(addr, ethutil.Big("0"), ethutil.Big("10000"), ethutil.Big("10000"), data)
    }

    tx.Sign(keys.PrivateKey)
    fmt.Println(tx.String())
    receipt := SimpleTransitionState(addr, block, tx)
    
    return tx, receipt
}

// set permissions for an account
// assumes we are talking to gendoug
func SetPerms(addr string, account Account, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt){
    hexAddr := addr 
    if addr[:2] != "0x"{
        hexAddr = "0x"+addr
    }

    txs := Transactions{}
    receipts := []*Receipt{}

    data := PackTxDataArgs("setperm", "tx", hexAddr, "0x"+strconv.Itoa(account.Permissions.Tx))
    fmt.Println("datahex:", ethutil.Bytes2Hex(data))
    tx, rec := MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)

    data = PackTxDataArgs("setperm", "mine", hexAddr, "0x"+strconv.Itoa(account.Permissions.Mining))
    tx, rec = MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)

    data = PackTxDataArgs("setperm", "create", hexAddr, "0x"+strconv.Itoa( account.Permissions.Create))
    tx, rec = MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)
    return txs, receipts
}

// this needs a proper home
func PackTxDataArgs(args ... string) []byte{
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
   return ret
}










/*
    These functions all deploy specific doug styles
    We're generalizing the functionality to genesis.json
    Hope to be rid of them soon ;)
    Though might be useful to keep some around...
    They're probably all broken ...
*/


// for testing. the keys are in keys.txt.
var ADDRS = []string{
        "bbbd0256041f7aed3ce278c56ee61492de96d001",                                 
        "b9398794cafb108622b07d9a01ecbed3857592d5", 
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


// doug and lists of valid miners/txers
func Valids(block *Block, eth EthManager){
    // private keys for these are stored in keys.txt
	for _, addr := range ADDRS{
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
    for _, a := range ADDRS{
        Txers.SetAddr(ethutil.Hex2Bytes(a), 1)
    }
    // add permitted miners to miners contract 
    Miners.SetAddr(ethutil.Hex2Bytes(ADDRS[0]), 1)

    block.State().Update()  
    block.State().Sync()
}

// add addresses and a simple contract
func GenesisTxsByDoug(block *Block, eth EthManager){
    // private keys for these are stored in keys.txt
	for _, addr := range ADDRS{
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

