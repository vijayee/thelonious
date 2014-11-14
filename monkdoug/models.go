package monkdoug

import (
    "math/big"
    //"errors"
    "bytes"
    "fmt"
    "strconv"
    "github.com/eris-ltd/thelonious/monkstate"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkcrypto"
    "github.com/eris-ltd/eris-std-lib/go-tests"
    "github.com/eris-ltd/thelonious/monkchain"
)



var (
    Model PermModel = nil // permissions model
)

// location struct (where is a permission?)
// the model must specify how to extract the permission from the location
type Location struct{
    addr []byte // contract addr
    row *big.Int // storage location
    pos *big.Int // nibble/bit/byte indicator
}


// doug validation requires a reference model to understand 
//  where the permissions are with respect to doug, the target addr, and the permission name
//  the model name should be specified in genesis.json
// now, the permissions model interface:
type PermModel interface{
    // return the current doug state
    Doug(state *monkstate.State) *monkstate.StateObject
    // return the location of a permission string for an address
//    PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error)
    // Get a permission string
//    GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value
    // Determine if a user has permission to do something
//    HasPermission(addr []byte, perm string, state *monkstate.State) bool 
    ValidatePerm(addr []byte, perm string, state *monkstate.State) error
    // Set some permissions for a given address. requires valid keypair
    SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt)

    SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt)
    // doug has a key-value store that is space partitioned for collision avoidance
    // resolve those values
//    GetValue(key, namespace string, state *monkstate.State) []byte

    // generic validation functions for arbitrary consensus models
    // hot shit yo!
    ValidateBlock(block *monkchain.Block) error
    ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error
}

type YesModel struct{
}

func NewYesModel() PermModel{
    return &YesModel{}
}

func (m *YesModel) Doug(state *monkstate.State) *monkstate.StateObject{
    return nil    
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return nil
}

func (m *YesModel) ValidateValue(name string, value interface{}, state *monkstate.State) bool{
    return true
}

func (m *YesModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) ValidateBlock(block *monkchain.Block) error{
    return nil
}

func (m *YesModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    return nil
}

type NoModel struct{
}

func NewNoModel() PermModel{
    return &NoModel{}
}

func (m *NoModel) Doug(state *monkstate.State) *monkstate.StateObject{
    return nil    
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateValue(name string, value interface{}, state *monkstate.State) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) ValidateBlock(block *monkchain.Block) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    return fmt.Errorf("No!")
}


// the LLL eris-std-lib model with types :)
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

func (m *StdLibModel) GetValue(key, namespace string, state *monkstate.State) []byte{
    switch(namespace){
        case "global":
           return vars.GetSingle(m.doug, key, state)
        default:
            return nil
    }
    return nil
}
    
func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    if m.HasPermission(addr, role, state){
        return nil
    }
    return InvalidPermError(addr, role)
}

// TODO: fix..
func (m *StdLibModel) ValidateValue(name string, value interface{}, state *monkstate.State) error{
    return nil
}

func (m *StdLibModel) ValidateBlock(block *monkchain.Block) error{
    // check that miner has permission to mine
    if !m.HasPermission(block.Coinbase, "mine", block.State()){
        return InvalidPermError(block.Coinbase, "mine")
    }
    // check that signature of block matches miners coinbase
    if !bytes.Equal(block.Signer(), block.Coinbase){
        return InvalidSigError(block.Signer(), block.Coinbase)
    }
    // check that its the miners turn in the round robin
    // TODO:

    return nil
}

func (m *StdLibModel) ValidateTx(tx *monkchain.Transaction, block *monkchain.Block) error{
    // check that sender has permission to transact or TODO: create
    if !m.HasPermission(tx.Sender(), "transact", block.State()){
        return InvalidPermError(tx.Sender(), "transact")
    }
    // check that max gas has not been exceeded
    //if !genDoug.ValidateValue("maxgas", self.tx.GasValue(), self.block.State()){
     //       return GasLimitTxError(gas, maxBig)
    
    return nil
}
