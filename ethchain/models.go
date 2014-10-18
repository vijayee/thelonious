package ethchain

import (
    "math/big"
    "errors"
    "fmt"
    "strconv"
    "github.com/eris-ltd/eth-go-mods/ethstate"
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethcrypto"
)



var (
    GENDOUG []byte = nil // dougs address
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
    Doug(state *ethstate.State) *ethstate.StateObject
    // return the location of a permission string for an address
    PermLocator(addr []byte, perm string, state *ethstate.State) (*Location, error)
    // Get a permission string
    GetPermission(addr []byte, perm string, state *ethstate.State) *ethutil.Value
    // Determine if a user has permission to do something
    HasPermission(addr []byte, perm string, state *ethstate.State) bool 
    // Set some permissions for a given address. requires valid keypair
    SetPermissions(addr []byte, permissions map[string]int, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt)

    // doug has a key-value store that is space partitioned for collision avoidance
    // resolve those values
    GetValue(key, namespace string, state *ethstate.State) []byte
}


// the easy fake model
type FakeModel struct{
    doug []byte
    txers string
    miners string
    create string
}

func NewFakeModel() PermModel{
    return &FakeModel{GENDOUG, "01", "02", "03"}
}

func (m *FakeModel) Doug(state *ethstate.State) *ethstate.StateObject{
    return state.GetStateObject(m.doug)
}

func (m *FakeModel) PermLocator(addr []byte, perm string, state *ethstate.State) (*Location, error){
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
    loc.addr = genDoug.GetStorage(ethutil.BigD(ethutil.Hex2Bytes(N))).Bytes()
    addrBig := ethutil.BigD(ethutil.LeftPadBytes(addr, 32))
    loc.row = addrBig

    return loc, nil
}

func (m *FakeModel) GetPermission(addr []byte, perm string, state *ethstate.State) *ethutil.Value{
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", ethutil.Bytes2Hex(addr), perm, err)
        return ethutil.NewValue(nil)
    }
    obj := state.GetStateObject(loc.addr)
    /*obj.EachStorage(func(k string, v *ethutil.Value){
        fmt.Println(ethutil.Bytes2Hex([]byte(k)), ethutil.Bytes2Hex(v.Bytes()))
    })*/
    val := obj.GetStorage(loc.row)
    return val
}

func (m *FakeModel) HasPermission(addr []byte, perm string, state *ethstate.State) bool{
    val := m.GetPermission(addr, perm, state)
    return !val.IsNil()
}

func (m *FakeModel) SetPermissions(addr []byte, permissions map[string]int, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt){
    return nil, nil
}

func (m *FakeModel) GetValue(key, namespace string, state *ethstate.State) []byte{
    return nil
}

// the proper genesis doug, ala Dr. McKinnon
type GenDougModel struct{
    doug []byte
    base *big.Int
}

func NewGenDougModel() PermModel{
    return &GenDougModel{GENDOUG, new(big.Int)}
}

func (m *GenDougModel) Doug(state *ethstate.State) *ethstate.StateObject{
    return state.GetOrNewStateObject(m.doug) // add or new so we can avoid panics..
}


func (m *GenDougModel) PermLocator(addr []byte, perm string, state *ethstate.State) (*Location, error) {
    // location of the locator is perm+offset
    locator := m.GetValue(perm, "perms", state) //m.resolvePerm(perm, state) 
    //PrintHelp(map[string]interface{}{"loc":locator}, m.Doug(state))

    if len(locator) == 0{
        return nil, errors.New("could not find locator")
    }
    pos := ethutil.BigD(locator[len(locator)-1:]) // first byte
    row := ethutil.Big("0")
    if len(locator) > 1{
        row = ethutil.BigD(locator[len(locator)-2:len(locator)-1])// second byte
    }
    // return permission string location
    addrBig := ethutil.BigD(ethutil.LeftPadBytes(addr, 32))
    permStrLocator := m.base.Add(m.base.Mul(addrBig, ethutil.Big("256")), row)

    return &Location{m.doug, permStrLocator, pos}, nil

}

func (m *GenDougModel) GetPermission(addr []byte, perm string, state *ethstate.State) *ethutil.Value{
    // get location object
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", ethutil.Bytes2Hex(addr), perm, err)
        return ethutil.NewValue(nil)
    }
    obj := state.GetStateObject(loc.addr)

    // recover permission string
    permstr := obj.GetStorage(loc.row)
    
    // recover permission from permission string (ie get nibble)
    permbit := m.base.Div(permstr.BigInt(), m.base.Exp(ethutil.Big("2"), loc.pos, nil))
    permBig := m.base.Mod(permbit, ethutil.Big("16"))
    return ethutil.NewValue(permBig)
}

// determines if addr has sufficient permissions to execute perm
func (m *GenDougModel) HasPermission(addr []byte, perm string, state *ethstate.State)bool{
    permBig := m.GetPermission(addr, perm, state).BigInt()
    return permBig.Int64() > 0
}

// set some permissions on an addr
// requires keys with sufficient privileges
func (m *GenDougModel) SetPermissions(addr []byte, permissions map[string]int, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt){

    txs := Transactions{}
    receipts := []*Receipt{}

    for perm, val := range permissions{
        data := ethutil.PackTxDataArgs("setperm", perm, "0x"+ethutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
        //fmt.Println("data for ", perm, ethutil.Bytes2Hex(data))
        tx, rec := MakeApplyTx("", GENDOUG, data, keys, block)
        txs = append(txs, tx)
        receipts = append(receipts, rec)
    }
    //fmt.Println(permissions)
    //os.Exit(0)
    return txs, receipts
}

func (m *GenDougModel) GetValue(key, namespace string, state *ethstate.State) []byte{
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
    //fmt.Println("loc after resolution for key in namespace:", key, namespace, ethutil.Bytes2Hex(loc.Bytes()))
    val := m.Doug(state).GetStorage(loc)
    //fmt.Println("corresponding value:", ethutil.Bytes2Hex(val.Bytes()))
    return val.Bytes()
}

// resolve addresses for keys based on namespace partition
// does not return the values, just their proper addresses!
// offset used to partition namespaces
// these don't need to take state if the offset is fixed
//      it is fixed, but maybe one day it wont be?

// resolve location of an address 
func (m *GenDougModel) resolveAddr(key string, state *ethstate.State) *big.Int{
    // addrs have no special offset
    return String2Big(key)

}

// resolve location of  a permission locator
func (m *GenDougModel) resolvePerm(key string, state *ethstate.State) *big.Int{
    // permissions have one offset
    offset := ethutil.BigD(m.GetValue("offset", "special", state) )
    // turn permission to big int
    permBig := String2Big(key) 
    // location of the permission locator is perm+offset
    //PrintHelp(map[string]interface{}{"offset":offset, "permbig":permBig, "sum":m.base.Add(offset, permBig)}, m.Doug(state))
    return m.base.Add(offset, permBig)
}

// resolve location of a named value
func (m *GenDougModel) resolveVal(key string, state *ethstate.State) *big.Int{
    // values have two offsets
    offset := ethutil.BigD(m.GetValue("offset", "special", state) )
    // turn key to big int
    valBig := String2Big(key) 
    // location of this value is (+ key (* 2 offset))
    return m.base.Add(m.base.Mul(offset, big.NewInt(2)), valBig)
}

// resolve position of special values
func (m *GenDougModel) resolveSpecial(key string, state *ethstate.State) *big.Int{
    switch(key){
        case "offset":
            return big.NewInt(7)
    }
    return nil
}
            //offset := m.Doug().GetStorage(ethutil.Big("7")).BigInt()
            //return 

func String2Big(s string) *big.Int{
    // right pad the string, convert to big num
    return ethutil.BigD(ethutil.PackTxDataArgs(s))
}







// pretty print chain queries and storage
func PrintHelp(m map[string]interface{}, obj *ethstate.StateObject){
    for k, v := range m{
        if vv, ok := v.(*ethutil.Value); ok{
            fmt.Println(k, ethutil.Bytes2Hex(vv.Bytes()))
        } else if vv, ok := v.(*big.Int); ok{
            fmt.Println(k, ethutil.Bytes2Hex(vv.Bytes()))
        } else if vv, ok := v.([]byte); ok{
            fmt.Println(k, ethutil.Bytes2Hex(vv))
        }
    }
    obj.EachStorage(func(k string, v *ethutil.Value){
        fmt.Println(ethutil.Bytes2Hex([]byte(k)), ethutil.Bytes2Hex(v.Bytes()))
    })
}


