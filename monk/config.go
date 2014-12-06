package monk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/thelonious/monkutil"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

var ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")

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
	LLLPath       string `json:"lll_path"`
	ContractPath  string `json:"contract_path"`
	GenesisConfig string `json:"genesis_config"`

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

	// Local Node
	Mining:           false,
	MaxPeers:         10,
	ClientIdentifier: "Thelonious(decerver)",
	Version:          "0.5.17",
	Identifier:       "chainId",
	KeySession:       "generous",
	KeyStore:         "file",
	KeyCursor:        0,
	KeyFile:          path.Join(ErisLtd, "thelonious", "monk", "keys.txt"),
	Adversary:        0,
	UseCheckpoint:    false,
	LatestCheckpoint: "",

	// Paths
	ConfigFile:    "config",
	RootDir:       path.Join(usr.HomeDir, ".monkchain2"),
	DbName:        "database",
	LLLPath:       "NETCALL", //path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
	ContractPath:  path.Join(ErisLtd, "eris-std-lib"),
	GenesisConfig: path.Join(ErisLtd, "thelonious", "monk", "genesis-std.json"),

	// Log
	LogFile:   "",
	DebugFile: "",
	LogLevel:  5,
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
		fmt.Println("could not read config", err)
		fmt.Println("resorting to defaults")
		mod.WriteConfig(config_file)
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
func (mod *MonkModule) SetConfig(field string, value interface{}) error {
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
	if cfg.LLLPath != "" {
		monkutil.PathToLLL = cfg.LLLPath
	}

	// check on data dir
	// create keys
	_, err := os.Stat(cfg.RootDir)
	if err != nil {
		os.Mkdir(cfg.RootDir, 0777)
		_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
		if err != nil {
			Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
		}
	}
	// a global monkutil.Config object is used for shared global access to the db.
	// this also uses rakyl/globalconf, but we mostly ignore all that
	monkutil.Config = &monkutil.ConfigManager{ExecPath: cfg.RootDir, Debug: true, Paranoia: true}
	// TODO: enhance this with more pkg level control
	InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, cfg.DebugFile)
}

// Is there really no way to copy a file in the std lib?
func Copy(src, dst string) {
	r, err := os.Open(src)
	if err != nil {
		logger.Errorln(err)
		return
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		logger.Errorln(err)
		return
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		logger.Errorln(err)
		return
	}
}
