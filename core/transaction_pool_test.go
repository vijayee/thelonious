package core

import (
	"crypto/ecdsa"
	"testing"

	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/ethdb"
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/event"
	"github.com/eris-ltd/new-thelonious/state"
)

// State query interface
type stateQuery struct{ db monkutil.Database }

func SQ() stateQuery {
	db, _ := ethdb.NewMemDatabase()
	return stateQuery{db: db}
}

func (self stateQuery) GetAccount(addr []byte) *state.StateObject {
	return state.NewStateObject(addr, self.db)
}

func transaction() *types.Transaction {
	return types.NewTransactionMessage(make([]byte, 20), monkutil.Big0, monkutil.Big0, monkutil.Big0, nil)
}

func setup() (*TxPool, *ecdsa.PrivateKey) {
	var m event.TypeMux
	key, _ := crypto.GenerateKey()
	return NewTxPool(&m), key
}

func TestTxAdding(t *testing.T) {
	pool, key := setup()
	tx1 := transaction()
	tx1.SignECDSA(key)
	err := pool.Add(tx1)
	if err != nil {
		t.Error(err)
	}

	err = pool.Add(tx1)
	if err == nil {
		t.Error("added tx twice")
	}
}

func TestAddInvalidTx(t *testing.T) {
	pool, _ := setup()
	tx1 := transaction()
	err := pool.Add(tx1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestRemoveSet(t *testing.T) {
	pool, _ := setup()
	tx1 := transaction()
	pool.addTx(tx1)
	pool.RemoveSet(types.Transactions{tx1})
	if pool.Size() > 0 {
		t.Error("expected pool size to be 0")
	}
}

func TestRemoveInvalid(t *testing.T) {
	pool, key := setup()
	tx1 := transaction()
	pool.addTx(tx1)
	pool.RemoveInvalid(SQ())
	if pool.Size() > 0 {
		t.Error("expected pool size to be 0")
	}

	tx1.SetNonce(1)
	tx1.SignECDSA(key)
	pool.addTx(tx1)
	pool.RemoveInvalid(SQ())
	if pool.Size() != 1 {
		t.Error("expected pool size to be 1, is", pool.Size())
	}
}
