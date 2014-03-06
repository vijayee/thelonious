package ethchain

import (
	"bytes"
	"github.com/ethereum/eth-go/ethutil"
	"log"
	"math"
	"math/big"
)

type BlockChain struct {
	Ethereum EthManager
	// The famous, the fabulous Mister GENESIIIIIIS (block)
	genesisBlock *Block
	// Last known total difficulty
	TD *big.Int

	LastBlockNumber uint64

	CurrentBlock  *Block
	LastBlockHash []byte
}

func NewBlockChain(ethereum EthManager) *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = NewBlockFromData(ethutil.Encode(Genesis))

	bc.setLastBlock()

	return bc
}

func (bc *BlockChain) Genesis() *Block {
	return bc.genesisBlock
}

func (bc *BlockChain) NewBlock(coinbase []byte, txs []*Transaction) *Block {
	var root interface{}
	var lastBlockTime int64
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.state.trie.Root
		hash = bc.LastBlockHash
		lastBlockTime = bc.CurrentBlock.Time
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
		ethutil.BigPow(2, 32),
		nil,
		"",
		txs)

	if bc.CurrentBlock != nil {
		var mul *big.Int
		if block.Time < lastBlockTime+42 {
			mul = big.NewInt(1)
		} else {
			mul = big.NewInt(-1)
		}

		diff := new(big.Int)
		diff.Add(diff, bc.CurrentBlock.Difficulty)
		diff.Div(diff, big.NewInt(1024))
		diff.Mul(diff, mul)
		diff.Add(diff, bc.CurrentBlock.Difficulty)
		block.Difficulty = diff
	}

	return block
}

func (bc *BlockChain) HasBlock(hash []byte) bool {
	data, _ := ethutil.Config.Db.Get(hash)
	return len(data) != 0
}

func (bc *BlockChain) GenesisBlock() *Block {
	return bc.genesisBlock
}

// Get chain return blocks from hash up to max in RLP format
func (bc *BlockChain) GetChainFromHash(hash []byte, max uint64) []interface{} {
	var chain []interface{}
	// Get the current hash to start with
	currentHash := bc.CurrentBlock.Hash()
	// Get the last number on the block chain
	lastNumber := bc.BlockInfo(bc.CurrentBlock).Number
	// Get the parents number
	parentNumber := bc.BlockInfoByHash(hash).Number
	// Get the min amount. We might not have max amount of blocks
	count := uint64(math.Min(float64(lastNumber-parentNumber), float64(max)))
	startNumber := parentNumber + count

	num := lastNumber
	for ; num > startNumber; currentHash = bc.GetBlock(currentHash).PrevHash {
		num--
	}
	for i := uint64(0); bytes.Compare(currentHash, hash) != 0 && num >= parentNumber && i < count; i++ {
		// Get the block of the chain
		block := bc.GetBlock(currentHash)
		currentHash = block.PrevHash

		chain = append(chain, block.Value().Val)

		num--
	}

	return chain
}

func (bc *BlockChain) GetChain(hash []byte, amount int) []*Block {
	genHash := bc.genesisBlock.Hash()

	block := bc.GetBlock(hash)
	var blocks []*Block

	for i := 0; i < amount && block != nil; block = bc.GetBlock(block.PrevHash) {
		blocks = append([]*Block{block}, blocks...)

		if bytes.Compare(genHash, block.Hash()) == 0 {
			break
		}
		i++
	}

	return blocks
}

func AddTestNetFunds(block *Block) {
	for _, addr := range []string{
		"8a40bfaa73256b60764c1bf40675a99083efb075", // Gavin
		"e6716f9544a56c530d868e4bfbacb172315bdead", // Jeffrey
		"1e12515ce3e0f817a4ddef9ca55788a1d66bd2df", // Vit
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4", // Alex
	} {
		//log.Println("2^200 Wei to", addr)
		codedAddr := ethutil.FromHex(addr)
		addr := block.state.GetAccount(codedAddr)
		addr.Amount = ethutil.BigPow(2, 200)
		block.state.UpdateAccount(codedAddr, addr)
	}
}

func (bc *BlockChain) setLastBlock() {
	data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		info := bc.BlockInfo(block)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = info.Number

		log.Printf("[CHAIN] Last known block height #%d\n", bc.LastBlockNumber)
	} else {
		AddTestNetFunds(bc.genesisBlock)

		bc.genesisBlock.state.trie.Sync()
		// Prepare the genesis block
		bc.Add(bc.genesisBlock)

		//log.Printf("root %x\n", bm.bc.genesisBlock.State().Root)
		//bm.bc.genesisBlock.PrintHash()
	}

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())

	log.Printf("Last block: %x\n", bc.CurrentBlock.Hash())
}

func (bc *BlockChain) SetTotalDifficulty(td *big.Int) {
	ethutil.Config.Db.Put([]byte("LastKnownTotalDifficulty"), td.Bytes())
	bc.TD = td
}

// Add a block to the chain and record addition information
func (bc *BlockChain) Add(block *Block) {
	bc.writeBlockInfo(block)

	// Prepare the genesis block
	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	ethutil.Config.Db.Put(block.Hash(), encodedBlock)
	ethutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)
}

func (bc *BlockChain) GetBlock(hash []byte) *Block {
	data, _ := ethutil.Config.Db.Get(hash)
	if len(data) == 0 {
		return nil
	}

	return NewBlockFromData(data)
}

func (bc *BlockChain) BlockInfoByHash(hash []byte) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(hash, []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (bc *BlockChain) BlockInfo(block *Block) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
func (bc *BlockChain) writeBlockInfo(block *Block) {
	bc.LastBlockNumber++
	bi := BlockInfo{Number: bc.LastBlockNumber, Hash: block.Hash(), Parent: block.PrevHash}

	// For now we use the block hash with the words "info" appended as key
	ethutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bc *BlockChain) Stop() {
	if bc.CurrentBlock != nil {
		log.Println("[CHAIN] Stopped")
	}
}