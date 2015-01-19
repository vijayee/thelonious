package helper

import (
	"log"
	"os"

	"github.com/eris-ltd/new-thelonious/thelutil"
	logpkg "github.com/eris-ltd/new-thelonious/logger"
)

var Logger logpkg.LogSystem
var Log = logpkg.NewLogger("TEST")

func init() {
	Logger = logpkg.NewStdLogSystem(os.Stdout, log.LstdFlags, logpkg.InfoLevel)
	logpkg.AddLogSystem(Logger)

	thelutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
}
