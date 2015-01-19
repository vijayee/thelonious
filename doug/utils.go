package doug

import (
	"fmt"
	"github.com/eris-ltd/lllc-server"
	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/thelutil"
	monkstate "github.com/eris-ltd/new-thelonious/state"
	"github.com/eris-ltd/new-thelonious/vm"
	"math/big"
	"os"
	"strconv"
)

var (
	GoPath = os.Getenv("GOPATH")
)

/*
   Functions for updating state without all the weight
   of the standard protocol.
   Mostly used for setting up the genesis block and for running
   local VM scripts (ie for computing consensus)
*/

// create a new tx from a script, with dummy keypair
// creates tx but does not sign!
func NewContract(scriptFile string) (*types.Transaction, error) {
	// if mutan, load the script. else, pass file name
	script, err := lllcserver.Compile(scriptFile)
	if err != nil {
		fmt.Println("failed compile", err)
		return nil, err
	}

	// create tx
	tx := types.NewContractCreationTx(thelutil.Big("543"), thelutil.Big("10000"), thelutil.Big("10000"), script)

	return tx, nil
}

// Apply a tx to the genesis block
func SimpleTransitionState(addr []byte, state *monkstate.StateDB, tx *types.Transaction) (*types.Receipt, error) {
	coinbase := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//	st := core.NewStateTransition(state.GetOrNewStateObject(coinbase), tx, state, nil)
	var env vm.Environment
	//var msg core.Message
	st := core.NewStateTransition(env, tx, state.GetOrNewStateObject(coinbase))
	st.AddGas(thelutil.Big("10000000000000000000000000000000000000000000000000000000000000000000000000000000000")) // gas is silly, but the vm needs it

	// if receiver address is given, use it
	// else, standard contract addr
	var receiver *monkstate.StateObject
	if addr != nil {
		receiver = state.GetOrNewStateObject(addr)
	} else {
		//receiver = st.MakeStateObject(state, tx)
		// TODO: tx to msg!
		receiver = core.MakeContract(tx, state)
	}

	sender := state.GetOrNewStateObject(tx.From())
	//	value := thelutil.Big("12342")

	//var script []byte
	var ret []byte
	var err error
	//var ref vm.ContextRef
	if types.IsContractAddr(tx.To()) {
		receiver.SetBalance(thelutil.Big("123456789098765432"))
		receiver.InitCode = tx.Data()
		//script = receiver.Init()
		//ret, err = st.Eval(tx, script, receiver, "genesis")
		ret, err, _ = env.Create(sender, receiver.Address(), tx.Data(), tx.Gas(), tx.GasPrice(), tx.Value())
	} else {
		//script = receiver.Code
		//ret, err = st.Eval(tx, script, receiver, "genesis")
		ret, err = env.Call(sender, tx.To(), tx.Data(), tx.Gas(), tx.GasPrice(), tx.Value())
	}

	if err != nil {
		return nil, fmt.Errorf("Eval error in simple transition state:", err.Error())
	}

	if types.IsContractAddr(tx.To()) {
		receiver.Code = ret
	}
	//msg.Output = ret

	//root := state.Root()

	//TODO: receipt := &types.Receipt{tx, thelutil.CopyBytes(root), new(big.Int)}
	receipt := new(types.Receipt)

	sender.Nonce += 1
	// remove stateobject used to deploy gen doug
	// state.DeleteStateObject(sender)
	return receipt, nil
}

// Make and apply an administrative tx (simplified vm processing).
// If addr is empty or invalid, use proper contract address.
// Include a codePath if it's a contract or data if its a tx
func MakeApplyTx(codePath string, addr, data []byte, keys *crypto.KeyPair, state *monkstate.StateDB) (*types.Transaction, *types.Receipt, error) {
	var tx *types.Transaction
	var err error
	if codePath != "" {
		tx, err = NewContract(codePath)
		if err != nil {
			return nil, nil, err
		}
	} else {
		tx = types.NewTransactionMessage(addr, thelutil.Big("0"), thelutil.Big("10000"), thelutil.Big("10000"), data)
	}
	acc := state.GetOrNewStateObject(keys.Address())
	tx.SetNonce(acc.Nonce)

	tx.Sign(keys.PrivateKey)
	receipt, err := SimpleTransitionState(addr, state, tx)
	if err != nil {
		return nil, nil, err
	}
	//txs := append(block.Transactions(), tx)
	//receipts := append(block.Receipts(), receipt)
	//block.SetReceipts(receipts, txs)

	return tx, receipt, nil
}

func String2Big(s string) *big.Int {
	// right pad the string, convert to big num
	return thelutil.BigD(thelutil.PackTxDataArgs(s))
}

/*// pretty print chain queries and storage
func PrintHelp(m map[string]interface{}, obj *monkstate.StateObject) {
	for k, v := range m {
		if vv, ok := v.(*thelutil.Value); ok {
			fmt.Println(k, thelutil.Bytes2Hex(vv.Bytes()))
		} else if vv, ok := v.(*big.Int); ok {
			fmt.Println(k, thelutil.Bytes2Hex(vv.Bytes()))
		} else if vv, ok := v.([]byte); ok {
			fmt.Println(k, thelutil.Bytes2Hex(vv))
		}
	}
	obj.EachStorage(func(k string, v *thelutil.Value) {
		fmt.Println(thelutil.Bytes2Hex([]byte(k)), thelutil.Bytes2Hex(v.Bytes()))
	})
}*/

func SetValue(addr []byte, args []string, keys *crypto.KeyPair, state *monkstate.StateDB) (*types.Transaction, *types.Receipt) {
	data := thelutil.PackTxDataArgs2(args...)
	tx, rec, _ := MakeApplyTx("", addr, data, keys, state)
	return tx, rec
}

func GetValue(addr []byte, query string, state *monkstate.StateDB) []byte {
	// TODO: get values from gendoug
	return nil
}

func SetPermissions(genAddr, addr []byte, permissions map[string]int, state *monkstate.StateDB, keys *crypto.KeyPair) (types.Transactions, []*types.Receipt) {
	txs := types.Transactions{}
	receipts := []*types.Receipt{}

	for perm, val := range permissions {
		data := thelutil.PackTxDataArgs2("setperm", perm, "0x"+thelutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
		tx, rec, _ := MakeApplyTx("", genAddr, data, keys, state)
		txs = append(txs, tx)
		receipts = append(receipts, rec)
	}
	return txs, receipts
}

// Run data through evm code and return value
func (m *VmModel) EvmCall(code, data []byte, stateObject *monkstate.StateObject, state *monkstate.StateDB, tx *types.Transaction, block *types.Block, dump bool) []byte {
	//gas := "10000000000000000000000"
	//msg := &monkstate.Message{}

	//closure := vm.NewClosure(msg, stateObject, stateObject, code, thelutil.Big(gas), thelutil.Big(price))

	env := NewEnv(state, tx, block, m.g.consensus)
	//vm.Verbose = true
	//ret, _, e := closure.Call(vm, data)
	ret, err := env.Call(stateObject, tx.To(), tx.Data(), tx.Gas(), tx.GasPrice(), tx.Value())

	if err != nil {
		fmt.Println("vm error!", err)
	}

	/*if dump {
		fmt.Println(string(env.State().Dump()))
	}*/

	return ret
}

type VMEnv struct {
	consensus core.Consensus
	state     *monkstate.StateDB
	block     *types.Block
	tx        *types.Transaction
	caller    []byte
}

func NewEnv(state *monkstate.StateDB, tx *types.Transaction, block *types.Block, consensus core.Consensus) *VMEnv {
	return &VMEnv{
		consensus: consensus,
		state:     state,
		block:     block,
		tx:        tx,
	}
}

func (self *VMEnv) Origin() []byte        { return []byte("000000000000000LOCAL") } //self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int { return nil }                            //self.block.Number }
func (self *VMEnv) PrevHash() []byte      { return nil }                            //self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte      { return nil }                            //self.block.Coinbase }
func (self *VMEnv) Time() int64           { return 0 }                              //self.block.Time }
func (self *VMEnv) Difficulty() *big.Int  { return nil }                            //self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte     { return nil }                            //self.block.Hash() }
func (self *VMEnv) Value() *big.Int       { return big.NewInt(0) }
func (self *VMEnv) GetHash(n uint64) []byte {
	/*	if block := self.chain.GetBlockByNumber(n); block != nil {
		return block.Hash()
	}*/

	return nil
}
func (self *VMEnv) GasLimit() *big.Int        { return self.block.GasLimit() }
func (self *VMEnv) State() *monkstate.StateDB { return self.state }
func (self *VMEnv) Depth() int                { return 0 } //self.depth }

func (self *VMEnv) SetDepth(i int) {} //self.depth = i }
func (self *VMEnv) Doug() []byte   { return self.consensus.Doug() }
func (self *VMEnv) DougValidate(addr []byte, role string, state *monkstate.StateDB) error {
	return self.consensus.ValidatePerm(addr, role, state)
}

func (self *VMEnv) AddLog(log monkstate.Log) {
	self.state.AddLog(log)
}

func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *VMEnv) vm(addr, data []byte, gas, price, value *big.Int) *core.Execution {
	return core.NewExecution(self, addr, data, gas, price, value)
}

func (self *VMEnv) Call(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Call(addr, me)
}
func (self *VMEnv) CallCode(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(me.Address(), data, gas, price, value)
	return exe.Call(addr, me)
}

func (self *VMEnv) Create(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(me)
}
