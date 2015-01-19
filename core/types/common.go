package types

import (
	"math/big"

	"github.com/eris-ltd/new-thelonious/state"
)

type BlockProcessor interface {
	Process(*Block) (*big.Int, state.Messages, error)
}
