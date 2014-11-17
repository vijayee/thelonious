package monkdoug

import (
    "bytes"
    "github.com/eris-ltd/thelonious/monkchain"
    vars "github.com/eris-ltd/eris-std-lib/go-tests"
)

// TODO: miner failover through self-adjusting difficulty
func (m *StdLibModel) CheckRoundRobin(prevBlock, block *monkchain.Block) error{
    // check that its the miners turn in the round robin
    if !bytes.Equal(prevBlock.PrevHash, monkchain.ZeroHash256){
        // if its not the genesis block, get coinbase of last block
        // find next entry in linked list
        prevCoinbase := prevBlock.Coinbase
        nextCoinbase, _ := vars.GetNextLinkedListElement(m.doug, "seq:name", string(prevCoinbase), prevBlock.State())
        if !bytes.Equal(nextCoinbase, block.Coinbase){
            return monkchain.InvalidTurnError(block.Coinbase, nextCoinbase)        
        }
    } else{
        // is it is genesis block, find first entry in linked list
        nextCoinbase, _ := vars.GetLinkedListHead(m.doug, "seq:name", prevBlock.State())
        if !bytes.Equal(nextCoinbase, block.Coinbase){
            return monkchain.InvalidTurnError(block.Coinbase, nextCoinbase)        
        }
    }
    return nil
}

// TODO !
func (m *StdLibModel) CheckUncles(prevBlock, block *monkchain.Block) error{
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

func (m *StdLibModel) CheckBlockTimes(prevBlock, block *monkchain.Block) error{
	diff := block.Time - prevBlock.Time
	if diff < 0 {
		return monkchain.ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time, prevBlock.Time)
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/
    return nil
}

func (m *EthModel) CheckBlockTimes(prevBlock, block *monkchain.Block) error{
	diff := block.Time - prevBlock.Time
	if diff < 0 {
		return monkchain.ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time, prevBlock.Time)
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/
    return nil
}
