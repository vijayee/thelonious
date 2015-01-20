package core

import (
	"math/big"

	"github.com/eris-ltd/new-thelonious/core/types"
	monkstate "github.com/eris-ltd/new-thelonious/state"
)

type OldVMEnv struct {
	state *monkstate.StateDB
	block *types.Block
	tx    Message
}

func OldNewEnv(state *monkstate.StateDB, tx Message, block *types.Block) *OldVMEnv {
	return &OldVMEnv{
		state: state,
		block: block,
		tx:    tx,
	}
}

func (self *OldVMEnv) Origin() []byte            { return self.tx.From() }
func (self *OldVMEnv) BlockNumber() *big.Int     { return self.block.Number() }
func (self *OldVMEnv) PrevHash() []byte          { return self.block.ParentHash() }
func (self *OldVMEnv) Coinbase() []byte          { return self.block.Coinbase() }
func (self *OldVMEnv) Time() int64               { return self.block.Time() }
func (self *OldVMEnv) Difficulty() *big.Int      { return self.block.Difficulty() }
func (self *OldVMEnv) BlockHash() []byte         { return self.block.Hash() }
func (self *OldVMEnv) Value() *big.Int           { return self.tx.Value() }
func (self *OldVMEnv) State() *monkstate.StateDB { return self.state }
func (self *OldVMEnv) Doug() []byte              { return genDoug.Doug() }
func (self *OldVMEnv) DougValidate(addr []byte, role string, state *monkstate.StateDB) error {
	return genDoug.ValidatePerm(addr, role, state)
}
