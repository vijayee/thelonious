package doug

import (
	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/state"
	"math/big"
)

func (m *StdLibModel) consensus(st *state.StateDB) string {
	consensusBytes := GetSingle(m.doug, "consensus", st)
	consensus := string(consensusBytes)
	return consensus
}

func (m *StdLibModel) blocktime(st *state.StateDB) int64 {
	blockTimeBytes := GetSingle(m.doug, "blocktime", st)
	blockTime := thelutil.BigD(blockTimeBytes).Int64()
	return blockTime
}

// TODO !
func (m *StdLibModel) CheckUncles(prevBlock, block *types.Block) error {
	// Check each uncle's previous hash. In order for it to be valid
	// is if it has the same block hash as the current
	/*
		for _, uncle := range block.Uncles {
			if bytes.Compare(uncle.PrevHash,prevBlock.PrevHash) != 0 {
				return ValidationError("Mismatch uncle's previous hash. Expected %x, got %x",prevBlock.PrevHash, uncle.PrevHash)
			}
		}
	*/
	return nil
}

func CheckBlockTimes(prevBlock, block *types.Block) error {
	diff := block.Time() - prevBlock.Time()
	if diff < 0 {
		return core.ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time(), prevBlock.Time())
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/
	return nil
}

// Adjust difficulty to meet block time
// TODO: testing and robustify. this is leaky
func adjustDifficulty(oldDiff *big.Int, oldTime, newTime, target int64) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Rsh(oldDiff, 8)
	if newTime >= oldTime+target {
		diff.Sub(oldDiff, adjust)
	} else {
		diff.Add(oldDiff, adjust)
	}
	return diff
}

// difficulty targets a specific block time
func EthDifficulty(timeTarget int64, block, parent *types.Block) *big.Int {
	return adjustDifficulty(parent.Difficulty(), parent.Time(), block.Time(), timeTarget)
}
