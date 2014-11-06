package monkpipe

import (
	"math/big"

	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkstate"
)

type VMEnv struct {
	state  *monkstate.State
	block  *monkchain.Block
	value  *big.Int
	sender []byte
}

func NewEnv(state *monkstate.State, block *monkchain.Block, value *big.Int, sender []byte) *VMEnv {
	return &VMEnv{
		state:  state,
		block:  block,
		value:  value,
		sender: sender,
	}
}

func (self *VMEnv) Origin() []byte         { return self.sender }
func (self *VMEnv) BlockNumber() *big.Int  { return self.block.Number }
func (self *VMEnv) PrevHash() []byte       { return self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte       { return self.block.Coinbase }
func (self *VMEnv) Time() int64            { return self.block.Time }
func (self *VMEnv) Difficulty() *big.Int   { return self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte      { return self.block.Hash() }
func (self *VMEnv) Value() *big.Int        { return self.value }
func (self *VMEnv) State() *monkstate.State { return self.state }
func (self *VMEnv) DougValidate(addr []byte, state *monkstate.State, role string) bool{ return monkchain.DougValidate(addr, state, role)}
