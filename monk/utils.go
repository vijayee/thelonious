package monk

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"bitbucket.org/kardianos/osext"
	eth "github.com/eris-ltd/thelonious"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkminer"
	"github.com/eris-ltd/thelonious/monkpipe"
	"github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkwire"
)

// this is basically go-etheruem/utils

// i think for now we only use StartMining, but there's porbably other goodies...

//var logger = monklog.NewLogger("CLI")
var interruptCallbacks = []func(os.Signal){}

// Register interrupt handlers callbacks
func RegisterInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

// go routine that call interrupt handlers in order of registering
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			logger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func openLogFile(Datadir string, filename string) *os.File {
	path := AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

func InitDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
		}
	}
}

func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}
	monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.LogLevel(LogLevel)))
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.DebugLevel))
	}
}

func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *monkutil.ConfigManager {
	InitDataDir(Datadir)
	return monkutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		logger.Errorln("Fatal: ", err)
		status = 1
	}
	monklog.Flush()
	os.Exit(status)
}

func NewDatabase(dbName string) monkutil.Database {
	db, err := monkdb.NewLDBDatabase(dbName)
	if err != nil {
		exit(err)
	}
	return db
}

func NewClientIdentity(clientIdentifier, version, customIdentifier string) *monkwire.SimpleClientIdentity {
	logger.Infoln("identity created")
	return monkwire.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
}

/*
func NewThelonious(db monkutil.Database, clientIdentity monkwire.ClientIdentity, keyManager *monkcrypto.KeyManager, usePnp bool, OutboundPort string, MaxPeer int) *eth.Thelonious {
	ethereum, err := eth.New(db, clientIdentity, keyManager, eth.CapDefault, usePnp)
	if err != nil {
		logger.Fatalln("eth start err:", err)
	}
	ethereum.Port = OutboundPort
	ethereum.MaxPeers = MaxPeer
	return ethereum
}*/

/*
func StartThelonious(ethereum *eth.Thelonious, UseSeed bool) {
	logger.Infof("Starting %s", ethereum.ClientIdentity())
	ethereum.Start(UseSeed)
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		monklog.Flush()
	})
}*/

func ShowGenesis(ethereum *eth.Thelonious) {
	logger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

func NewKeyManager(KeyStore string, Datadir string, db monkutil.Database) *monkcrypto.KeyManager {
	var keyManager *monkcrypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = monkcrypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = monkcrypto.NewFileKeyManager(Datadir)
	default:
		exit(fmt.Errorf("unknown keystore type: %s", KeyStore))
	}
	return keyManager
}

func DefaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/ethereal"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
}

func KeyTasks(keyManager *monkcrypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {

	var err error
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.Init(KeyRing, 0, true)
		}
		exit(err)
	case len(SecretFile) > 0:
		SecretFile = monkutil.ExpandHomePath(SecretFile)

		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.InitFromSecretsFile(KeyRing, 0, SecretFile)
		}
		exit(err)
	case len(ExportDir) > 0:
		err = keyManager.Init(KeyRing, 0, false)
		if err == nil {
			err = keyManager.Export(ExportDir)
		}
		exit(err)
	default:
		// Creates a keypair if none exists
		err = keyManager.Init(KeyRing, 0, false)
		if err != nil {
			exit(err)
		}
	}
}

func StartRpc(ethereum *eth.Thelonious, RpcHost string, RpcPort int) {
	var err error
	rpcAddr := RpcHost + ":" + strconv.Itoa(RpcPort)
	ethereum.RpcServer, err = monkrpc.NewJsonRpcServer(monkpipe.NewJSPipe(ethereum), rpcAddr)
	if err != nil {
		logger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

var miner *monkminer.Miner

func GetMiner() *monkminer.Miner {
	return miner
}

func StartMining(ethereum *eth.Thelonious) bool {

	if !ethereum.Mining {
		ethereum.Mining = true
		addr := ethereum.KeyManager().Address()

		go func() {
			logger.Infoln("Start mining")
			if miner == nil {
				miner = monkminer.NewDefaultMiner(addr, ethereum)
			}
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			for !ethereum.IsUpToDate() {
				time.Sleep(5 * time.Second)
			}
			miner.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
}

func FormatTransactionData(data string) []byte {
	d := monkutil.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := monkutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

func StopMining(ethereum *eth.Thelonious) bool {
	if ethereum.Mining && miner != nil {
		miner.Stop()
		logger.Infoln("Stopped mining")
		ethereum.Mining = false
		miner = nil
		return true
	}

	return false
}

// Replay block
func BlockDo(ethereum *eth.Thelonious, hash []byte) error {
	block := ethereum.ChainManager().GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := ethereum.ChainManager().GetBlock(block.PrevHash)

	_, err := ethereum.BlockManager().ApplyDiff(parent.State(), parent, block)
	if err != nil {
		return err
	}

	return nil

}

// If an address is empty, load er up
// vestige of ye old key days
func CheckZeroBalance(pipe *monkpipe.Pipe, keyMang *monkcrypto.KeyManager) {
	keys := keyMang.KeyRing()
	masterPair := keys.GetKeyPair(0)
	logger.Infoln("master has ", pipe.Balance(keys.GetKeyPair(keys.Len()-1).Address()))
	for i := 0; i < keys.Len(); i++ {
		k := keys.GetKeyPair(i).Address()
		val := pipe.Balance(k)
		logger.Infoln("key ", i, " ", monkutil.Bytes2Hex(k), " ", val)
		v := val.Int()
		if v < 100 {
			_, err := pipe.Transact(masterPair, k, monkutil.NewValue(monkutil.Big("10000000000000000000")), monkutil.NewValue(monkutil.Big("1000")), monkutil.NewValue(monkutil.Big("1000")), "")
			if err != nil {
				logger.Infoln("Error transfering funds to ", monkutil.Bytes2Hex(k))
			}
		}
	}
}
