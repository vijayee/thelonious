package monkdoug

import (
    "math/big"
    "errors"
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
    PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error)
    // Get a permission string
    GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value
    // Determine if a user has permission to do something
    HasPermission(addr []byte, perm string, state *monkstate.State) bool 
    // Set some permissions for a given address. requires valid keypair
    SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt)

    SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt)
    // doug has a key-value store that is space partitioned for collision avoidance
    // resolve those values
    GetValue(key, namespace string, state *monkstate.State) []byte
}

type YesModel struct{
}

func NewYesModel() monkchain.GenDougModel{
    return &YesModel{}
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    return true
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

type NoModel struct{
}

func NewNoModel() monkchain.GenDougModel{
    return &NoModel{}
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    return false
}

func (m *NoModel) ValidateValue(name string, value interface{}, state *monkstate.State) bool{
    return false
}

func (m *NoModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

// the easy fake model
type FakeModel struct{
    doug []byte
    txers string
    miners string
    create string
}

func NewFakeModel(gendoug []byte) monkchain.GenDougModel{
    return &FakeModel{gendoug, "01", "02", "03"}
}

func (m *FakeModel) Doug(state *monkstate.State) *monkstate.StateObject{
    return state.GetStateObject(m.doug)
}

func (m *FakeModel) PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error){
    loc := new(Location)

    var N string
    switch(perm){
        case "tx":
            N = m.txers
        case "mine":
            N = m.miners
        case "create":
            N = m.create
        default:
            return nil, errors.New("Invalid permission name")
    }
    genDoug := state.GetStateObject(m.doug)
    loc.addr = genDoug.GetStorage(monkutil.BigD(monkutil.Hex2Bytes(N))).Bytes()
    addrBig := monkutil.BigD(monkutil.LeftPadBytes(addr, 32))
    loc.row = addrBig

    return loc, nil
}

func (m *FakeModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value{
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", monkutil.Bytes2Hex(addr), perm, err)
        return monkutil.NewValue(nil)
    }
    obj := state.GetStateObject(loc.addr)
    /*obj.EachStorage(func(k string, v *monkutil.Value){
        fmt.Println(monkutil.Bytes2Hex([]byte(k)), monkutil.Bytes2Hex(v.Bytes()))
    })*/
    val := obj.GetStorage(loc.row)
    return val
}

func (m *FakeModel) HasPermission(addr []byte, perm string, state *monkstate.State) bool{
    val := m.GetPermission(addr, perm, state)
    return !val.IsNil()
}

func (m *FakeModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}


func (m *FakeModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *FakeModel) GetValue(key, namespace string, state *monkstate.State) []byte{
    return nil
}

func (m *FakeModel) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    return m.HasPermission(addr, role, state)        
}

func (m *FakeModel) ValidateValue(name string, value interface {}, state *monkstate.State) bool{
    return true 
}

// the proper genesis doug, ala Dr. McKinnon
type GenDougModel struct{
    doug []byte
    base *big.Int
}

func NewGenDougModel(gendoug []byte) monkchain.GenDougModel{
    return &GenDougModel{gendoug, new(big.Int)}
}

func (m *GenDougModel) Doug(state *monkstate.State) *monkstate.StateObject{
    return state.GetOrNewStateObject(m.doug) // add or new so we can avoid panics..
}


func (m *GenDougModel) PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error) {
    // location of the locator is perm+offset
    locator := m.GetValue(perm, "perms", state) //m.resolvePerm(perm, state) 
    //PrintHelp(map[string]interface{}{"loc":locator}, m.Doug(state))

    if len(locator) == 0{
        return nil, errors.New("could not find locator")
    }
    pos := monkutil.BigD(locator[len(locator)-1:]) // first byte
    row := monkutil.Big("0")
    if len(locator) > 1{
        row = monkutil.BigD(locator[len(locator)-2:len(locator)-1])// second byte
    }
    // return permission string location
    addrBig := monkutil.BigD(monkutil.LeftPadBytes(addr, 32))
    permStrLocator := m.base.Add(m.base.Mul(addrBig, monkutil.Big("256")), row)

    return &Location{m.doug, permStrLocator, pos}, nil

}

func (m *GenDougModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value{
    // get location object
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", monkutil.Bytes2Hex(addr), perm, err)
        return monkutil.NewValue(nil)
    }
    obj := state.GetStateObject(loc.addr)

    // recover permission string
    permstr := obj.GetStorage(loc.row)
    
    // recover permission from permission string (ie get nibble)
    permbit := m.base.Div(permstr.BigInt(), m.base.Exp(monkutil.Big("2"), loc.pos, nil))
    permBig := m.base.Mod(permbit, monkutil.Big("16"))
    return monkutil.NewValue(permBig)
}

// determines if addr has sufficient permissions to execute perm
func (m *GenDougModel) HasPermission(addr []byte, perm string, state *monkstate.State)bool{
    permBig := m.GetPermission(addr, perm, state).BigInt()
    return permBig.Int64() > 0
}

// set some permissions on an addr
// requires keys with sufficient privileges
func (m *GenDougModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){

    txs := monkchain.Transactions{}
    receipts := []*monkchain.Receipt{}

    for perm, val := range permissions{
        data := monkutil.PackTxDataArgs("setperm", perm, "0x"+monkutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
        //fmt.Println("data for ", perm, monkutil.Bytes2Hex(data))
        tx, rec := MakeApplyTx("", m.doug, data, keys, block)
        txs = append(txs, tx)
        receipts = append(receipts, rec)
    }
    //fmt.Println(permissions)
    //os.Exit(0)
    return txs, receipts
}

func (m *GenDougModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *GenDougModel) GetValue(key, namespace string, state *monkstate.State) []byte{
    var loc *big.Int
    //fmt.Println("get value:", key, namespace)
    switch(namespace){
        case "addrs":
            loc = m.resolveAddr(key, state)
        case "perms":
            loc = m.resolvePerm(key, state)
        case "values":
            loc = m.resolveVal(key, state)    
        case "special":
            loc = m.resolveSpecial(key, state)
        default:
            return nil
    }
    //fmt.Println("loc after resolution for key in namespace:", key, namespace, monkutil.Bytes2Hex(loc.Bytes()))
    val := m.Doug(state).GetStorage(loc)
    //fmt.Println("corresponding value:", monkutil.Bytes2Hex(val.Bytes()))
    return val.Bytes()
}

// resolve addresses for keys based on namespace partition
// does not return the values, just their proper addresses!
// offset used to partition namespaces
// these don't need to take state if the offset is fixed
//      it is fixed, but maybe one day it wont be?

// resolve location of an address 
func (m *GenDougModel) resolveAddr(key string, state *monkstate.State) *big.Int{
    // addrs have no special offset
    return String2Big(key)

}

// resolve location of  a permission locator
func (m *GenDougModel) resolvePerm(key string, state *monkstate.State) *big.Int{
    // permissions have one offset
    offset := monkutil.BigD(m.GetValue("offset", "special", state) )
    // turn permission to big int
    permBig := String2Big(key) 
    // location of the permission locator is perm+offset
    //PrintHelp(map[string]interface{}{"offset":offset, "permbig":permBig, "sum":m.base.Add(offset, permBig)}, m.Doug(state))
    return m.base.Add(offset, permBig)
}

// resolve location of a named value
func (m *GenDougModel) resolveVal(key string, state *monkstate.State) *big.Int{
    // values have two offsets
    offset := monkutil.BigD(m.GetValue("offset", "special", state) )
    // turn key to big int
    valBig := String2Big(key) 
    // location of this value is (+ key (* 2 offset))
    return m.base.Add(m.base.Mul(offset, big.NewInt(2)), valBig)
}

// resolve position of special values
func (m *GenDougModel) resolveSpecial(key string, state *monkstate.State) *big.Int{
    switch(key){
        case "offset":
            return big.NewInt(7)
    }
    return nil
}


func (m *GenDougModel) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    return m.HasPermission(addr, role, state)
}

func (m *GenDougModel) ValidateValue(name string, value interface{}, state *monkstate.State) bool{
    return true
}

// the LLL eris-std-lib model with types :)
type StdLibModel struct{
    doug []byte
    base *big.Int
}

func NewStdLibModel(gendoug []byte) monkchain.GenDougModel{
    return &StdLibModel{gendoug, new(big.Int)}
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

    
func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.State) bool{
    return m.HasPermission(addr, role, state)
}

func (m *StdLibModel) ValidateValue(name string, value interface{}, state *monkstate.State) bool{
    return true
}
    
func String2Big(s string) *big.Int{
    // right pad the string, convert to big num
    return monkutil.BigD(monkutil.PackTxDataArgs(s))
}

// pretty print chain queries and storage
func PrintHelp(m map[string]interface{}, obj *monkstate.StateObject){
    for k, v := range m{
        if vv, ok := v.(*monkutil.Value); ok{
            fmt.Println(k, monkutil.Bytes2Hex(vv.Bytes()))
        } else if vv, ok := v.(*big.Int); ok{
            fmt.Println(k, monkutil.Bytes2Hex(vv.Bytes()))
        } else if vv, ok := v.([]byte); ok{
            fmt.Println(k, monkutil.Bytes2Hex(vv))
        }
    }
    obj.EachStorage(func(k string, v *monkutil.Value){
        fmt.Println(monkutil.Bytes2Hex([]byte(k)), monkutil.Bytes2Hex(v.Bytes()))
    })
}

