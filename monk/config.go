package monk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/decerver-interfaces/glue/utils"
	"github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/thelonious/monkutil"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"reflect"
)

var (
	GoPath     = os.Getenv("GOPATH")
	usr, _     = user.Current() // error?!
	ErisLtd    = utils.ErisLtd
	Decerver   = utils.Decerver
	Thelonious = path.Join(utils.Blockchains, "thelonious")

	DefaultRoot          = path.Join(Thelonious, "default-chain")
	DefaultGenesisConfig = path.Join(ErisLtd, "thelonious", "monk", "defaults", "genesis.json")
	DefaultKeyFile       = path.Join(ErisLtd, "thelonious", "monk", "defaults", "keys.txt")
	DefaultLLLPath       = path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc")
	DefaultLLLServer     = "http://lllc.erisindustries.com/compile"
)

type ChainConfig struct {
	// Networking
	ListenHost string `json:"local_host"`
	ListenPort int    `json:"local_port"`
	Listen     bool   `json:"listen"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
	UseSeed    bool   `json:"use_seed"`
	RpcHost    string `json:"rpc_host"`
	RpcPort    int    `json:"rpc_port"`
	ServeRpc   bool   `json:"serve_rpc"`

	// ChainId and Name
	ChainId   string `json:"chain_id"`
	ChainName string `json:"chain_name"`

	// Local Node
	Mining           bool   `json:"mining"`
	MaxPeers         int    `json:"max_peers"`
	ClientIdentifier string `json:"client"`
	Version          string `json:"version"`
	Identifier       string `json:"id"`
	KeySession       string `json:"key_session"`
	KeyStore         string `json:"key_store"`
	KeyCursor        int    `json:"key_cursor"`
	KeyFile          string `json:"key_file"`
	Adversary        int    `json:"adversary"`
	UseCheckpoint    bool   `json:"use_checkpoint"`
	LatestCheckpoint string `json:"latest_checkpoint"`

	// Paths
	ConfigFile    string `json:"config_file"`
	RootDir       string `json:"root_dir"`
	DbName        string `json:"db_name"`
	DbMem         bool   `json:"db_mem"`
	ContractPath  string `json:"contract_path"`
	GenesisConfig string `json:"genesis_config"`

	// Language Compilation
	LLLPath   string `json:"lll_path"`
	LLLServer string `json:"lll_server"`
	LLLLocal  bool   `json:"lll_local"`

	// Logs
	LogFile   string `json:"log_file"`
	DebugFile string `json:"debug_file"`
	LogLevel  int    `json:"log_level"`
}

// set default config object
var DefaultConfig = &ChainConfig{
	// Network
	ListenHost: "0.0.0.0",
	ListenPort: 30303,
	Listen:     true,
	RemoteHost: "",
	RemotePort: 30303,
	UseSeed:    false,
	RpcHost:    "",
	RpcPort:    30304,
	ServeRpc:   false,

	// ChainId and Name
	ChainId:   "",
	ChainName: "",

	// Local Node
	Mining:           false,
	MaxPeers:         10,
	ClientIdentifier: "Thelonious(decerver)",
	Version:          "0.7.0",
	Identifier:       "chainId",
	KeySession:       "generous",
	KeyStore:         "file",
	KeyCursor:        0,
	KeyFile:          DefaultKeyFile,
	Adversary:        0,
	UseCheckpoint:    false,
	LatestCheckpoint: "",

	// Paths
	ConfigFile:    "config",
	RootDir:       "",
	DbName:        "database",
	DbMem:         false,
	ContractPath:  path.Join(ErisLtd, "eris-std-lib"),
	GenesisConfig: DefaultGenesisConfig,

	// Language Compilation
	LLLPath:   DefaultLLLPath,
	LLLServer: DefaultLLLServer,
	LLLLocal:  false,

	// Log
	LogFile:   "",
	DebugFile: "",
	LogLevel:  5,
}

func InitChain(configPath string) error {
	err := utils.InitDecerverDir()
	if err != nil {
		return err
	}
	err = utils.InitDataDir(Thelonious)
	if err != nil {
		return err
	}
	err = utils.WriteJson(DefaultConfig, path.Join(configPath, "monk-config.json"))
	if err != nil {
		return err
	}
	return utils.WriteJson(monkdoug.DefaultGenesis, path.Join(configPath, "genesis.json"))
}

// Marshal the current configuration to file in pretty json.
func (mod *MonkModule) WriteConfig(config_file string) {
	b, err := json.Marshal(mod.monk.config)
	if err != nil {
		fmt.Println("error marshalling config:", err)
		return
	}
	var out bytes.Buffer
	json.Indent(&out, b, "", "\t")
	ioutil.WriteFile(config_file, out.Bytes(), 0600)
}

// Unmarshal the configuration file into module's config struct.
func (mod *MonkModule) ReadConfig(config_file string) {
	b, err := ioutil.ReadFile(config_file)
	if err != nil {
		logger.Errorln("Could not read config file", err)
		logger.Errorln("Did you run `monk -init`?")
		return
	}
	var config ChainConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		fmt.Println("error unmarshalling config from file:", err)
		fmt.Println("resorting to defaults")
		//mod.monk.config = DefaultConfig
		return
	}
	*(mod.Config) = config
}

// Set a field in the config struct.
func (mod *MonkModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mod.monk.config).Elem()
	f := cv.FieldByName(field)
	kind := f.Kind()

	k := reflect.ValueOf(value).Kind()
	if kind != k {
		return fmt.Errorf("Invalid kind. Expected %s, received %s", kind, k)
	}

	if kind == reflect.String {
		f.SetString(value.(string))
	} else if kind == reflect.Int {
		f.SetInt(int64(value.(int)))
	} else if kind == reflect.Bool {
		f.SetBool(value.(bool))
	}
	return nil
}

// Set the config object directly
func (mod *MonkModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mod.monk.config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Set package global variables (LLLPath, monkutil.Config, logging).
// Create the root data dir if it doesn't exist, and copy keys if they are available
func (monk *Monk) thConfig() {
	cfg := monk.config
	// set lll path
	if cfg.LLLLocal {
		if cfg.LLLPath != "" {
			monkutil.PathToLLL = cfg.LLLPath
		}
	} else {
		// TODO: set server address in monkutil...
		//  (drop NETCALL crap)
	}

	// check on data dir
	// create keys
	utils.InitDataDir(cfg.RootDir)
	_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
	if err != nil {
		utils.Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
	}
	// if the root dir is the default dir, make sure genesis.json's are available
	// TODO: handle this better
	_, err = os.Stat(path.Join(cfg.RootDir, "genesis.json"))
	fmt.Println(err)
	if err != nil {
		fmt.Println("copy!", DefaultGenesisConfig)
		utils.Copy(DefaultGenesisConfig, path.Join(cfg.RootDir, "genesis.json"))
	}

	// a global monkutil.Config object is used for shared global access to the db.
	// this also uses rakyl/globalconf, but we mostly ignore all that
	monkutil.Config = &monkutil.ConfigManager{ExecPath: cfg.RootDir, Debug: true, Paranoia: true}
	// TODO: enhance this with more pkg level control
	utils.InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, cfg.DebugFile)
}
