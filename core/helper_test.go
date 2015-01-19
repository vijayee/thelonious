package core

import (
	"container/list"
	"fmt"

	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/ethdb"
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/event"
	"github.com/eris-ltd/new-thelonious/p2p"
)

// Implement our EthTest Manager
type TestManager struct {
	// stateManager *StateManager
	eventMux *event.TypeMux

	db         monkutil.Database
	txPool     *TxPool
	blockChain *ChainManager
	Blocks     []*types.Block
}

func (s *TestManager) IsListening() bool {
	return false
}

func (s *TestManager) IsMining() bool {
	return false
}

func (s *TestManager) PeerCount() int {
	return 0
}

func (s *TestManager) Peers() *list.List {
	return list.New()
}

func (s *TestManager) ChainManager() *ChainManager {
	return s.blockChain
}

func (tm *TestManager) TxPool() *TxPool {
	return tm.txPool
}

// func (tm *TestManager) StateManager() *StateManager {
// 	return tm.stateManager
// }

func (tm *TestManager) EventMux() *event.TypeMux {
	return tm.eventMux
}
func (tm *TestManager) Broadcast(msgType p2p.Msg, data []interface{}) {
	fmt.Println("Broadcast not implemented")
}

func (tm *TestManager) ClientIdentity() p2p.ClientIdentity {
	return nil
}
func (tm *TestManager) KeyManager() *crypto.KeyManager {
	return nil
}

func (tm *TestManager) Db() monkutil.Database {
	return tm.db
}

func NewTestManager() *TestManager {
	monkutil.ReadConfig(".ethtest", "/tmp/ethtest", "ETH")

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		fmt.Println("Could not create mem-db, failing")
		return nil
	}

	testManager := &TestManager{}
	testManager.eventMux = new(event.TypeMux)
	testManager.db = db
	// testManager.txPool = NewTxPool(testManager)
	// testManager.blockChain = NewChainManager(testManager)
	// testManager.stateManager = NewStateManager(testManager)

	// Start the tx pool
	testManager.txPool.Start()

	return testManager
}
