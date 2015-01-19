package vm

import (
	"math/big"

	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/monkutil"
)

type Address interface {
	Call(in []byte) []byte
}

type PrecompiledAccount struct {
	Gas func(l int) *big.Int
	fn  func(in []byte) []byte
}

func (self PrecompiledAccount) Call(in []byte) []byte {
	return self.fn(in)
}

var Precompiled = PrecompiledContracts()

// XXX Could set directly. Testing requires resetting and setting of pre compiled contracts.
func PrecompiledContracts() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// ECRECOVER
		string(monkutil.LeftPadBytes([]byte{1}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			return GasEcrecover
		}, ecrecoverFunc},

		// SHA256
		string(monkutil.LeftPadBytes([]byte{2}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31)/32 + 1)
			n.Mul(n, GasSha256)
			return n
		}, sha256Func},

		// RIPEMD160
		string(monkutil.LeftPadBytes([]byte{3}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31)/32 + 1)
			n.Mul(n, GasRipemd)
			return n
		}, ripemd160Func},

		string(monkutil.LeftPadBytes([]byte{4}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31)/32 + 1)
			n.Mul(n, GasMemCpy)

			return n
		}, memCpy},
	}
}

func sha256Func(in []byte) []byte {
	return crypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return monkutil.LeftPadBytes(crypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	// In case of an invalid sig. Defaults to return nil
	defer func() { recover() }()

	hash := in[:32]
	v := monkutil.BigD(in[32:64]).Bytes()[0] - 27
	sig := append(in[64:], v)

	return monkutil.LeftPadBytes(crypto.Sha3(crypto.Ecrecover(append(hash, sig...))[1:])[12:], 32)
}

func memCpy(in []byte) []byte {
	return in
}
