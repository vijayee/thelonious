package monkdoug

import (
    "math/big"
    "bytes"

    "github.com/eris-ltd/thelonious/monkchain"
    "github.com/eris-ltd/thelonious/monkutil"
    vars "github.com/eris-ltd/eris-std-lib/go-tests"
)

// Base difficulty of the chain is 2^($difficulty), with $difficulty 
// stored in GenDoug
func (m *StdLibModel) baseDifficulty(state *monkstate.State) *big.Int{
    difv := vars.GetSingle(m.doug, "difficulty", state) 
    return monkutil.BigPow(2, int(monkutil.ReadVarInt(difv)))
}

// Difficulty for miners in a round robin
func (m *StdLibModel) RoundRobinDifficulty(block, parent *monkchain.Block) *big.Int{
    state := parent.State()
    newdiff := m.baseDifficulty(state)
    // find relative position of coinbase in the linked list (i)
    // difficulty should be (base difficulty)*2^i
    var i int
    nMiners := vars.GetLinkedListLength(m.doug, "seq:name", state)
    // this is the proper next coinbase
    next := m.nextCoinbase(parent)
    for i = 0; i<nMiners; i++ {
        if bytes.Equal(next, block.Coinbase){
            break
        }
        next, _ = vars.GetNextLinkedListElement(m.doug, "seq:name", string(next), state)
    }
    newdiff = big.NewInt(0).Mul(monkutil.BigPow(2, i), newdiff)
    return newdiff
}

func (m *StdLibModel) StakeDifficulty(block, parent *monkchain.Block) *big.Int{
    //TODO
    return nil
}

func EthDifficulty(block, parent *monkchain.Block) *big.Int{
    diff := new(big.Int)

    adjust := new(big.Int).Rsh(parent.Difficulty, 10)
    if block.Time >= parent.Time+5 {
        diff.Sub(parent.Difficulty, adjust)
    } else {
        diff.Add(parent.Difficulty, adjust)
    }
    return diff
}
