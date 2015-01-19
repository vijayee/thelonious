package doug

import (
	"fmt"
	"math/big"
	//"log"
	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/pow"
	ezp "github.com/eris-ltd/new-thelonious/pow/ezp"
	monkstate "github.com/eris-ltd/new-thelonious/state"
)

var Adversary = 0

type Protocol struct {
	g         *GenesisConfig
	consensus core.Consensus
}

func (p *Protocol) Doug() []byte {
	return p.g.byteAddr
}

func (p *Protocol) Deploy(block *types.Block) ([]byte, error) {
	// TODO: try deployer, fall back to default deployer
	return p.g.Deployer(block)
}

func (p *Protocol) ValidateChainID(chainId []byte, genesisBlock *types.Block) error {
	return nil
}

// Determine whether to accept a new checkpoint
func (p *Protocol) Participate(coinbase []byte, parent *types.Block) bool {
	return p.consensus.Participate(coinbase, parent)
}

func (p *Protocol) Difficulty(block, parent *types.Block) *big.Int {
	return p.consensus.Difficulty(block, parent)
}

func (p *Protocol) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	return p.consensus.ValidatePerm(addr, role, state)
}

func (p *Protocol) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	return p.consensus.ValidateBlock(block, bc)
}

func (p *Protocol) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	return p.consensus.ValidateTx(tx, state)
}

func (p *Protocol) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	return p.consensus.CheckPoint(proposed, bc)
}

// The yes model grants all permissions
type YesModel struct {
	g *GenesisConfig
}

func (m *YesModel) Doug() []byte {
	return nil
}

func (m *YesModel) Deploy(block *types.Block) ([]byte, error) {
	return nil, nil
}

func NewYesModel(g *GenesisConfig) core.Consensus {
	return &YesModel{g}
}

func (m *YesModel) Participate(coinbase []byte, parent *types.Block) bool {
	return true
}

func (m *YesModel) Difficulty(block, parent *types.Block) *big.Int {
	return thelutil.BigPow(2, m.g.Difficulty)
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	return nil
}

func (m *YesModel) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	return nil
}

func (m *YesModel) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	return nil
}

func (m *YesModel) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	return true
}

// The no model grants no permissions
type NoModel struct {
	g *GenesisConfig
}

func (m *NoModel) Doug() []byte {
	return nil
}

func (m *NoModel) Deploy(block *types.Block) ([]byte, error) {
	return nil, nil
}

func NewNoModel(g *GenesisConfig) core.Consensus {
	return &NoModel{g}
}

func (m *NoModel) Participate(coinbase []byte, parent *types.Block) bool {
	// we tell it to start mining even though we know it will fail
	// because this model is mostly just used for testing...
	return true
}

func (m *NoModel) Difficulty(block, parent *types.Block) *big.Int {
	return thelutil.BigPow(2, m.g.Difficulty)
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	return false
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

func NewVmModel(g *GenesisConfig) core.Consensus {
	contract := make(map[string]SysCall)
	return &VmModel{g, g.byteAddr, contract}
}

func (m *VmModel) Doug() []byte {
	return m.doug
}

func (m *VmModel) Deploy(block *types.Block) ([]byte, error) {
	return nil, nil
}

// TODO:
//  - enforce read-only option for vm (no SSTORE)

func (m *VmModel) Participate(coinbase []byte, parent *types.Block) bool {
	state := monkstate.New(parent.Root(), m.g.db)
	if scall, ok := m.getSysCall("compute-participate", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		coinbaseHex := thelutil.Bytes2Hex(coinbase)
		data := thelutil.PackTxDataArgs2(coinbaseHex)
		ret := m.EvmCall(code, data, obj, state, nil, parent, true)
		// TODO: check not nil
		return thelutil.BigD(ret).Uint64() > 0
	}
	return true
}

func (m *VmModel) pickCallObjAndCode(addr []byte, state *monkstate.StateDB) (obj *monkstate.StateObject, code []byte) {
	obj = state.GetStateObject(addr)
	code = obj.Code
	//if useDoug {
	//	obj = state.GetStateObject(m.doug)
	//}
	return
}

func (m *VmModel) getSysCall(name string, state *monkstate.StateDB) (SysCall, bool) {
	if s, ok := m.contract[name]; ok {
		return s, ok
	}

	addr := GetValue(m.doug, name, state)
	if addr != nil {
		return SysCall{byteAddr: addr}, true
	}
	return SysCall{}, false
}

func (m *VmModel) Difficulty(block, parent *types.Block) *big.Int {
	state := monkstate.New(parent.Root(), m.g.db)
	if scall, ok := m.getSysCall("compute-difficulty", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		data := packBlockParent(block, parent)
		douglogger.Infoln("Calling difficulty contract")
		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		//fmt.Println("RETURN DIF:", ret)
		// TODO: check not nil
		return thelutil.BigD(ret)
	}
	r := EthDifficulty(5*60, block, parent)
	//fmt.Println("RETURN DIF:", r)
	return r
	//return thelutil.BigPow(2, m.g.Difficulty)
}

func packBlockParent(block, parent *types.Block) []byte {
	block1rlp := thelutil.Encode(block.Header())
	l1 := len(block1rlp)
	l1bytes := big.NewInt(int64(l1)).Bytes()
	block2rlp := thelutil.Encode(parent.Header())
	l2 := len(block2rlp)
	l2bytes := big.NewInt(int64(l2)).Bytes()

	// data is
	// (len block 1), (block 1), (len block 2), (block 2), (len sig for block 1), (sig block 1)
	data := []byte{}
	data = append(data, thelutil.LeftPadBytes(l1bytes, 32)...)
	data = append(data, block1rlp...)
	data = append(data, thelutil.LeftPadBytes(l2bytes, 32)...)
	data = append(data, block2rlp...)
	return data
}

func (m *VmModel) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	var ret []byte
	if scall, ok := m.getSysCall("permission-verify", state); ok {
		contract := scall.byteAddr
		obj, code := m.pickCallObjAndCode(contract, state)
		data := thelutil.PackTxDataArgs2(thelutil.Bytes2Hex(addr), role)
		douglogger.Infoln("Calling permision verify contract")
		ret = m.EvmCall(code, data, obj, state, nil, nil, true)
	} else {
		// get perm from doug
		doug := state.GetStateObject(m.doug)
		data := thelutil.PackTxDataArgs2("checkperm", role, "0x"+thelutil.Bytes2Hex(addr))
		douglogger.Infoln("Calling permision verify (GENDOUG) contract")
		ret = m.EvmCall(doug.Code, data, doug, state, nil, nil, true)
	}
	if thelutil.BigD(ret).Uint64() > 0 {
		return nil
	}
	return fmt.Errorf("Permission error")
}

func (m *VmModel) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	parent := bc.CurrentBlock()
	state := monkstate.New(parent.Root(), m.g.db)

	if scall, ok := m.getSysCall("block-verify", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		//sig := block.GetSig()

		/*sigrlp := thelutil.Encode([]interface{}{sig[:32], sig[32:64], thelutil.RightPadBytes([]byte{sig[64] - 27}, 32)})
		lsig := len(sigrlp)
		lsigbytes := big.NewInt(int64(lsig)).Bytes()*/

		// data is
		// (len block 1), (block 1), (len block 2), (block 2), (len sig for block 1), (sig block 1)
		data := packBlockParent(block, parent)
		//data = append(data, thelutil.LeftPadBytes(lsigbytes, 32)...)
		//data = append(data, sigrlp...)

		douglogger.Infoln("Calling block verify contract")
		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		if thelutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	return m.ValidatePerm(block.Coinbase(), "mine", state)
}

func (m *VmModel) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	if scall, ok := m.getSysCall("tx-verify", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		data := tx.RlpEncode()
		l := big.NewInt(int64(len(data))).Bytes()
		data = append(thelutil.LeftPadBytes(l, 32), data...)

		douglogger.Infoln("Calling tx verify contract")
		ret := m.EvmCall(code, data, obj, state, tx, nil, true)
		if thelutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	var perm string
	if types.IsContractAddr(tx.To()) {
		perm = "create"
	} else {
		perm = "transact"
	}
	return m.ValidatePerm(tx.From(), perm, state)
}

func (m *VmModel) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	// TODO: checkpoint validation contract
	return true
}

// The stdlib model grants permissions based on the state of the gendoug
// It depends on the eris-std-lib for its storage model
type StdLibModel struct {
	base *big.Int
	doug []byte
	g    *GenesisConfig
	poW  pow.PoW
}

func (m *StdLibModel) Doug() []byte {
	return m.doug
}

func (m *StdLibModel) Deploy(block *types.Block) ([]byte, error) {
	return nil, nil
}

func NewStdLibModel(g *GenesisConfig) core.Consensus {
	return &StdLibModel{
		base: new(big.Int),
		doug: g.byteAddr,
		g:    g,
		poW:  &ezp.EasyPow{},
	}
}

func (m *StdLibModel) GetPermission(addr []byte, perm string, state *monkstate.StateDB) *thelutil.Value {
	public := GetSingle(m.doug, "public:"+perm, state)
	// A stand-in for a one day more sophisticated system
	if len(public) > 0 {
		return thelutil.NewValue(1)
	}
	locator := GetLinkedListElement(m.doug, "permnames", perm, state)
	locatorBig := thelutil.BigD(locator)
	locInt := locatorBig.Uint64()
	permStr := GetKeyedArrayElement(m.doug, "perms", thelutil.Bytes2Hex(addr), int(locInt), state)
	return thelutil.NewValue(permStr)
}

func (m *StdLibModel) HasPermission(addr []byte, perm string, state *monkstate.StateDB) bool {
	permBig := m.GetPermission(addr, perm, state).BigInt()
	return permBig.Int64() > 0
}

// Save energy in the round robin by not mining until close to your turn
// or too much time has gone by
func (m *StdLibModel) Participate(coinbase []byte, parent *types.Block) bool {
	return true
}

// Difficulty of the current block for a given coinbase
func (m *StdLibModel) Difficulty(block, parent *types.Block) *big.Int {
	var b *big.Int
	parentState := monkstate.New(parent.Root(), m.g.db)
	consensus := m.consensus(parentState)

	// compute difficulty according to consensus model
	switch consensus {
	case "constant":
		//TODO: b = m.baseDifficulty(parent.State())
	default:
		blockTime := m.blocktime(parentState)
		b = EthDifficulty(blockTime, block, parent)
	}
	return b
}

func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	if Adversary != 0 {
		return nil
	}
	if m.HasPermission(addr, role, state) {
		return nil
	}
	return core.InvalidPermError(addr, role)
}

func (m *StdLibModel) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	if Adversary != 0 {
		return nil
	}

	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.ParentHash())

	// check that miner has permission to mine
	prevState := monkstate.New(prevBlock.Root(), m.g.db)
	if !m.HasPermission(block.Coinbase(), "mine", prevState) {
		return core.InvalidPermError(block.Coinbase(), "mine")
	}

	// check that signature of block matches miners coinbase
	/*if !bytes.Equal(block.Signer(), block.Coinbase) {
		return core.InvalidSigError(block.Signer(), block.Coinbase)
	}*/

	// check if the block difficulty is correct
	// it must be specified exactly
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty().Cmp(newdiff) != 0 {
		return core.InvalidDifficultyError(block.Difficulty(), newdiff, block.Coinbase())
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
	if !m.poW.Verify(block) {
		return core.ValidationError("Block's nonce is invalid (= %v)", thelutil.Bytes2Hex(block.Nonce()))
	}

	return nil
}

func (m *StdLibModel) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	if Adversary != 0 {
		return nil
	}

	// check that sender has permission to transact or create
	var perm string
	if types.IsContractAddr(tx.To()) {
		perm = "create"
	} else {
		perm = "transact"
	}
	if !m.HasPermission(tx.From(), perm, state) {
		return core.InvalidPermError(tx.From(), perm)
	}
	// check that tx uses less than maxgas
	gas := tx.Gas()
	max := GetSingle(m.doug, "maxgastx", state)
	maxBig := thelutil.BigD(max)
	if max != nil && gas.Cmp(maxBig) > 0 {
		return core.GasLimitTxError(gas, maxBig)
	}
	// Make sure this transaction's nonce is correct
	sender := state.GetOrNewStateObject(tx.From())
	if sender.Nonce != tx.Nonce() {
		return core.NonceError(tx.Nonce(), sender.Nonce)
	}
	return nil
}

func (m *StdLibModel) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	// TODO: something reasonable
	return true
}

type EthModel struct {
	poW pow.PoW
	g   *GenesisConfig
}

func (m *EthModel) Doug() []byte {
	return nil
}

func (m *EthModel) Deploy(block *types.Block) ([]byte, error) {
	return nil, nil
}

func NewEthModel(g *GenesisConfig) core.Consensus {
	return &EthModel{&ezp.EasyPow{}, g}
}

func (m *EthModel) Participate(coinbase []byte, parent *types.Block) bool {
	return true
}

func (m *EthModel) Difficulty(block, parent *types.Block) *big.Int {
	return EthDifficulty(int64(m.g.BlockTime), block, parent)
}

func (m *EthModel) ValidatePerm(addr []byte, role string, state *monkstate.StateDB) error {
	return nil
}

func (m *EthModel) ValidateBlock(block *types.Block, bc *core.ChainManager) error {
	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.ParentHash())

	// check that signature of block matches miners coinbase
	// XXX: not strictly necessary for eth...
	/*if !bytes.Equal(block.Signer(), block.Coinbase) {
		return core.InvalidSigError(block.Signer(), block.Coinbase)
	}*/

	// check if the difficulty is correct
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty().Cmp(newdiff) != 0 {
		return core.InvalidDifficultyError(block.Difficulty(), newdiff, block.Coinbase())
	}

	// check block times
	if err := CheckBlockTimes(prevBlock, block); err != nil {
		return err
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !m.poW.Verify(block) {
		return core.ValidationError("Block's nonce is invalid (= %v)", thelutil.Bytes2Hex(block.Nonce()))
	}

	return nil
}

func (m *EthModel) ValidateTx(tx *types.Transaction, state *monkstate.StateDB) error {
	// Make sure this transaction's nonce is correct
	sender := state.GetOrNewStateObject(tx.From())
	if sender.Nonce != tx.Nonce() {
		return core.NonceError(tx.Nonce(), sender.Nonce)
	}
	return nil
}

func (m *EthModel) CheckPoint(proposed []byte, bc *core.ChainManager) bool {
	// TODO: can we authenticate eth checkpoints?
	//   or just do something reasonable
	return false
}
