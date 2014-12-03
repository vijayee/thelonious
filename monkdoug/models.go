package monkdoug

import (
	"bytes"
	"fmt"
	"math/big"
	"time"
	//"log"
	vars "github.com/eris-ltd/eris-std-lib/go-tests"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monkutil"
)

var Adversary = 0

/*
   Permission models are used for setting up the genesis block
   and for validating blocks and transactions.
   They allow for arbitrary extensions of consensus
*/

// The yes model grants all permissions
type YesModel struct {
	g *GenesisConfig
}

func NewYesModel(g *GenesisConfig) monkchain.Protocol {
	return &YesModel{g}
}

func (m *YesModel) Doug() []byte {
	return nil
}

func (m *YesModel) Deploy(block *monkchain.Block) {
	m.g.Deploy(block)
}

func (m *YesModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	return true
}

func (m *YesModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return nil
}

func (m *YesModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	return nil
}

func (m *YesModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	return nil
}

// The no model grants no permissions
type NoModel struct {
	g *GenesisConfig
}

func NewNoModel(g *GenesisConfig) monkchain.Protocol {
	return &NoModel{g}
}

func (m *NoModel) Doug() []byte {
	return nil
}

func (m *NoModel) Deploy(block *monkchain.Block) {
	m.g.Deploy(block)
}

func (m *NoModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	// we tell it to start mining even though we know it will fail
	// because this model is mostly just used for testing...
	return true
}

func (m *NoModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	return fmt.Errorf("No!")
}

// The VM Model runs all processing through the EVM
type VmModel struct {
	g    *GenesisConfig
	doug []byte

	// map of contract names to syscalls
	// names are json tags, addresses are
	// left-padded struct field names (VmConsensus struct)
	contract map[string]SysCall
}

func NewVmModel(g *GenesisConfig) monkchain.Protocol {
	contract := make(map[string]SysCall)
	return &VmModel{g, g.byteAddr, contract}
}

func (m *VmModel) Doug() []byte {
	return m.doug
}

func (m *VmModel) Deploy(block *monkchain.Block) {
	m.g.Deploy(block)
}

// TODO:
//  - enforce read-only option for vm

func (m *VmModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	if scall, ok := m.contract["compute-participate"]; ok {
		addr := scall.byteAddr
		state := parent.State()
		obj, code := m.pickCallObjAndCode(addr, state, scall.Doug)
		coinbaseHex := monkutil.Bytes2Hex(coinbase)
		data := monkutil.PackTxDataArgs2(coinbaseHex)
		ret := m.EvmCall(code, data, obj, state, nil, parent, true)
		// TODO: check not nil
		return monkutil.BigD(ret).Uint64() > 0
	}
	return true
}

func (m *VmModel) pickCallObjAndCode(addr []byte, state *monkstate.State, useDoug bool) (obj *monkstate.StateObject, code []byte) {
	obj = state.GetStateObject(addr)
	code = obj.Code
	if useDoug {
		obj = state.GetStateObject(m.doug)
	}
	return
}

func (m *VmModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	if scall, ok := m.contract["compute-difficulty"]; ok {
		addr := scall.byteAddr
		state := parent.State()
		obj, code := m.pickCallObjAndCode(addr, state, scall.Doug)
		coinbase := monkutil.Bytes2Hex(block.Coinbase)
		data := monkutil.PackTxDataArgs2(coinbase)
		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		// TODO: check not nil
		return monkutil.BigD(ret)
	}
	return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *VmModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	var ret []byte
	if scall, ok := m.contract["permission-verify"]; ok {
		contract := scall.byteAddr
		obj, code := m.pickCallObjAndCode(contract, state, scall.Doug)
		data := monkutil.PackTxDataArgs2(monkutil.Bytes2Hex(addr), role)
		ret = m.EvmCall(code, data, obj, state, nil, nil, true)
	} else {
		// get perm from doug
		doug := state.GetStateObject(m.doug)
		data := monkutil.PackTxDataArgs2("checkperm", role, "0x"+monkutil.Bytes2Hex(addr))
		ret = m.EvmCall(doug.Code, data, doug, state, nil, nil, true)
	}
	if monkutil.BigD(ret).Uint64() > 0 {
		return nil
	}
	return fmt.Errorf("Permission error")
}

func (m *VmModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	if scall, ok := m.contract["block-verify"]; ok {
		addr := scall.byteAddr
		parent := bc.CurrentBlock()
		state := parent.State()
		obj, code := m.pickCallObjAndCode(addr, state, scall.Doug)
		// get block args
		prevhash := block.PrevHash
		unclesha := block.UncleSha
		coinbase := block.Coinbase
		stateroot := monkutil.NewValue(block.GetRoot()).Bytes()
		txsha := block.TxSha
		diff := block.Difficulty.Bytes()
		number := block.Number.Bytes()
		minGasPrice := block.MinGasPrice.Bytes()
		gasLim := block.GasLimit.Bytes()
		gasUsed := block.GasUsed.Bytes()
		t := big.NewInt(block.Time).Bytes()
		extra := []byte(block.Extra)
		sig := block.GetSig()

		prevdiff := parent.Difficulty.Bytes()
		prevT := big.NewInt(parent.Time).Bytes()

		data := monkutil.PackTxDataBytes(prevhash, unclesha, coinbase, stateroot, txsha, diff, prevdiff, number, minGasPrice, gasLim, gasUsed, t, prevT, extra, sig)

		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		if monkutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	return m.ValidatePerm(block.Coinbase, "mine", block.State())
}

func (m *VmModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	if scall, ok := m.contract["tx-verify"]; ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state, scall.Doug)
		// get tx args
		nonce := big.NewInt(int64(tx.Nonce)).Bytes() // TODO: safe cast?
		rec := tx.Recipient
		value := tx.Value.Bytes()
		gas := tx.Gas.Bytes()
		gasPrice := tx.GasPrice.Bytes()
		data := tx.Data
		sig := tx.GetSig()

		data = monkutil.PackTxDataBytes(nonce, rec, value, gas, gasPrice, sig, data)
		ret := m.EvmCall(code, data, obj, state, tx, nil, true)
		if monkutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	return m.ValidatePerm(tx.Sender(), "transact", state)
}

// The stdlib model grants permissions based on the state of the gendoug
// It depends on the eris-std-lib for its storage model
type StdLibModel struct {
	base *big.Int
	doug []byte
	g    *GenesisConfig
	pow  monkchain.PoW
}

func NewStdLibModel(g *GenesisConfig) monkchain.Protocol {
	return &StdLibModel{
		base: new(big.Int),
		doug: g.byteAddr,
		g:    g,
		pow:  &monkchain.EasyPow{},
	}
}

func (m *StdLibModel) Doug() []byte {
	return m.doug
}

func (m *StdLibModel) Deploy(block *monkchain.Block) {
	m.g.Deploy(block)
}

func (m *StdLibModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value {
	public := vars.GetSingle(m.doug, "public:"+perm, state)
	// A stand-in for a one day more sophisticated system
	if len(public) > 0 {
		return monkutil.NewValue(1)
	}
	locator := vars.GetLinkedListElement(m.doug, "permnames", perm, state)
	locatorBig := monkutil.BigD(locator)
	locInt := locatorBig.Uint64()
	permStr := vars.GetKeyedArrayElement(m.doug, "perms", monkutil.Bytes2Hex(addr), int(locInt), state)
	return monkutil.NewValue(permStr)
}

func (m *StdLibModel) HasPermission(addr []byte, perm string, state *monkstate.State) bool {
	permBig := m.GetPermission(addr, perm, state).BigInt()
	return permBig.Int64() > 0
}

// Save energy in the round robin by not mining until close to your turn
// or too much time has gone by
func (m *StdLibModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	if Adversary != 0 {
		return true
	}

	consensus := m.consensus(parent.State())
	// if we're not in a round robin, always mine
	if consensus != "robin" {
		return true
	}
	// find out our distance from the current next miner
	next := m.nextCoinbase(parent)
	nMiners := vars.GetLinkedListLength(m.doug, "seq:name", parent.State())
	var i int
	for i = 0; i < nMiners; i++ {
		next, _ = vars.GetNextLinkedListElement(m.doug, "seq:name", string(next), parent.State())
		if bytes.Equal(next, coinbase) {
			break
		}
	}
	// if we're less than halfway from the current miner, we should mine
	if i <= int(nMiners/2) {
		return true
	}
	// if we're more than halfway, but enough time has gone by, we should mine
	mDiff := i - int(nMiners/2)
	t := parent.Time
	cur := time.Now().Unix()
	blocktime := m.blocktime(parent.State())
	tDiff := (cur - t) / blocktime
	if tDiff > int64(mDiff) {
		return true
	}
	// otherwise, we should not mine
	return false
}

// Difficulty of the current block for a given coinbase
func (m *StdLibModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	var b *big.Int

	consensus := m.consensus(parent.State())

	// compute difficulty according to consensus model
	switch consensus {
	case "robin":
		b = m.RoundRobinDifficulty(block, parent)
	case "stake-weight":
		b = m.StakeDifficulty(block, parent)
	case "constant":
		b = m.baseDifficulty(parent.State())
	default:
		blockTime := m.blocktime(parent.State())
		b = EthDifficulty(blockTime, block, parent)
	}
	return b
}

func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	if Adversary != 0 {
		return nil
	}
	if m.HasPermission(addr, role, state) {
		return nil
	}
	return monkchain.InvalidPermError(addr, role)
}

func (m *StdLibModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	if Adversary != 0 {
		return nil
	}

	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.PrevHash)

	// check that miner has permission to mine
	if !m.HasPermission(block.Coinbase, "mine", prevBlock.State()) {
		return monkchain.InvalidPermError(block.Coinbase, "mine")
	}

	// check that signature of block matches miners coinbase
	if !bytes.Equal(block.Signer(), block.Coinbase) {
		return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
	}

	// check if the block difficulty is correct
	// it must be specified exactly
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty.Cmp(newdiff) != 0 {
		return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)
	}

	// TODO: is there a time when some consensus element is
	// not specified in difficulty and must appear here?
	// Do we even budget for lists of signers/forgers and all
	// that nutty PoS stuff?

	// check block times
	if err := CheckBlockTimes(prevBlock, block); err != nil {
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

func (m *StdLibModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	if Adversary != 0 {
		return nil
	}

	// check that sender has permission to transact or create
	var perm string
	if tx.IsContract() {
		perm = "create"
	} else {
		perm = "transact"
	}
	if !m.HasPermission(tx.Sender(), perm, state) {
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

type EthModel struct {
	pow monkchain.PoW
	g   *GenesisConfig
}

func NewEthModel(g *GenesisConfig) monkchain.Protocol {
	return &EthModel{&monkchain.EasyPow{}, g}
}

func (m *EthModel) Doug() []byte {
	return nil
}

func (m *EthModel) Deploy(block *monkchain.Block) {
	m.g.Deploy(block)
}

func (m *EthModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	return true
}

func (m *EthModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return EthDifficulty(int64(m.g.BlockTime), block, parent)
}

func (m *EthModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return nil
}

func (m *EthModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.PrevHash)

	// check that signature of block matches miners coinbase
	// XXX: not strictly necessary for eth...
	if !bytes.Equal(block.Signer(), block.Coinbase) {
		return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
	}

	// check if the difficulty is correct
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty.Cmp(newdiff) != 0 {
		return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)
	}

	// check block times
	if err := CheckBlockTimes(prevBlock, block); err != nil {
		return err
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !m.pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return monkchain.ValidationError("Block's nonce is invalid (= %v)", monkutil.Bytes2Hex(block.Nonce))
	}

	return nil
}

func (m *EthModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	// Make sure this transaction's nonce is correct
	sender := state.GetOrNewStateObject(tx.Sender())
	if sender.Nonce != tx.Nonce {
		return monkchain.NonceError(tx.Nonce, sender.Nonce)
	}
	return nil
}
