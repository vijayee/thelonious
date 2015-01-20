package monkvm

import (
	monklog "github.com/eris-ltd/new-thelonious/logger"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"math/big"
)

var vmlogger = monklog.NewLogger("VM")

var (
	GasStep    = big.NewInt(1)
	GasSha     = big.NewInt(20)
	GasSLoad   = big.NewInt(20)
	GasSStore  = big.NewInt(100)
	GasBalance = big.NewInt(20)
	GasNonce   = big.NewInt(20)
	GasCreate  = big.NewInt(100)
	GasCall    = big.NewInt(20)
	GasMemory  = big.NewInt(1)
	GasData    = big.NewInt(5)
	GasTx      = big.NewInt(500)

	Pow256 = thelutil.BigPow(2, 256)

	LogTyPretty byte = 0x1
	LogTyDiff   byte = 0x2
)
