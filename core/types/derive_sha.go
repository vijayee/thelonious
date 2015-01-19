package types

import (
	"github.com/eris-ltd/new-thelonious/ethdb"
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) []byte {
	db, _ := ethdb.NewMemDatabase()
	trie := trie.New(nil, db)
	for i := 0; i < list.Len(); i++ {
		trie.Update(monkutil.Encode(i), list.GetRlp(i))
	}

	return trie.Root()
}
