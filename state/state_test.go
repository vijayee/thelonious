package state

import (
	checker "gopkg.in/check.v1"

	"github.com/eris-ltd/new-thelonious/theldb"
	"github.com/eris-ltd/new-thelonious/thelutil"
)

type StateSuite struct {
	state *StateDB
}

var _ = checker.Suite(&StateSuite{})

// var ZeroHash256 = make([]byte, 32)

func (s *StateSuite) TestDump(c *checker.C) {
	key := []byte{0x01}
	value := []byte("foo")
	s.state.trie.Update(key, value)
	dump := s.state.Dump()
	c.Assert(dump, checker.NotNil)
}

func (s *StateSuite) SetUpTest(c *checker.C) {
	thelutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	db, _ := theldb.NewMemDatabase()
	s.state = New(nil, db)
}

func (s *StateSuite) TestSnapshot(c *checker.C) {
	stateobjaddr := []byte("aa")
	storageaddr := thelutil.Big("0")
	data1 := thelutil.NewValue(42)
	data2 := thelutil.NewValue(43)

	// get state object
	stateObject := s.state.GetOrNewStateObject(stateobjaddr)
	// set inital state object value
	stateObject.SetStorage(storageaddr, data1)
	// get snapshot of current state
	snapshot := s.state.Copy()

	// get state object. is this strictly necessary?
	stateObject = s.state.GetStateObject(stateobjaddr)
	// set new state object value
	stateObject.SetStorage(storageaddr, data2)
	// restore snapshot
	s.state.Set(snapshot)

	// get state object
	stateObject = s.state.GetStateObject(stateobjaddr)
	// get state storage value
	res := stateObject.GetStorage(storageaddr)

	c.Assert(data1, checker.DeepEquals, res)
}
