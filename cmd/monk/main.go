package main

import (
	"flag"
	"fmt"
	"github.com/eris-ltd/thelonious/monk"
	"path"
)

var (
	listenHost = flag.String("listen-host", "0.0.0.0", "Set listen ip address")
	listenPort = flag.Int("listen-port", 30303, "Set tcp listen port")
	listen     = flag.Bool("listen", true, "Listen for incoming connections")
	remoteHost = flag.String("remote-host", "", "Peer server ip address")
	remotePort = flag.Int("remote-port", 30303, "Peer server port")
	useSeed    = flag.Bool("use-seed", false, "Bootstrap p2p through seed node")
	rpcHost    = flag.String("rpc-host", "", "Set rpc host ip address")
	rpcPort    = flag.Int("rpc-port", 30304, "Set rpc host port")
	serveRpc   = flag.Bool("serve-rpc", false, "Run the rpc server")

	chainId   = flag.String("chainId", "", "Select chain by chainId")
	chainName = flag.String("name", "", "Select chain by name")

	mining           = flag.Bool("mine", false, "Turn mining on")
	maxPeers         = flag.Int("max-peers", 10, "Maximum number of peer connections")
	clientId         = flag.String("clientId", "TheloniousMonk", "P2P client id")
	version          = flag.String("version", "0.7.0", "P2P version")
	identifier       = flag.String("identifier", "", "Custom client identifier")
	keySession       = flag.String("key-session", "generous", "Set key session")
	keyStore         = flag.String("key-store", "file", "Load keys from file or db")
	keyCursor        = flag.Int("key-cursor", 0, "Select which key to use")
	keyFile          = flag.String("key-file", monk.DefaultKeyFile, "File to load keys from")
	adversary        = flag.Int("adversary", 0, "Set node to be adversarial")
	useCheckpoint    = flag.Bool("use-checkpoint", false, "Use a blockchain checkpoint")
	latestCheckpoint = flag.String("latest-checkpoint", "", "Set the latest checkpoint")

	configFile    = flag.String("config-file", "config", "What is this even?")
	rootDir       = flag.String("root-dir", "", "Set the root database directory")
	dbName        = flag.String("db-name", "database", "Set the name of the database folder")
	dbMem         = flag.Bool("db-mem", false, "Use a memory database instead of on-disk")
	contractPath  = flag.String("contract-path", path.Join(monk.ErisLtd, "eris-std-lib"), "Set the contract path")
	genesisConfig = flag.String("genesis-config", monk.DefaultGenesisConfig, "Set the genesis config file")

	lllPath   = flag.String("lll-path", monk.DefaultLLLPath, "Set the path to the lll compiler")
	lllServer = flag.String("lll-server", monk.DefaultLLLServer, "Set the url to an lll compile server")
	lllLocal  = flag.Bool("lll-local", false, "Use the local lll compiler or the server")

	logFile   = flag.String("log-file", "", "Set the log file")
	debugFile = flag.String("debug-file", "", "Set the debug file")
	logLevel  = flag.Int("log-level", 5, "Set the logger level")
)

func main() {
	flag.Parse()

	m := monk.NewMonk(nil)

	m.Config.ListenHost = *listenHost
	m.Config.ListenPort = *listenPort
	m.Config.Listen = *listen
	m.Config.RemoteHost = *remoteHost
	m.Config.RemotePort = *remotePort
	m.Config.UseSeed = *useSeed
	m.Config.RpcHost = *rpcHost
	m.Config.RpcPort = *rpcPort
	m.Config.ServeRpc = *serveRpc

	m.Config.ChainId = *chainId
	m.Config.ChainName = *chainName

	m.Config.Mining = *mining
	m.Config.MaxPeers = *maxPeers
	m.Config.ClientIdentifier = *clientId
	m.Config.Version = *version
	m.Config.Identifier = *identifier
	m.Config.KeySession = *keySession
	m.Config.KeyStore = *keyStore
	m.Config.KeyCursor = *keyCursor
	m.Config.KeyFile = *keyFile
	m.Config.Adversary = *adversary
	m.Config.UseCheckpoint = *useCheckpoint
	m.Config.LatestCheckpoint = *latestCheckpoint

	m.Config.ConfigFile = *configFile
	m.Config.RootDir = *rootDir
	m.Config.DbName = *dbName
	m.Config.DbMem = *dbMem
	m.Config.ContractPath = *contractPath
	m.Config.GenesisConfig = *genesisConfig

	m.Config.LLLPath = *lllPath
	m.Config.LLLServer = *lllServer
	m.Config.LLLLocal = *lllLocal

	m.Config.LogFile = *logFile
	m.Config.DebugFile = *debugFile
	m.Config.LogLevel = *logLevel

	fmt.Println(m.Config.Mining)

	m.Init()
	m.Start()

	for {
		select {}
	}

}
