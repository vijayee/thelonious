package monkchain

import (
	"bytes"
	"fmt"
	"math/big"
    "container/list"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkutil"
	//"github.com/eris-ltd/thelonious/monkcrypto"
)

var chainlogger = monklog.NewLogger("CHAIN")

/*
    A canonical chain begins right away, with blocks saved in database
    New blocks placed in working tree
    If at any time a branch in workingTree gets higher diff
        blocks than canonical, a re-org is called, the new chain
        is copied into database, the old canonical is placed in workingTree
        and the relevant blocks are removed from workingTree if they have 
        no children
        XXX: should old canonical be removed from leveldb?
            how do we even track canonical?
            can we put it in a trie?
*/

type ChainManager struct {
	Ethereum EthManager
	// The famous, the fabulous Mister GENESIIIIIIS (block)
	genesisBlock *Block
	// Last known total difficulty
	TD *big.Int

    // Our canonical chain
	LastBlockNumber uint64
	CurrentBlock  *Block
	LastBlockHash []byte

    // Cache of competing chains
    // Every non-canonical block is cached here
    // When a link gets td > TD, we re-org
    workingTree map[string] *link
}

func NewChainManager(ethereum EthManager) *ChainManager{
	bc := &ChainManager{}
	bc.genesisBlock = NewBlockFromBytes(monkutil.Encode(Genesis))
	bc.Ethereum = ethereum
    bc.workingTree = make(map[string]*link)
    // Prepare the genesis block!
    bc.Ethereum.GenesisPointer(bc.genesisBlock)

	bc.setLastBlock()

	return bc
}

func (bc *ChainManager) Genesis() *Block {
	return bc.genesisBlock
}

// Only called by the miner
func (bc *ChainManager) NewBlock(coinbase []byte) *Block {
	var root interface{}
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.state.Trie.Root
		hash = bc.LastBlockHash
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
		monkutil.BigPow(2, 12),
		nil,
		"")

    // TODO: How do we feel about this
	block.MinGasPrice = big.NewInt(10000000000000)

	parent := bc.CurrentBlock
	if parent != nil {
        block.Difficulty = genDoug.Difficulty(block, parent)
		block.Number = new(big.Int).Add(bc.CurrentBlock.Number, monkutil.Big1)
		block.GasLimit = monkutil.BigPow(10, 50) //block.CalcGasLimit(bc.CurrentBlock)

	}

	return block
}

func (bc *ChainManager) HasBlock(hash []byte) bool {
	data, _ := monkutil.Config.Db.Get(hash)
	return len(data) != 0
}

// TODO: At one point we might want to save a block by prevHash in the db to optimise this...
func (bc *ChainManager) HasBlockWithPrevHash(hash []byte) bool {
	block := bc.CurrentBlock

	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(hash, block.PrevHash) == 0 {
			return true
		}
	}
	return false
}

func (bc *ChainManager) CalculateBlockTD(block *Block) *big.Int {
	blockDiff := new(big.Int)

	for _, uncle := range block.Uncles {
		blockDiff = blockDiff.Add(blockDiff, uncle.Difficulty)
	}
	blockDiff = blockDiff.Add(blockDiff, block.Difficulty)

	return blockDiff
}

func (bc *ChainManager) GenesisBlock() *Block {
	return bc.genesisBlock
}

func (self *ChainManager) GetChainHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		chain = append(chain, block.Hash())

		if block.Number.Cmp(monkutil.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.PrevHash)
	}

	return
}




/*
func AddTestNetFunds(block *Block) {
	for _, addr := range []string{
		"51ba59315b3a95761d0863b05ccc7a7f54703d99",
		"e4157b34ea9615cfbde6b4fda419828124b70c78",
		"b9c015918bdaba24b4ff057a92a3873d6eb201be",
		"6c386a4b26f73c802f34673f7248bb118f97424a",
		"cd2a3d9f938e13cd947ec05abc7fe734df8dd826",
		"2ef47100e0787b915105fd5e3f4ff6752079d5cb",
		"e6716f9544a56c530d868e4bfbacb172315bdead",
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4",
	} {
		codedAddr := monkutil.Hex2Bytes(addr)
		account := block.state.GetAccount(codedAddr)
		account.Balance = monkutil.Big("1606938044258990275541962092341162602522202993782792835301376") //monkutil.BigPow(2, 200)
		block.state.UpdateStateObject(account)
	}
}
*/

func (bc *ChainManager) setLastBlock() {

    // check for last block. if none exists, fire up a genesis
	data, _ := monkutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = block.Number.Uint64()

	} else {
        // genesis block must be prepared ahead of time
		bc.add(bc.genesisBlock)
		fk := append([]byte("bloom"), bc.genesisBlock.Hash()...)
		bc.Ethereum.Db().Put(fk, make([]byte, 255))
		bc.CurrentBlock = bc.genesisBlock
	}
    // set the genDoug model for determining chain permissions
    genDoug = bc.Ethereum.GenesisModel()

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = monkutil.BigD(monkutil.Config.Db.LastKnownTD())

	chainlogger.Infof("Last block (#%d) %x\n", bc.LastBlockNumber, bc.CurrentBlock.Hash())

}

func (bc *ChainManager) SetTotalDifficulty(td *big.Int) {
	monkutil.Config.Db.Put([]byte("LTD"), td.Bytes())
	bc.TD = td
}

// Add a block to the canonical chain and record addition information
func (bc *ChainManager) add(block *Block) {
	bc.writeBlockInfo(block)
	// Prepare the genesis block

	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	monkutil.Config.Db.Put(block.Hash(), encodedBlock)
	monkutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)
}

func (self *ChainManager) CalcTotalDiff(block *Block) (*big.Int, error) {
	parent := self.GetBlock(block.PrevHash)
	if parent == nil {
		return nil, fmt.Errorf("Unable to calculate total diff without known parent %x", block.PrevHash)
	}

	parentTd := parent.BlockInfo().TD

	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	td := new(big.Int)
	td = td.Add(parentTd, uncleDiff)
	td = td.Add(td, block.Difficulty)

	return td, nil
}

// Tries to return the block regardless of if it's canonical
// or simply workingTree
func (bc *ChainManager) GetBlock(hash []byte) *Block {
    b := bc.GetBlockCanonical(hash)
    if b == nil{
        b = bc.GetBlockWorking(hash)
    }
    return b
}

// Strictly returns workingTree blocks
func (bc *ChainManager) GetBlockWorking(hash []byte) *Block{
    if l, ok := bc.workingTree[string(hash)]; ok{
        return l.block
    }
    return nil
}

// Strictly returns canonical blocks
func (bc *ChainManager) GetBlockCanonical(hash []byte) *Block{
	data, _ := monkutil.Config.Db.Get(hash)
	if len(data) == 0 {
        return nil
	}
	return NewBlockFromBytes(data)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *Block {
	block := self.CurrentBlock
	for ; block != nil; block = self.GetBlock(block.PrevHash) {
		if block.Number.Uint64() == num {
			break
		}
	}

	if block != nil && block.Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
}

func (self *ChainManager) GetBlockBack(num uint64) *Block {
	block := self.CurrentBlock

	for ; num != 0 && block != nil; num-- {
		block = self.GetBlock(block.PrevHash)
	}

	return block
}

func (bc *ChainManager) BlockInfoByHash(hash []byte) BlockInfo {
	bi := BlockInfo{}
	data, _ := monkutil.Config.Db.Get(append(hash, []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (bc *ChainManager) BlockInfo(block *Block) BlockInfo {
	bi := BlockInfo{}
	data, _ := monkutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
func (bc *ChainManager) writeBlockInfo(block *Block) {
    if block.Number.Cmp(big.NewInt(0)) != 0{
	    bc.LastBlockNumber++
    }
	bi := BlockInfo{Number: bc.LastBlockNumber, Hash: block.Hash(), Parent: block.PrevHash, TD: bc.TD}

	// For now we use the block hash with the words "info" appended as key
	monkutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bc *ChainManager) Stop() {
	if bc.CurrentBlock != nil {
		chainlogger.Infoln("Stopped")
	}
}


// a link in the working tree
type link struct {
	block    *Block
	//messages state.Messages
	td       *big.Int

    parent *link
    children []*link
    
}

// Blockchain coming in from the block pool or from miners
type BlockChain struct {
	*list.List
}

func NewChain(blocks Blocks) *BlockChain {
	chain := &BlockChain{list.New()}
	for _, block := range blocks {
		chain.PushBack(&link{block, nil, nil, nil})
	}
	return chain
}

// Validate the new chain with respect to its parent
// Adds all blocks to the workingTree if none of them fail
// TODO: Note this will sync new states (we may not want that, but it shouldn't
//  get in the way, it's just storage we dont need to keep around. If it's an attack, we can study it later :) )
func (self *ChainManager) TestChain(chain *BlockChain) (td *big.Int, err error) {
    // Is the parent on canonical or in the workingTree?

    // We need to add this chain to the workingTree
    // since we use GetBlock all over the place
    // but if any ProcessWithParent returns an error,
    // we remove all the ones we hadn't already seen
    for e := chain.Front(); e != nil; e = e.Next(){
        l := e.Value.(*link)
        block := l.block
        s := string(block.Hash())
        // check if we've seen this block
        if _, ok := self.workingTree[s]; ok{
            // remove from chain so it won't be processed or removed
            chain.Remove(e)
        } else{
            // add parent
            if f := e.Prev(); f != nil{
                l.parent = f.Value.(*link)
            } else {
                // the parent is either in workingTree or on canonical
                if p := self.workingTree[string(block.PrevHash)]; p!=nil{
                    l.parent = p
                } else{
                    // sanity check that parent is on canonical
                    if b := self.GetBlockCanonical(block.PrevHash); b==nil{
                        return nil, fmt.Errorf("Chain does not have known parent")
                    }
                    // use nil as marker for branch off canonical
                    l.parent = nil
                }
            }
            // add child
            l.children = []*link{}
            if f := e.Next(); f != nil{
                l.children = append(l.children, f.Value.(*link))
            }
            // add to tree
            self.workingTree[s] = l
        }
    }

    // if any blocks fail, remove all
    // otherwise, add front of chain to child of its parent
    defer func(){
        if err != nil{
            for e := chain.Front(); e != nil; e = e.Next(){
                l := e.Value.(*link)
                block := l.block
                delete(self.workingTree, string(block.Hash()))
            }
        } else{
            e := chain.Front()
            if parent := e.Value.(*link).parent; parent != nil{
                // TODO: make sure this works (pointers ...)
                parent.children = append(parent.children, e.Value.(*link))
            }
        }
    }()

    // Process the chain starting from its parent to ensure its valid
	for e := chain.Front(); e != nil; e = e.Next() {
        l      := e.Value.(*link)
        block  := l.block
        parent := self.GetBlock(block.PrevHash)
        // note parent may be on canonical or a fork on workingTree

		if parent == nil {
			err = fmt.Errorf("incoming chain broken on hash %x\n", block.PrevHash[0:4])
			return
		}

		//var messages state.Messages
		td, err = self.Ethereum.StateManager().ProcessWithParent(block, parent)
		if err != nil {
			chainlogger.Infoln(err)
			chainlogger.Debugf("Block #%v failed (%x...)\n", block.Number, block.Hash()[0:4])
			chainlogger.Debugln(block)
			err = fmt.Errorf("incoming chain failed %v\n", err)
			return
		}

		l.td = td
		//l.messages = messages
	}
    return td, nil
    
}

// This function assumes you've done your checking. No validity checking is done at this stage anymore
// This function can go three ways
//      1) Add blocks to the canonical chain
//      2) Sum the difficulties of a chain in the working tree
//      3) if result of 2 > TD, cause a re-org, 
//              giving a new canonical chain,
//              putting the old canonical in the working tree
func (self *ChainManager) InsertChain(chain *BlockChain) {
    // All blocks passed validation. Ready to add to working tree
    // Determine if we are
    // 1) On top of the best chain
    // 2) A new fork off the best chain
    // 3) On top of a fork
    // 4) A new fork off a fork

    frontLink := chain.Front().Value.(*link)
    // first link in the new chain
    front := frontLink.block
    // link's anchor point 
    parent := self.GetBlock(front.PrevHash)
    // canonical head
    head := self.CurrentBlock
   
    // 1) Check if parent is top block on chain
    if bytes.Compare(head.Hash(), parent.Hash()) == 0{
        // We are lengthening canonical!
        // for each block, set the new difficulty, add to chain
        for e := chain.Front(); e != nil; e = e.Next() {
            link := e.Value.(*link)

            self.SetTotalDifficulty(link.td)
            self.add(link.block)
            delete(self.workingTree, string(link.block.Hash()))

            // XXX: Post. Do we do this here? Prob better for caller ...
            //self.Ethereum.Reactor().Post(NewBlockEvent{link.block})
            //self.Ethereum.Reactor().Post(link.messages)
        }

        // summarize
        b, e := chain.Front(), chain.Back()
        if b != nil && e != nil {
            front, back := b.Value.(*link).block, e.Value.(*link).block
            chainlogger.Infof("Imported %d blocks. #%v (%x) / %#v (%x)", chain.Len(), front.Number, front.Hash()[0:4], back.Number, back.Hash()[0:4])
        }
        return
    }

    // This block is part of a fork
    // Its parent is either on canonical (not in workingTree) 
    //  or it extends a fork that already exists
    // Sum the difficulty and check if its time for a re-org
    var td *big.Int
    if l, ok := self.workingTree[string(parent.Hash())]; !ok{
        // 2) This is a new fork off the main chain
        chainlogger.Infof("Fork detected off parent %x at height %d. Head %x at %d", parent.Hash(), parent.Number, head.Hash(), head.Number)
        
        // get TD of parent on canonical
        td = parent.BlockInfo().TD        
    } else {
        // 3,4) If it's on the working tree, it's either a new head of a fork
        // or a new fork off a fork, but we don't really care which
        chainlogger.Infoln("Extending a fork...")

        // Get the total diff from that node (it should have been updated
        //  when that node was added to workingTree)
        td = l.td
    }

    // Sum difficulties along this chain in the workingTree
    // TODO: Can we do this in TestChain?
    base := new(big.Int)
    for e := chain.Front(); e != nil; e = e.Next(){
        l := e.Value.(*link)
        block := l.block
        // add up difficulty
        td = base.Add(td, block.Difficulty)
        // this block should already be known
        b := self.workingTree[string(block.Hash())]
        b.td = td
    }

    // if the new chain is crowned most gangsta
    if td.Cmp(self.TD) > 0{
        chainlogger.Infoln("A fork has overtaken canonical. Time for a reorg!")

        self.reOrg(chain)
    }
    /*
	if td.Cmp(self.TD) <= 0 {
		err = &TDError{td, self.TD}
		return
	}*/
}

func (self *ChainManager) reOrg(chain *BlockChain){
    // Find branch point
    // Pop them off the top of canonical into a chain
    //  add the chain to working tree
    // Pop the new canonical chain out of workingTree and into database

    // Create array of blocks from new head back to branch point
    // Deletes them from workingTree
    // Uses memory links. Maybe we should use prev hashes?
	bchain := &BlockChain{list.New()}
    for l := chain.Back().Value.(*link); l != nil; l = l.parent{
        bchain.PushFront(&link{l.block, nil, nil, nil})
        delete(self.workingTree, string(l.block.Hash()))
    }

    ancestorHash := bchain.Front().Value.(*link).block.PrevHash
    ancestor := self.GetBlockCanonical(ancestorHash)
  
    oldHeadHash := self.LastBlockHash
    oldHead := self.GetBlockCanonical(oldHeadHash)
   
    // revert the blockchain 
    chainlogger.Infof("Reverting blockchain to block %x at height %d, a reversion of %d blocks", ancestorHash, ancestor.Number,new(big.Int).Sub(oldHead.Number,ancestor.Number))
    self.CurrentBlock = ancestor
    self.LastBlockHash = ancestorHash

    // process the new chain on top
    // we've already done this
    // but we're also paranoid
    _, err := self.TestChain(bchain)
    if err != nil{
        chainlogger.Infoln("Reorg failed as new chain failed processing. This shouldn't have happened and may mean trouble")
        self.CurrentBlock = oldHead
        self.LastBlockHash = oldHeadHash
        return
    }
    self.InsertChain(bchain)

    // move old canonical into workingTree chain
	bchain = &BlockChain{list.New()}
    for b := oldHead; bytes.Compare(b.Hash(), ancestorHash) != 0; b = self.GetBlock(b.PrevHash){
        bchain.PushFront(&link{b, nil, nil, nil})
        // TODO: remove from database
    }

    // again, we have already processed, since its fucking canonical
    // but this is easy for now, gives an extra check
    _, err = self.TestChain(bchain)
    if err != nil{
        chainlogger.Infoln("Adding the old canonical chain to the workingTree failed. This shouldn't happen, and may imply that Jesus has returned")
        return
    }
    self.InsertChain(bchain)
    
}





