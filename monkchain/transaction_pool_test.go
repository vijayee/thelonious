package monkchain

import (
	"container/list"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkwire"
	"testing"
)

func newChainManager2() *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = NewBlockFromBytes(monkutil.Encode(Genesis))
	bc.Reset()
	return bc
}

type fakeEth2 struct{}

func (e *fakeEth2) BlockManager() *BlockManager                            { return nil }
func (e *fakeEth2) ChainManager() *ChainManager                            { return newChainManager2() }
func (e *fakeEth2) TxPool() *TxPool                                        { return &TxPool{} }
func (e *fakeEth2) Broadcast(msgType monkwire.MsgType, data []interface{}) {}
func (e *fakeEth2) Reactor() *monkreact.ReactorEngine                      { return monkreact.New() }
func (e *fakeEth2) PeerCount() int                                         { return 0 }
func (e *fakeEth2) IsMining() bool                                         { return false }
func (e *fakeEth2) IsListening() bool                                      { return false }
func (e *fakeEth2) Peers() *list.List                                      { return nil }
func (e *fakeEth2) KeyManager() *monkcrypto.KeyManager                     { return nil }
func (e *fakeEth2) ClientIdentity() monkwire.ClientIdentity                { return nil }
func (e *fakeEth2) Db() monkutil.Database                                  { return nil }
func (e *fakeEth2) GenesisPointer(block *Block)                            {}
func (e *fakeEth2) GenesisModel() GenDougModel                             { return nil }

func TestValidateTransaction(t *testing.T) {
	txs := map[string]string{
		"badsig":   "f87180881bc16d674ec80000881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d008061a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566",
		"gib":      "10fa5",
		"gib2":     "10fa56a89200b3c",
		"gib3":     "f971808813c16d67555ec800881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d008061a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566a06162636465666768696a6b6c6d6e6f707172737475767778797a6162636465",
		"empty":    "",
		"good":     "f87180881bc16d674ec80000881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d00801ba0f65d4f7cb2b546719799c5e91baa99da7b37e0f75b2ab23e58ade0797dde9a83a0606f21ab49acc9e6deb39b67c856ac0814b995fc374be6d06d312b72d9fc9a98",
		"missing1": "f87180881bc16d674ec80000881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d00801ba0f65d4f7cb2b546719799c5e91baa99da7b37e0f75b2ab23e58ade0797dde9a83a0606f21ab49acc9e6deb39b67c856ac0814b995fc374be6d06d312b72d9fc9a",
	}

	pool := NewTxPool(new(fakeEth2))

	f := func(s string, iserr bool) {
		tx := NewTransactionFromBytes(monkutil.Hex2Bytes(s))
		err := pool.ValidateTransaction(tx)
		if iserr {
			if err == nil {
				t.Error("Expected an error")
			}
		} else {
			if err != nil {
				t.Error("Did not expect an error")
			}
		}
	}

	// proper tx with a made up signature
	f(txs["badsig"], true)
	// made up giberish (bad rlp)
	f(txs["gib"], true)
	// made up giberish (bad rlp)
	f(txs["gib2"], true)
	// slightly modified badsig (bad rlp)
	// causes a panic on the rlp decoder
	//f(txs["gib3"], true)
	// an empty string
	f(txs["empty"], true)
	// a well formed sig
	// TODO: better mock BlockManager so we can have funds in this account for a valid tx
	// f(txs["good"], false)
	// a well formed sig missing the final byte
	//f(txs["missing1"], true)

}
