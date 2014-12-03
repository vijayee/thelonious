package monkdoug

import (
	"fmt"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkvm"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"strconv"
)

var (
	GoPath  = os.Getenv("GOPATH")
	ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")
)

/*
   Functions for updating state without all the weight
   of the standard protocol.
   Mostly used for setting up the genesis block
*/

// create a new tx from a script, with dummy keypair
// creates tx but does not sign!
func NewContract(scriptFile string) (*monkchain.Transaction, error) {
	// if mutan, load the script. else, pass file name
	var s string
	if scriptFile[len(scriptFile)-3:] == ".mu" {
		r, err := ioutil.ReadFile(scriptFile)
		if err != nil {
			fmt.Println("could not load contract!", scriptFile, err)
			return nil, err
		}
		s = string(r)
	} else {
		s = scriptFile
	}
	script, err := monkutil.Compile(string(s), false)
	if err != nil {
		fmt.Println("failed compile", err)
		return nil, err
	}

	// create tx
	tx := monkchain.NewContractCreationTx(monkutil.Big("543"), monkutil.Big("10000"), monkutil.Big("10000"), script)

	return tx, nil
}

// apply tx to genesis block
func SimpleTransitionState(addr []byte, block *monkchain.Block, tx *monkchain.Transaction) *monkchain.Receipt {
	state := block.State()
	st := monkchain.NewStateTransition(monkstate.NewStateObject(block.Coinbase), tx, state, block)
	st.AddGas(monkutil.Big("10000000000000000000000000000000000000000000000000000000000000000000000000000000000")) // gas is silly, but the vm needs it

	var script []byte
	receiver := state.GetOrNewStateObject(addr)
	if tx.CreatesContract() {
		receiver.Balance = monkutil.Big("123456789098765432")
		receiver.InitCode = tx.Data
		receiver.State = monkstate.New(monktrie.New(monkutil.Config.Db, ""))
		script = receiver.Init()
	} else {
		script = receiver.Code
	}

	sender := state.GetOrNewStateObject(tx.Sender())
	value := monkutil.Big("12342")

	msg := state.Manifest().AddMessage(&monkstate.Message{
		To: receiver.Address(), From: sender.Address(),
		Input:  tx.Data,
		Origin: sender.Address(),
		Block:  block.Hash(), Timestamp: block.Time, Coinbase: block.Coinbase, Number: block.Number,
		Value: value,
	})
	// TODO: this should switch on creates contract (init vs code) ?
	ret, err := st.Eval(msg, script, receiver, "init")
	if err != nil {
		fmt.Println("Eval error in simple transition state:", err)
		os.Exit(0)
	}
	if tx.CreatesContract() {
		receiver.Code = ret
	}
	msg.Output = ret

	rootI := state.Root()
	var root []byte
	if r, ok := rootI.([]byte); ok {
		root = r
	} else if r, ok := rootI.(string); ok {
		root = []byte(r)
	}

	receipt := &monkchain.Receipt{tx, monkutil.CopyBytes(root), new(big.Int)}
	// remove stateobject used to deploy gen doug
	state.DeleteStateObject(sender)
	return receipt
}

// make and apply an administrative tx (simplified vm processing)
// addr is typically gendoug
// TODO: if addr is empty or invalid, use proper contract addr
func MakeApplyTx(codePath string, addr, data []byte, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt, error) {
	var tx *monkchain.Transaction
	var err error
	if codePath != "" {
		tx, err = NewContract(codePath)
		if err != nil {
			return nil, nil, err
		}
	} else {
		tx = monkchain.NewTransactionMessage(addr, monkutil.Big("0"), monkutil.Big("10000"), monkutil.Big("10000"), data)
	}

	tx.Sign(keys.PrivateKey)
	//fmt.Println(tx.String())
	receipt := SimpleTransitionState(addr, block, tx)
	txs := append(block.Transactions(), tx)
	receipts := append(block.Receipts(), receipt)
	block.SetReceipts(receipts, txs)

	return tx, receipt, nil
}

func String2Big(s string) *big.Int {
	// right pad the string, convert to big num
	return monkutil.BigD(monkutil.PackTxDataArgs(s))
}

// pretty print chain queries and storage
func PrintHelp(m map[string]interface{}, obj *monkstate.StateObject) {
	for k, v := range m {
		if vv, ok := v.(*monkutil.Value); ok {
			fmt.Println(k, monkutil.Bytes2Hex(vv.Bytes()))
		} else if vv, ok := v.(*big.Int); ok {
			fmt.Println(k, monkutil.Bytes2Hex(vv.Bytes()))
		} else if vv, ok := v.([]byte); ok {
			fmt.Println(k, monkutil.Bytes2Hex(vv))
		}
	}
	obj.EachStorage(func(k string, v *monkutil.Value) {
		fmt.Println(monkutil.Bytes2Hex([]byte(k)), monkutil.Bytes2Hex(v.Bytes()))
	})
}

func SetValue(addr []byte, args []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt) {
	data := monkutil.PackTxDataArgs2(args...)
	tx, rec, _ := MakeApplyTx("", addr, data, keys, block)
	return tx, rec
}

func SetPermissions(genAddr, addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt) {
	txs := monkchain.Transactions{}
	receipts := []*monkchain.Receipt{}

	for perm, val := range permissions {
		data := monkutil.PackTxDataArgs2("setperm", perm, "0x"+monkutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
		tx, rec, _ := MakeApplyTx("", genAddr, data, keys, block)
		txs = append(txs, tx)
		receipts = append(receipts, rec)
	}
	return txs, receipts
}

// Run data through evm code and return value
func (m *VmModel) EvmCall(code, data []byte, stateObject *monkstate.StateObject, state *monkstate.State, tx *monkchain.Transaction, block *monkchain.Block, dump bool) []byte {
	gas := "1000000000000000"
	price := "10000000"

	closure := monkvm.NewClosure(nil, stateObject, stateObject, code, monkutil.Big(gas), monkutil.Big(price))

	env := NewEnv(state, tx, block, m.g.protocol)
	vm := monkvm.New(env)
	vm.Verbose = true
	ret, _, e := closure.Call(vm, data)

	if e != nil {
		fmt.Println("vm error!", e)
	}

	/*if dump {
		fmt.Println(string(env.State().Dump()))
	}*/

	return ret
}

type VMEnv struct {
	protocol monkchain.Protocol
	state    *monkstate.State
	block    *monkchain.Block
	tx       *monkchain.Transaction
}

func NewEnv(state *monkstate.State, tx *monkchain.Transaction, block *monkchain.Block, protocol monkchain.Protocol) *VMEnv {
	return &VMEnv{
		protocol: protocol,
		state:    state,
		block:    block,
		tx:       tx,
	}
}

func (self *VMEnv) Origin() []byte          { return []byte("000000000000000LOCAL") } //self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int   { return nil }                            //self.block.Number }
func (self *VMEnv) PrevHash() []byte        { return nil }                            //self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte        { return nil }                            //self.block.Coinbase }
func (self *VMEnv) Time() int64             { return 0 }                              //self.block.Time }
func (self *VMEnv) Difficulty() *big.Int    { return nil }                            //self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte       { return nil }                            //self.block.Hash() }
func (self *VMEnv) Value() *big.Int         { return big.NewInt(0) }
func (self *VMEnv) State() *monkstate.State { return self.state }
func (self *VMEnv) Doug() []byte            { return self.protocol.Doug() }
func (self *VMEnv) DougValidate(addr []byte, role string, state *monkstate.State) error {
	return self.protocol.ValidatePerm(addr, role, state)
}
