package state

import (
	"encoding/json"
	"fmt"

	"github.com/eris-ltd/new-thelonious/thelutil"
)

type Account struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	Root     string            `json:"root"`
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
}

type World struct {
	Root     string             `json:"root"`
	Accounts map[string]Account `json:"accounts"`
}

func (self *StateDB) Dump() []byte {
	world := World{
		Root:     thelutil.Bytes2Hex(self.trie.Root()),
		Accounts: make(map[string]Account),
	}

	it := self.trie.Iterator()
	for it.Next() {
		stateObject := NewStateObjectFromBytes(it.Key, it.Value, self.db)

		account := Account{Balance: stateObject.balance.String(), Nonce: stateObject.Nonce, Root: thelutil.Bytes2Hex(stateObject.Root()), CodeHash: thelutil.Bytes2Hex(stateObject.codeHash)}
		account.Storage = make(map[string]string)

		storageIt := stateObject.State.trie.Iterator()
		for storageIt.Next() {
			account.Storage[thelutil.Bytes2Hex(it.Key)] = thelutil.Bytes2Hex(it.Value)
		}
		world.Accounts[thelutil.Bytes2Hex(it.Key)] = account
	}

	json, err := json.MarshalIndent(world, "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}

// Debug stuff
func (self *StateObject) CreateOutputForDiff() {
	fmt.Printf("%x %x %x %x\n", self.Address(), self.State.Root(), self.balance.Bytes(), self.Nonce)
	it := self.State.trie.Iterator()
	for it.Next() {
		fmt.Printf("%x %x\n", it.Key, it.Value)
	}
}
