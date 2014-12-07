package monk

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/eris-ltd/decerver-interfaces/glue/genblock"
	"github.com/eris-ltd/decerver-interfaces/glue/utils"
	"github.com/eris-ltd/epm-go"

	eth "github.com/eris-ltd/thelonious"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkminer"
	"github.com/eris-ltd/thelonious/monkpipe"
	"github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/thelonious/monkutil"
)

// this is basically go-etheruem/utils

// TODO: use the interupts...

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

// TODO: dwell on this more too
func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *monkutil.ConfigManager {
	utils.InitDataDir(Datadir)
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

func ShowGenesis(ethereum *eth.Thelonious) {
	logger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

// TODO: work this baby
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

// TODO: use this...
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

// Set the EPM contract root
func setContractPath(p string) {
	epm.ContractPath = p
}

// Deploy a pdx onto a block
// This is used as a monkdoug deploy function
func epmDeploy(block *monkchain.Block, pkgDef string) ([]byte, error) {
	m := genblock.NewGenBlockModule(block)
	m.Config.LogLevel = 5
	m.Init()
	m.Start()
	e := epm.NewEPM(m, ".epm-log")
	err := e.Parse(pkgDef)
	if err != nil {
		return nil, err
	}
	epm.ErrMode = epm.ReturnOnErr
	err = e.ExecuteJobs()
	if err != nil {
		return nil, err
	}
	e.Commit()
	chainId, err := m.ChainId()
	if err != nil {
		return nil, err
	}
	return chainId, nil
}

// Deploy sequence (done through monk interface for simplicity):
//  - Create .temp/ for database in current dir
//  - Read genesis.json and populate struct
//  - Deploy genesis block and return chainId
//  - Move .temp/ into ~/.decerver/blockchain/thelonious/chainID
//  - write name to index file if provided and no conflict
func DeploySequence(name, genesis, config string) {
	root := ".temp"
	chainId := DeployChain(root, genesis, config)
	InstallChain(root, name, genesis, config, chainId)
	exit(nil)
}

func DeployChain(root, genesis, config string) string {
	// startup and deploy
	m := NewMonk(nil)
	m.ReadConfig(config)
	m.Config.RootDir = root
	m.Config.GenesisConfig = genesis
	m.Init()
	// get the chain id
	data, err := monkutil.Config.Db.Get([]byte("ChainID"))
	if err != nil {
		exit(err)
	} else if len(data) == 0 {
		exit(fmt.Errorf("ChainID is empty!"))
	}
	chainId := monkutil.Bytes2Hex(data)
	return chainId
}

func InstallChain(root, name, genesis, config, chainId string) {
	// move datastore
	utils.InitDataDir(path.Join(Thelonious, chainId, "datastore"))
	rename(root, path.Join(Thelonious, chainId, "datastore"))
	copy(config, path.Join(Thelonious, chainId, "config.json"))
	copy(genesis, path.Join(Thelonious, chainId, "genesis.json"))

	// update refs
	if name != "" {
		err := utils.NewChainRef(name, chainId)
		if err != nil {
			exit(err)
		}
		logger.Infof("Created ref %s to point to chain %s\n", name, chainId)
	}
}

func rename(oldpath, newpath string) {
	err := os.Rename(oldpath, newpath)
	if err != nil {
		exit(err)
	}
}

func copy(oldpath, newpath string) {
	err := utils.Copy(oldpath, newpath)
	if err != nil {
		exit(err)
	}
}
