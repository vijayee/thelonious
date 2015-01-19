package helper

import (
	"errors"
	"math/big"

	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/state"
	"github.com/eris-ltd/new-thelonious/vm"
)

type Env struct {
	depth        int
	state        *state.StateDB
	skipTransfer bool
	initial      bool
	Gas          *big.Int

	origin   []byte
	parent   []byte
	coinbase []byte

	number     *big.Int
	time       int64
	difficulty *big.Int
	gasLimit   *big.Int

	logs state.Logs
}

func NewEnv(state *state.StateDB) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *state.StateDB, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(state)

	env.origin = monkutil.Hex2Bytes(exeValues["caller"])
	env.parent = monkutil.Hex2Bytes(envValues["previousHash"])
	env.coinbase = monkutil.Hex2Bytes(envValues["currentCoinbase"])
	env.number = monkutil.Big(envValues["currentNumber"])
	env.time = monkutil.Big(envValues["currentTimestamp"]).Int64()
	env.difficulty = monkutil.Big(envValues["currentDifficulty"])
	env.gasLimit = monkutil.Big(envValues["currentGasLimit"])
	env.Gas = new(big.Int)

	return env
}

func (self *Env) Origin() []byte        { return self.origin }
func (self *Env) BlockNumber() *big.Int { return self.number }
func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() []byte      { return self.coinbase }
func (self *Env) Time() int64           { return self.time }
func (self *Env) Difficulty() *big.Int  { return self.difficulty }
func (self *Env) State() *state.StateDB { return self.state }
func (self *Env) GasLimit() *big.Int    { return self.gasLimit }
func (self *Env) GetHash(n uint64) []byte {
	return crypto.Sha3([]byte(big.NewInt(int64(n)).String()))
}
func (self *Env) AddLog(log state.Log) {
	self.logs = append(self.logs, log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) error {
	if self.skipTransfer {
		// ugly hack
		if self.initial {
			self.initial = false
			return nil
		}

		if from.Balance().Cmp(amount) < 0 {
			return errors.New("Insufficient balance in account")
		}

		return nil
	}
	return vm.Transfer(from, to, amount)
}

func (self *Env) vm(addr, data []byte, gas, price, value *big.Int) *core.Execution {
	exec := core.NewExecution(self, addr, data, gas, price, value)

	return exec
}

func (self *Env) Call(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err
}
func (self *Env) CallCode(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(caller.Address(), data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *Env) Create(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(caller)
}

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		to    = FromHex(exec["address"])
		from  = FromHex(exec["caller"])
		data  = FromHex(exec["data"])
		gas   = monkutil.Big(exec["gas"])
		price = monkutil.Big(exec["gasPrice"])
		value = monkutil.Big(exec["value"])
	)
	// Reset the pre-compiled contracts for VM tests.
	vm.Precompiled = make(map[string]*vm.PrecompiledAccount)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(state, env, exec)
	vmenv.skipTransfer = true
	vmenv.initial = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.logs, vmenv.Gas, err
}

func RunState(statedb *state.StateDB, env, tx map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		keyPair, _ = crypto.NewKeyPairFromSec([]byte(monkutil.Hex2Bytes(tx["secretKey"])))
		to         = FromHex(tx["to"])
		data       = FromHex(tx["data"])
		gas        = monkutil.Big(tx["gasLimit"])
		price      = monkutil.Big(tx["gasPrice"])
		value      = monkutil.Big(tx["value"])
		caddr      = FromHex(env["currentCoinbase"])
	)

	// Set pre compiled contracts
	vm.Precompiled = vm.PrecompiledContracts()

	coinbase := statedb.GetOrNewStateObject(caddr)
	coinbase.SetGasPool(monkutil.Big(env["currentGasLimit"]))

	message := NewMessage(keyPair.Address(), to, data, value, gas, price)
	Log.DebugDetailf("message{ to: %x, from %x, value: %v, gas: %v, price: %v }\n", message.to[:4], message.from[:4], message.value, message.gas, message.price)
	vmenv := NewEnvFromMap(statedb, env, tx)
	st := core.NewStateTransition(vmenv, message, coinbase)
	vmenv.origin = keyPair.Address()
	ret, err := st.TransitionState()
	statedb.Update(vmenv.Gas)

	return ret, vmenv.logs, vmenv.Gas, err
}

type Message struct {
	from, to          []byte
	value, gas, price *big.Int
	data              []byte
}

func NewMessage(from, to, data []byte, value, gas, price *big.Int) Message {
	return Message{from, to, value, gas, price, data}
}

func (self Message) Hash() []byte       { return nil }
func (self Message) From() []byte       { return self.from }
func (self Message) To() []byte         { return self.to }
func (self Message) GasPrice() *big.Int { return self.price }
func (self Message) Gas() *big.Int      { return self.gas }
func (self Message) Value() *big.Int    { return self.value }
func (self Message) Nonce() uint64      { return 0 }
func (self Message) Data() []byte       { return self.data }
