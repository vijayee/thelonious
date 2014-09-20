package ethchain

import (
	"bytes"
	"math/big"
    "os"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethvm"
    "fmt"
)

var chainlogger = ethlog.NewLogger("CHAIN")

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
	bc.genesisBlock = NewBlockFromBytes(ethutil.Encode(Genesis))
	bc.Ethereum = ethereum

	bc.setLastBlock()

	return bc
}

func (bc *BlockChain) Genesis() *Block {
	return bc.genesisBlock
}

func (bc *BlockChain) NewBlock(coinbase []byte) *Block {
	var root interface{}
	var lastBlockTime int64
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.state.Trie.Root
		hash = bc.LastBlockHash
		lastBlockTime = bc.CurrentBlock.Time
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
//		ethutil.BigPow(2, 32),
		ethutil.BigPow(2, 12),
		nil,
		"")

	block.MinGasPrice = big.NewInt(10000000000000)

	parent := bc.CurrentBlock
	if parent != nil {
		diff := new(big.Int)

		adjust := new(big.Int).Rsh(parent.Difficulty, 10)
		if block.Time >= lastBlockTime+5 {
			diff.Sub(parent.Difficulty, adjust)
		} else {
			diff.Add(parent.Difficulty, adjust)
		}
		block.Difficulty = ethutil.BigPow(2, 17) //diff
		block.Number = new(big.Int).Add(bc.CurrentBlock.Number, ethutil.Big1)
		block.GasLimit = ethutil.BigPow(10, 50) //block.CalcGasLimit(bc.CurrentBlock)

	}

	return block
}

func (bc *BlockChain) HasBlock(hash []byte) bool {
	data, _ := ethutil.Config.Db.Get(hash)
	return len(data) != 0
}

// TODO: At one point we might want to save a block by prevHash in the db to optimise this...
func (bc *BlockChain) HasBlockWithPrevHash(hash []byte) bool {
	block := bc.CurrentBlock

	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(hash, block.PrevHash) == 0 {
			return true
		}
	}
	return false
}

func (bc *BlockChain) CalculateBlockTD(block *Block) *big.Int {
	blockDiff := new(big.Int)

	for _, uncle := range block.Uncles {
		blockDiff = blockDiff.Add(blockDiff, uncle.Difficulty)
	}
	blockDiff = blockDiff.Add(blockDiff, block.Difficulty)

	return blockDiff
}

func (bc *BlockChain) GenesisBlock() *Block {
	return bc.genesisBlock
}

func (self *BlockChain) GetChainHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		chain = append(chain, block.Hash())

		if block.Number.Cmp(ethutil.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.PrevHash)
	}

	return
}


func AddTestNetFunds(block *Block, eth EthManager){
	for _, addr := range []string{
        "bbbd0256041f7aed3ce278c56ee61492de96d001",
        "b9398794cafb108622b07d9a01ecbed3857592d5",
	} {
		codedAddr := ethutil.Hex2Bytes(addr)
		account := block.state.GetAccount(codedAddr)
		account.Balance = ethutil.Big("1606938044258990275541962092341162602522202993782792835301376") //ethutil.BigPow(2, 200)
		block.state.UpdateStateObject(account)
	}

    GenesisTxs(block, eth)
}

// doesn't allow code, no vm
func InsertSimpleContract(block *Block){
    addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
    fmt.Println("the genesis doug!!!", addr)
    state := block.state
    contract := state.NewStateObject(addr)
    contract.InitCode = ethutil.Hex2Bytes("...")
    contract.Code = ethutil.Hex2Bytes("600060005460006020023560055700")
    contract.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))
    amount := ethutil.Big("12345678987654345678")
    contract.Balance = amount
    storage := contract.State.Trie
    storage.Update("5", "a")

    block.state.UpdateStateObject(contract)

}

// add contract to genesis block - uses vm
func GenesisTxs(block *Block, eth EthManager){
   
    // mutan contract 
    s := `contract.storage[10] = 12
          return compile { 
           contract.storage[5] = this.data[0]   
          }`

    script, err := ethutil.Compile(s, false)
    if err != nil{
        fmt.Println("failed compile", err)
        os.Exit(0)
    }

    // dummy keys for signing
    keys := ethcrypto.GenerateNewKeyPair() 

    // create tx
    tx := NewContractCreationTx(ethutil.Big("543"), ethutil.Big("10000"), ethutil.Big("10000"), script)
    tx.Sign(keys.PrivateKey)

    txs := Transactions{tx}
    receipts := []*Receipt{}
   
    // new state transition 
    state := block.State()
    st := NewStateTransition(ethstate.NewStateObject(block.Coinbase), tx, state, block)
    st.AddGas(ethutil.Big("1000000")) // gas is silly, but the vm needs it
    // a slimmed down st.TransitionState() : 
        addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
        fmt.Println("man oh man", ethutil.Bytes2Hex(addr))
        receiver := state.NewStateObject(addr)
        receiver.InitCode = tx.Data
        receiver.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))
        sender := state.GetOrNewStateObject(tx.Sender())  
        value := ethutil.Big("12342")

        msg := state.Manifest().AddMessage(&ethstate.Message{
            To: receiver.Address(), From: sender.Address(),
            Input:  tx.Data,
            Origin: sender.Address(),
            Block:  block.Hash(), Timestamp: block.Time, Coinbase: block.Coinbase, Number: block.Number,
            Value: value,
        })
        code, err := st.Eval(msg, receiver.Init(), receiver, "init")
        fmt.Println("Eval error:", err)
        receiver.Code = code
        msg.Output = code
    //end TransitionState
    state.Update()
    receipt := &Receipt{tx, ethutil.CopyBytes(state.Root().([]byte)), new(big.Int)}
    receipts = append(receipts, receipt)
    block.SetReceipts(receipts, txs)
    state.Update()  
}


// lame old attempt ...
func InitContract(addr, script []byte, block *Block){

    tx := NewContractCreationTx(ethutil.Big("543"), ethutil.Big("0"), ethutil.Big("0"), script)
    crecv := []byte("00....")
    state := block.state
    contract := state.NewStateObject(addr)
    contract.InitCode = ethutil.Hex2Bytes("...")
    contract.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))
    amount := ethutil.Big("12345678987654345678")
    contract.Balance = amount

    fmt.Println("contract:", contract)

    msg := &ethstate.Message{
        To: crecv, From: []byte(""),
        Input: script,
        Origin: []byte(""),
        Block: block.Hash(),
        Timestamp: block.Time,
        Coinbase: block.Coinbase,
        Number: block.Number,
        Value: amount,
    }

    transactor := contract
    env := NewEnv(state, tx, block)
    callerClosure := ethvm.NewClosure(msg, transactor, contract, script, ethutil.Big("0"), ethutil.Big("0"))

	vm := ethvm.New(env)
	vm.Verbose = true
	vm.Fn = "init"

	ret, _, err := callerClosure.Call(vm, tx.Data)

	if err == nil {
		// Execute POSTs
		for e := vm.Queue().Front(); e != nil; e = e.Next() {
			msg := e.Value.(*ethvm.Message)

			msg.Exec(msg.Addr(), transactor)
		}
	}
    fmt.Println(ret)
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
		codedAddr := ethutil.Hex2Bytes(addr)
		account := block.state.GetAccount(codedAddr)
		account.Balance = ethutil.Big("1606938044258990275541962092341162602522202993782792835301376") //ethutil.BigPow(2, 200)
		block.state.UpdateStateObject(account)
	}
}
*/

func (bc *BlockChain) Fuck(thing string){
    fmt.Println(thing)
    addr := ethcrypto.Sha3Bin([]byte("the genesis doug"))
    block := bc.CurrentBlock
    st := block.state
    c := st.GetAccount(addr)
    fmt.Println("bal:", c.Balance)
    store := c.State.Trie
    it := store.NewIterator()
    it.Each(func(key string, value *ethutil.Value) {
        fmt.Println(ethutil.Bytes2Hex([]byte(key)), value)
    })

}

func (bc *BlockChain) setLastBlock() {
	// Prep genesis
	AddTestNetFunds(bc.genesisBlock, bc.Ethereum)

	data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = block.Number.Uint64()

	} else {
		bc.genesisBlock.state.Trie.Sync()
		// Prepare the genesis block
		bc.Add(bc.genesisBlock)
		fk := append([]byte("bloom"), bc.genesisBlock.Hash()...)
		bc.Ethereum.Db().Put(fk, make([]byte, 255))
	}

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())

	chainlogger.Infof("Last block (#%d) %x\n", bc.LastBlockNumber, bc.CurrentBlock.Hash())

}

func (bc *BlockChain) SetTotalDifficulty(td *big.Int) {
	ethutil.Config.Db.Put([]byte("LTD"), td.Bytes())
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

	return NewBlockFromBytes(data)
}

func (self *BlockChain) GetBlockByNumber(num uint64) *Block {
	block := self.CurrentBlock
	for ; block.Number.Uint64() != num; block = self.GetBlock(block.PrevHash) {
	}

	if block.Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
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
		chainlogger.Infoln("Stopped")
	}
}
