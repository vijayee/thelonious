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
//  the model should be specified in genesis.json
// the permissions model interface:
type PermModel interface{
    Doug() []byte
    PermLocator(addr []byte, perm string, state *ethstate.State) (*Location, error)
    Permission(addr []byte, perm string, state *ethstate.State) bool
    SetPermissions(addr []byte, account Account, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt)
}

func SetDougModel(model string){
    switch(model){
        case "fake":
            Model = NewFakeModel()
        case "dennis":
            Model = NewGenDougModel()
        default:
            Model = NewFakeModel()
    }
}

// use genesis block and permissions model to validate addr's role
func DougValidate(addr []byte, state *ethstate.State, role string) bool{
    fmt.Println("doug validating!")
    if GENDOUG == nil{
        return true
    }

    if Model == nil{
        return false
    }
    return Model.Permission(addr, role, state)
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

func (m *FakeModel) Doug() []byte{
    return m.doug
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

func (m *FakeModel) Permission(addr []byte, perm string, state *ethstate.State) bool{
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", ethutil.Bytes2Hex(addr), perm, err)
        return false
    }
    obj := state.GetStateObject(loc.addr)
    /*obj.EachStorage(func(k string, v *ethutil.Value){
        fmt.Println(ethutil.Bytes2Hex([]byte(k)), ethutil.Bytes2Hex(v.Bytes()))
    })*/
    val := obj.GetStorage(loc.row)
    return !val.IsNil()
}

func (m *FakeModel) SetPermissions(addr []byte, account Account, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt){
    return nil, nil
}


// the proper genesis doug, ala Dr. McKinnon
type GenDougModel struct{
    doug []byte
}

func NewGenDougModel() PermModel{
    return &GenDougModel{GENDOUG}
}

func (m *GenDougModel) Doug() []byte{
    return m.doug
}

func (m *GenDougModel) PermLocator(addr []byte, perm string, state *ethstate.State) (*Location, error) {
    base := new(big.Int)
    gen := state.GetStateObject(m.doug)
    // get offset (so permission names dont collide with contract names)
    offset := gen.GetStorage(ethutil.Big("7")).BigInt()
    // turn permission to big int
    permBig := ethutil.BigD(ethutil.PackTxDataArgs(perm))
    // location of the locator is perm+offset
    locatorLocator := base.Add(offset, permBig)
    // get locator (specifies position and row)
    locator := gen.GetStorage(locatorLocator).Bytes()

    /*
    fmt.Println("offset", ethutil.Bytes2Hex(offset.Bytes()))
    fmt.Println("permBig", ethutil.Bytes2Hex(permBig.Bytes()))
    fmt.Println("locatorLocator", ethutil.Bytes2Hex(locatorLocator.Bytes()))
    fmt.Println("locator", ethutil.Bytes2Hex(locator))

    gen.EachStorage(func(k string, v *ethutil.Value){
        fmt.Println(ethutil.Bytes2Hex([]byte(k)), ethutil.Bytes2Hex(v.Bytes()))
    })*/
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
    permStrLocator := base.Add(base.Mul(addrBig, ethutil.Big("256")), row)

    return &Location{m.doug, permStrLocator, pos}, nil

}

func (m *GenDougModel) Permission(addr []byte, perm string, state *ethstate.State)bool{
    base := new(big.Int)
    // get location object
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("err on perm locator", ethutil.Bytes2Hex(addr), perm, err)
        return false
    }
    obj := state.GetStateObject(loc.addr)

    // recover permission string
    permstr := obj.GetStorage(loc.row)
    
    // recover permission from permission string (ie get nibble)
    permbit := base.Div(permstr.BigInt(), base.Exp(ethutil.Big("2"), loc.pos, nil))
    permBig := base.Mod(permbit, ethutil.Big("16"))
    return permBig.Int64() > 0
}


func (m *GenDougModel) SetPermissions(addr []byte, account Account, block *Block, keys *ethcrypto.KeyPair) (Transactions, []*Receipt){

    txs := Transactions{}
    receipts := []*Receipt{}

    data := ethutil.PackTxDataArgs("setperm", "tx", "0x"+ethutil.Bytes2Hex(addr), "0x"+strconv.Itoa(account.Permissions.Tx))
    tx, rec := MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)

    data = ethutil.PackTxDataArgs("setperm", "mine", "0x"+ethutil.Bytes2Hex(addr), "0x"+strconv.Itoa(account.Permissions.Mining))
    tx, rec = MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)

    data = ethutil.PackTxDataArgs("setperm", "create", "0x"+ethutil.Bytes2Hex(addr), "0x"+strconv.Itoa( account.Permissions.Create))
    tx, rec = MakeApplyTx("", GENDOUG, data, keys, block)
    txs = append(txs, tx)
    receipts = append(receipts, rec)
    return txs, receipts
}
