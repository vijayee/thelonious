package monkdoug

import (
    "math/big"
    "bytes"
    "fmt"
    "strconv"
    "strings"
    //"log"
    "github.com/eris-ltd/thelonious/monkstate"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkcrypto"
    "github.com/eris-ltd/thelonious/monkchain"
    vars "github.com/eris-ltd/eris-std-lib/go-tests"
)

// location struct (where is a permission?)
// the model must specify how to extract the permission from the location
// TODO: deprecate
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
    // Set some permissions and values in gendoug. requires valid keypair
    SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt)
    SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt)

    // generic validation functions for arbitrary consensus models
    // satisfies monkchain.GenDougModel
    Difficulty(block, parent *monkchain.Block) *big.Int
    ValidatePerm(addr []byte, perm string, state *monkstate.State) error
    ValidateBlock(block *monkchain.Block) error
    ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error
}

/*
    The yes model grants all permissions 
*/
type YesModel struct{
    g *GenesisConfig
}

func NewYesModel(g *GenesisConfig) PermModel{
    return &YesModel{g}
}

func (m *YesModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *YesModel) Difficulty(block, parent *monkchain.Block) *big.Int{
    return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return nil
}

func (m *YesModel) ValidateBlock(block *monkchain.Block) error{
    return nil
}

func (m *YesModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error{
    return nil
}

/*
    The no model grants no permissions
*/
type NoModel struct{
    g *GenesisConfig
}

func NewNoModel(g *GenesisConfig) PermModel{
    return &NoModel{g}
}

func (m *NoModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *NoModel) Difficulty(block, parent *monkchain.Block) *big.Int{
    return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateBlock(block *monkchain.Block) error{
    return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error{
    return fmt.Errorf("No!")
}

/*
    The stdlib model grants permissions based on the state of the gendoug
    It depends on the eris-std-lib for its storage model
*/
type StdLibModel struct{
    base *big.Int
    doug []byte
    g *GenesisConfig
    pow monkchain.PoW
}

func NewStdLibModel(g *GenesisConfig) PermModel{
    return &StdLibModel{
        base:   new(big.Int),
        doug:   g.ByteAddr, 
        g:      g,
        pow:    &monkchain.EasyPow{},
    }
}

func (m *StdLibModel) PermLocator(addr []byte, perm string, state *monkstate.State) (*Location, error){
    // locator for perm w.r.t the address
    locator := vars.GetLinkedListElement(m.doug, "permnames", perm, state)
    locatorBig := monkutil.BigD(locator)

    return &Location{m.doug, locatorBig, nil}, nil
}

func (m *StdLibModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value{
    public := vars.GetSingle(m.doug, "public:"+perm, state)
    // A stand-in for a one day more sophisticated system
    if len(public) > 0{
        return monkutil.NewValue(1)
    }
    loc, err := m.PermLocator(addr, perm, state)
    if err != nil{
        fmt.Println("Sorrry tough guy, perm locator failed", err)
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
        tx, rec := MakeApplyTx("", m.doug, data, keys, block)
        txs = append(txs, tx)
        receipts = append(receipts, rec)
    }
    return txs, receipts
}

func (m *StdLibModel) SetValue(addr []byte, args []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    data := monkutil.PackTxDataArgs2(args...)
    tx, rec := MakeApplyTx("", addr, data, keys, block)
    return tx, rec
}

// Difficulty of the current block for a given coinbase
func (m *StdLibModel) Difficulty(block, parent *monkchain.Block) *big.Int{
    var b *big.Int
    consensusBytes := vars.GetSingle(m.doug, "consensus", prevBlock.State())
    consensus := string(consensusBytes)

    // compute difficulty according to consensus model
    // TODO: relieve methods from model struct
    if strings.Equal(consensus, "seq"){
        b = m.RoundRobinDifficulty(block, parent)
    } else if strings.Equal(consensus, "stake-weight"){
        b = m.StakeDifficulty(block, parent)
    } else {
        b = EthDifficulty(block, parent)
    }
    return b
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
        return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
    }
    
    // check if the block difficulty is correct 
    // it must be specified exactly
    newdiff := m.Difficulty(block, prevBlock)
    if block.Difficulty.Cmp(newdiff) != 0{
        return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)        
    }

    // TODO: is there a time when some consensus element is 
    // not specified in difficulty and must appear here?
    // Do we even budget for lists of signers/forgers and all
    // that nutty PoS stuff?

    // check block times
    if err:= CheckBlockTimes(prevBlock, block); err != nil{
        return err
    }

	// Verify the nonce of the block. Return an error if it's not valid
    // TODO: for now we leave pow on everything
    // soon we will want to generalize/relieve
    // also, variable hashing algos
	if !m.pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return monkchain.ValidationError("Block's nonce is invalid (= %v)", monkutil.Bytes2Hex(block.Nonce))
	}

    return nil
}

func (m *StdLibModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error{
    // check that sender has permission to transact or create
    var perm string
    if tx.IsContract(){
        perm = "create"
    } else{
        perm = "transact"
    }
    if !m.HasPermission(tx.Sender(), perm, state){
        return monkchain.InvalidPermError(tx.Sender(), perm)
    }
    // check that tx uses less than maxgas
    gas := tx.GasValue()
    max := vars.GetSingle(m.doug, "maxgastx", state) 
    maxBig := monkutil.BigD(max)
    if max != nil && gas.Cmp(maxBig) > 0 {
        return monkchain.GasLimitTxError(gas, maxBig)
    }
	// Make sure this transaction's nonce is correct
    sender := state.GetOrNewStateObject(tx.Sender())
	if sender.Nonce != tx.Nonce {
		return monkchain.NonceError(tx.Nonce, sender.Nonce)
	}
    return nil
}

type EthModel struct{
    pow monkchain.PoW
    g *GenesisConfig
}

func NewEthModel(g *GenesisConfig) PermModel{
    return &EthModel{&monkchain.EasyPow{}, g}
}

func (m *EthModel) SetPermissions(addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt){
    return nil, nil
}

func (m *EthModel) SetValue(addr []byte, data []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt){
    return nil, nil
}

func (m *EthModel) Difficulty(block, parent *monkchain.Block) *big.Int{
    return EthDifficulty(block, parent)
}

func (m *EthModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return nil
}

func (m *EthModel) ValidateBlock(block *monkchain.Block) error{
    // we have to verify using the state of the previous block!
    prevBlock := monkchain.GetBlock(block.PrevHash)

    // check that signature of block matches miners coinbase
    // XXX: not strictly necessary for eth...
    if !bytes.Equal(block.Signer(), block.Coinbase){
        return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
    }
    
	// check if the difficulty is correct
    newdiff := m.Difficulty(block, prevBlock)
    if block.Difficulty.Cmp(newdiff) != 0{
        return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)        
    }
    
    // check block times
    if err:= CheckBlockTimes(prevBlock, block); err != nil{
        return err
    }

	// Verify the nonce of the block. Return an error if it's not valid
	if !m.pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return monkchain.ValidationError("Block's nonce is invalid (= %v)", monkutil.Bytes2Hex(block.Nonce))
	}

    return nil
}

func (m *EthModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error{
	// Make sure this transaction's nonce is correct
    sender := state.GetOrNewStateObject(tx.Sender())
	if sender.Nonce != tx.Nonce {
		return monkchain.NonceError(tx.Nonce, sender.Nonce)
	}
    return nil
}
