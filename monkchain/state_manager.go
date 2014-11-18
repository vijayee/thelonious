package monkchain

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkwire"
)

var statelogger = monklog.NewLogger("STATE")

type BlockProcessor interface {
	ProcessBlock(block *Block)
}

type Peer interface {
	Inbound() bool
	LastSend() time.Time
	LastPong() int64
	Host() []byte
	Port() uint16
	Version() string
	PingTime() string
	Connected() *int32
	Caps() *monkutil.Value
}

type EthManager interface {
	StateManager() *StateManager
	BlockChain() *BlockChain
	TxPool() *TxPool
	Broadcast(msgType monkwire.MsgType, data []interface{})
	Reactor() *monkreact.ReactorEngine
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() *list.List
	KeyManager() *monkcrypto.KeyManager
	ClientIdentity() monkwire.ClientIdentity
	Db() monkutil.Database
    GenesisPointer(block *Block) // deploy the genesis block
    GenesisModel() GenDougModel // return the genesis model
}

// Model defining the protocol
type GenDougModel interface{
    Difficulty(coinbase []byte, block *Block) *big.Int
    ValidatePerm(addr []byte, role string, state *monkstate.State) error
    ValidateBlock(block *Block) error
    ValidateTx(tx *Transaction, state *monkstate.State) error
}

// Private global genDoug variable for checking permissions on arbitrary
// chain related actions. Set by setLastBlock when we boot up the blockchain
var genDoug GenDougModel

// Public function so we can validate permissions using the genDoug from outside this package
func DougValidatePerm(addr []byte, role string, state *monkstate.State) error{
    return genDoug.ValidatePerm(addr, role, state)
}

type StateManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex
	// Canonical block chain
	bc *BlockChain
	// non-persistent key/value memory storage
	mem map[string]*big.Int
	// Proof of work used for validating
	Pow PoW
	// The ethereum manager interface
	Ethereum EthManager
	// The managed states
	// Transiently state. The trans state isn't ever saved, validated and
	// it could be used for setting account nonces without effecting
	// the main states.
	transState *monkstate.State
	// Mining state. The mining state is used purely and solely by the mining
	// operation.
	miningState *monkstate.State

	// The last attempted block is mainly used for debugging purposes
	// This does not have to be a valid block and will be set during
	// 'Process' & canonical validation.
	lastAttemptedBlock *Block
}

func NewStateManager(ethereum EthManager) *StateManager {
	sm := &StateManager{
		mem:      make(map[string]*big.Int),
		Pow:      &EasyPow{},
		Ethereum: ethereum,
		bc:       ethereum.BlockChain(),
	}
	sm.transState = ethereum.BlockChain().CurrentBlock.State().Copy()
	sm.miningState = ethereum.BlockChain().CurrentBlock.State().Copy()

	return sm
}

func (sm *StateManager) CurrentState() *monkstate.State {
	return sm.Ethereum.BlockChain().CurrentBlock.State()
}

func (sm *StateManager) TransState() *monkstate.State {
	return sm.transState
}

func (sm *StateManager) MiningState() *monkstate.State {
	return sm.miningState
}

func (sm *StateManager) NewMiningState() *monkstate.State {
	sm.miningState = sm.Ethereum.BlockChain().CurrentBlock.State().Copy()

	return sm.miningState
}

func (sm *StateManager) BlockChain() *BlockChain {
	return sm.bc
}

func (self *StateManager) ProcessTransactions(coinbase *monkstate.StateObject, state *monkstate.State, block, parent *Block, txs Transactions) (Receipts, Transactions, Transactions, error) {
	var (
		receipts           Receipts
		handled, unhandled Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
	)

done:
	for i, tx := range txs {
		txGas := new(big.Int).Set(tx.Gas)

		cb := state.GetStateObject(coinbase.Address())
        // TODO: deal with this
		st := NewStateTransitionEris(cb, tx, state, block, self.bc.Genesis()) // ERIS
		err = st.TransitionState()
		if err != nil {
			statelogger.Infoln(err)
			switch {
			case IsNonceErr(err):
                self.Ethereum.Reactor().Post("newTx:post:fail", &TxFail{tx, err})
				err = nil // ignore error
				continue
            case IsGasLimitTxErr(err):
                self.Ethereum.Reactor().Post("newTx:post:fail", &TxFail{tx, err})
				err = nil // ignore error
				continue
			case IsGasLimitErr(err):
				unhandled = txs[i:]
                for _, t := range unhandled{
                    self.Ethereum.Reactor().Post("newTx:post:fail", &TxFail{t, err})
                }
				break done
			default:
				statelogger.Infoln("this tx registered an error and may have failed:", err)
				err = nil
                // TODO: should this have a tx:fail ?
				//return nil, nil, nil, err
			}
		}

        if st.msg != nil{
            // if msg is nil, an error should have triggered above
            // publish return value
            self.Ethereum.Reactor().Post("tx:"+string(tx.Hash())+":return", st.msg.Output)
        }

		// Notify all subscribers
		self.Ethereum.Reactor().Post("newTx:post", tx)

		// Update the state with pending changes
		state.Update()

		txGas.Sub(txGas, st.gas)
		accumelative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, txGas))
		receipt := &Receipt{tx, monkutil.CopyBytes(state.Root().([]byte)), accumelative}

		if i < len(block.Receipts()) {
			original := block.Receipts()[i]
			if !original.Cmp(receipt) {
				if monkutil.Config.Diff {
					os.Exit(1)
				}

				err := fmt.Errorf("#%d receipt failed (r) %v ~ %x  <=>  (c) %v ~ %x (%x...)", i+1, original.CumulativeGasUsed, original.PostState[0:4], receipt.CumulativeGasUsed, receipt.PostState[0:4], receipt.Tx.Hash()[0:4])

				return nil, nil, nil, err
			}
		}

		receipts = append(receipts, receipt)
		handled = append(handled, tx)

		if monkutil.Config.Diff && monkutil.Config.DiffType == "all" {
			state.CreateOutputForDiff()
		}
	}

	parent.GasUsed = totalUsedGas

	return receipts, handled, unhandled, err
}

func (sm *StateManager) Process(block *Block, dontReact bool) (err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.bc.HasBlock(block.Hash()) {
		return nil
	}

	if !sm.bc.HasBlock(block.PrevHash) {
		return ParentError(block.PrevHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.PrevHash)
		state  = parent.State()
	)

	// Defer the Undo on the Trie. If the block processing happened
	// we don't want to undo but since undo only happens on dirty
	// nodes this won't happen because Commit would have been called
	// before that.
	defer state.Reset()

	if monkutil.Config.Diff && monkutil.Config.DiffType == "all" {
		fmt.Printf("## %x %x ##\n", block.Hash(), block.Number)
	}

	receipts, err := sm.ApplyDiff(state, parent, block)
	if err != nil {
		return err
	}

	txSha := CreateTxSha(receipts)
	if bytes.Compare(txSha, block.TxSha) != 0 {
		return fmt.Errorf("Error validating tx sha. Received %x, got %x", block.TxSha, txSha)
	}

	// Block validation
	if err = sm.ValidateBlock(block); err != nil {
		statelogger.Errorln("Error validating block:", err)
		return err
	}

	if err = sm.AccumelateRewards(state, block, parent); err != nil {
		statelogger.Errorln("Error accumulating reward", err)
		return err
	}

	state.Update()

	if !block.State().Cmp(state) {
		err = fmt.Errorf("Invalid merkle root.\nrec: %x\nis:  %x", block.State().Trie.Root, state.Trie.Root)
		return
	}

	// Calculate the new total difficulty and sync back to the db
	if sm.CalculateTD(block) {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		state.Sync()

		// Add the block to the chain
		sm.bc.Add(block)

		sm.transState = state.Copy()

		// Create a bloom bin for this block
		filter := sm.createBloomFilter(state)
		// Persist the data
		fk := append([]byte("bloom"), block.Hash()...)
		sm.Ethereum.Db().Put(fk, filter.Bin())

		statelogger.Infof("Imported block #%d (%x...)\n", block.Number, block.Hash()[0:4])
		if dontReact == false {
			sm.Ethereum.Reactor().Post("newBlock", block)

			state.Manifest().Reset()
		}

		sm.Ethereum.TxPool().RemoveInvalid(state)
	} else {
		statelogger.Errorln("total diff failed")
	}

	return nil
}

func (sm *StateManager) ApplyDiff(state *monkstate.State, parent, block *Block) (receipts Receipts, err error) {
	coinbase := state.GetOrNewStateObject(block.Coinbase)
	coinbase.SetGasPool(block.CalcGasLimit(parent))

	// Process the transactions on to current block
	receipts, _, _, err = sm.ProcessTransactions(coinbase, state, block, parent, block.Transactions())
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (sm *StateManager) CalculateTD(block *Block) bool {
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(sm.bc.TD, uncleDiff)
	td = td.Add(td, block.Difficulty)

	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(sm.bc.TD) > 0 {
		// Set the new total difficulty back to the block chain
		sm.bc.SetTotalDifficulty(td)

		return true
	}

	return false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *StateManager) ValidateBlock(block *Block) error {
    // all validation is done through the genDoug
    return genDoug.ValidateBlock(block)
}

func (sm *StateManager) AccumelateRewards(state *monkstate.State, block, parent *Block) error {
	reward := new(big.Int).Set(BlockReward)

	knownUncles := monkutil.Set(parent.Uncles)
	nonces := monkutil.NewSet(block.Nonce)
	for _, uncle := range block.Uncles {
		if nonces.Include(uncle.Nonce) {
			// Error not unique
			return UncleError("Uncle not unique")
		}

		uncleParent := sm.bc.GetBlock(uncle.PrevHash)
		if uncleParent == nil {
			return UncleError("Uncle's parent unknown")
		}

		if uncleParent.Number.Cmp(new(big.Int).Sub(parent.Number, big.NewInt(6))) < 0 {
			return UncleError("Uncle too old")
		}

		if knownUncles.Include(uncle.Hash()) {
			return UncleError("Uncle in chain")
		}

		nonces.Insert(uncle.Nonce)

		r := new(big.Int)
		r.Mul(BlockReward, big.NewInt(15)).Div(r, big.NewInt(16))

		uncleAccount := state.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}

	// Get the account associated with the coinbase
	account := state.GetAccount(block.Coinbase)
	// Reward amount of ether to the coinbase address
	account.AddAmount(reward)

	return nil
}

func (sm *StateManager) Stop() {
	sm.bc.Stop()
}

// Manifest will handle both creating notifications and generating bloom bin data
func (sm *StateManager) createBloomFilter(state *monkstate.State) *BloomFilter {
	bloomf := NewBloomFilter(nil)

	for _, msg := range state.Manifest().Messages {
		bloomf.Set(msg.To)
		bloomf.Set(msg.From)
	}

	sm.Ethereum.Reactor().Post("messages", state.Manifest().Messages)

	return bloomf
}

func (sm *StateManager) GetMessages(block *Block) (messages []*monkstate.Message, err error) {
	if !sm.bc.HasBlock(block.PrevHash) {
		return nil, ParentError(block.PrevHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.PrevHash)
		state  = parent.State().Copy()
	)

	defer state.Reset()

	sm.ApplyDiff(state, parent, block)

	sm.AccumelateRewards(state, block, parent)

	return state.Manifest().Messages, nil
}
