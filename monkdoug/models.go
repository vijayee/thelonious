package monkdoug

import (
    "math/big"
    "bytes"
    "fmt"
    "strconv"
    "github.com/eris-ltd/thelonious/monkstate"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkcrypto"
    "github.com/eris-ltd/thelonious/monkchain"
    vars "github.com/eris-ltd/eris-std-lib/go-tests"
)

// location struct (where is a permission?)
// the model must specify how to extract the permission from the location
type Location struct{
    addr []byte // contract addr
    row *big.Int // storage location
    pos *big.Int // nibble/bit/byte indicator
}

/*
    Permission models are used for setting up the genesis block
    and for validating blocks and transactions.
    They allow for arbitrary extensions of consensus
*/
type PermModel interface{
    // Set some permissions for a given address. requires valid keypair
    SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt)
    SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt)

    // generic validation functions for arbitrary consensus models
    // satisfies monkchain.GenDougModel
    ValidatePerm(addr []byte, perm string, state *monkstate.State) error
    ValidateBlock(block *monkchain.Block) error
    ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error
}

/*
    The yes model grants all permissions 
*/
type YesModel struct{
}

func NewYesModel() PermModel{
    return &YesModel{}
}

func (m *YesModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return nil
}

func (m *YesModel) ValidateBlock(block *monkchain.Block) error{
    return nil
}

func (m *YesModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    return nil
}

/*
    The no model grants no permissions
*/
type NoModel struct{
}

func NewNoModel() PermModel{
    return &NoModel{}
}

func (m *NoModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateBlock(block *monkchain.Block) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    return fmt.Errorf("No!")
}

/*
    The stdlib model grants permissions based on the state of the gendoug
    It depends on the eris-std-lib for its storage model
*/
type StdLibModel struct{
    doug []byte
}

func NewStdLibModel(g *GenesisConfig) PermModel{
    return &StdLibModel{g.ByteAddr}
}

func (m *StdLibModel) Doug(state *monkstate.State) *monkstate.StateObject{
    return state.GetOrNewStateObject(m.doug)
}

func (m *StdLibModel) PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error){
    // where can we find the perm w.r.t the address?
    locator := vars.GetLinkedListElement(m.doug, "permnames", perm, state)
    locatorBig := monkutil.BigD(locator)

    return &Location{m.doug, locatorBig, nil}, nil
}

func (m *StdLibModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value{
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        // suck a dick
    }
    
    locInt := loc.row.Uint64()
    
    permStr := vars.GetKeyedArrayElement(m.doug, "perms", monkutil.Bytes2Hex(addr), int(locInt), state)
    return monkutil.NewValue(permStr)
}

func (m *StdLibModel) HasPermission(addr []byte, perm string, state *monkstate.State) bool{
    permBig := m.GetPermission(addr, perm, state).BigInt()
    return permBig.Int64() > 0
}

func (m *StdLibModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){

    txs := monkchain.Transactions{}
    receipts := []*monkchain.Receipt{}

    for perm, val := range permissions{
        data := monkutil.PackTxDataArgs2("setperm", perm, "0x"+monkutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
        //fmt.Println("data for ", perm, monkutil.Bytes2Hex(data))
        tx, rec := MakeApplyTx("", m.doug, data, keys, block)
        txs = append(txs, tx)
        receipts = append(receipts, rec)
    }
    //fmt.Println(permissions)
    //os.Exit(0)
    return txs, receipts
}

func (m *StdLibModel) SetValue(addr []byte, args []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    data := monkutil.PackTxDataArgs2(args...)
    tx, rec := MakeApplyTx("", addr, data, keys, block)
    return tx, rec
}

func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    if m.HasPermission(addr, role, state){
        return nil
    }
    return monkchain.InvalidPermError(addr, role)
}

func (m *StdLibModel) ValidateBlock(block *monkchain.Block) error{
    // we have to verify using the state of the previous block!
    prevBlock := monkchain.GetBlock(block.PrevHash)

    // check that miner has permission to mine
    if !m.HasPermission(block.Coinbase, "mine", prevBlock.State()){
        return monkchain.InvalidPermError(block.Coinbase, "mine")
    }
    // check that signature of block matches miners coinbase
    if !bytes.Equal(block.Signer(), block.Coinbase){
        //return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
    }
    // check that its the miners turn in the round robin
    if !bytes.Equal(prevBlock.PrevHash, monkchain.ZeroHash256){
        // if its not the genesis block, get coinbase of last block
        // find next entry in linked list
        prevCoinbase := prevBlock.Coinbase
        nextCoinbase, _ := vars.GetNextLinkedListElement(m.doug, "seq:name", string(prevCoinbase), prevBlock.State())
        if !bytes.Equal(nextCoinbase, block.Coinbase){
            return monkchain.InvalidTurnError(block.Coinbase, nextCoinbase)        
        }
    } else{
        // is it is genesis block, find first entry in linked list
        nextCoinbase, _ := vars.GetLinkedListHead(m.doug, "seq:name", prevBlock.State())
        if !bytes.Equal(nextCoinbase, block.Coinbase){
            return monkchain.InvalidTurnError(block.Coinbase, nextCoinbase)        
        }
    }

    return nil
}

func (m *StdLibModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    // check that sender has permission to transact or create
    var perm string
    if tx.IsContract(){
        perm = "create"
    } else{
        perm = "transact"
    }
    if !m.HasPermission(tx.Sender(), perm, block.State()){
        return monkchain.InvalidPermError(tx.Sender(), perm)
    }
    // check that tx uses less than maxgas
    gas := tx.GasValue()
    max := vars.GetSingle(m.doug, "maxgas", block.State()) 
    maxBig := monkutil.BigD(max)
    if max != nil && gas.Cmp(maxBig) > 0 {
        return monkchain.GasLimitTxError(gas, maxBig)
    }
    
    return nil
}
