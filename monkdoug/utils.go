package monkdoug

import (
	"fmt"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/thelonious/monkutil"
	"io/ioutil"
	"math/big"
	"os"
	"path"
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
func NewContract(scriptFile string) *monkchain.Transaction {
	// if mutan, load the script. else, pass file name
	var s string
	if scriptFile[len(scriptFile)-3:] == ".mu" {
		r, err := ioutil.ReadFile(scriptFile)
		if err != nil {
			fmt.Println("could not load contract!", scriptFile, err)
			os.Exit(0)
		}
		s = string(r)
	} else {
		s = scriptFile
	}
	script, err := monkutil.Compile(string(s), false)
	if err != nil {
		fmt.Println("failed compile", err)
		os.Exit(0)
	}

	// create tx
	tx := monkchain.NewContractCreationTx(monkutil.Big("543"), monkutil.Big("10000"), monkutil.Big("10000"), script)

	return tx
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
func MakeApplyTx(codePath string, addr, data []byte, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt) {
	var tx *monkchain.Transaction
	if codePath != "" {
		tx = NewContract(codePath)
	} else {
		tx = monkchain.NewTransactionMessage(addr, monkutil.Big("0"), monkutil.Big("10000"), monkutil.Big("10000"), data)
	}

	tx.Sign(keys.PrivateKey)
	//fmt.Println(tx.String())
	receipt := SimpleTransitionState(addr, block, tx)
	txs := append(block.Transactions(), tx)
	receipts := append(block.Receipts(), receipt)
	block.SetReceipts(receipts, txs)

	return tx, receipt
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

// Run data through evm code and return value
func EvmCall(code, data []byte, dump bool) []byte {
	gas := "1000000000000000"
	price := "10000000"

	stateObject := state.NewStateObject([]byte("evmuser"))
	closure := vm.NewClosure(nil, stateObject, stateObject, code, ethutil.Big(gas), ethutil.Big(price))

	env := monkchain.NewEnv()
	ret, _, e := closure.Call(vm.New(env, vm.DebugVmTy), data)

	logger.Flush()
	if e != nil {
		perr(e)
	}

	if dump {
		fmt.Println(string(env.state.Dump()))
	}

	return ret
}

func SetValue(addr []byte, args []string, keys *monkcrypto.KeyPair, block *monkchain.Block) (*monkchain.Transaction, *monkchain.Receipt) {
	data := monkutil.PackTxDataArgs2(args...)
	tx, rec := MakeApplyTx("", addr, data, keys, block)
	return tx, rec
}

func SetPermissions(genAddr, addr []byte, permissions map[string]int, block *monkchain.Block, keys *monkcrypto.KeyPair) (monkchain.Transactions, []*monkchain.Receipt) {
	txs := monkchain.Transactions{}
	receipts := []*monkchain.Receipt{}

	for perm, val := range permissions {
		data := monkutil.PackTxDataArgs2("setperm", perm, "0x"+monkutil.Bytes2Hex(addr), "0x"+strconv.Itoa(val))
		tx, rec := MakeApplyTx("", genAddr, data, keys, block)
		txs = append(txs, tx)
		receipts = append(receipts, rec)
	}
	return txs, receipts
}
