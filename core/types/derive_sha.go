package types

import (
	"github.com/eris-ltd/new-thelonious/theldb"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) []byte {
	db, _ := theldb.NewMemDatabase()
	trie := trie.New(nil, db)
	for i := 0; i < list.Len(); i++ {
		trie.Update(thelutil.Encode(i), list.GetRlp(i))
	}

	return trie.Root()
}
