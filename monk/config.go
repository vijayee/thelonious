package monk

import (
    "errors"
    "fmt"
    "path"
    "bytes"
    "io/ioutil"
    "encoding/json"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkchain"
)

type ChainConfig struct{
    Port int        `json:"port"`
    Mining bool     `json:"mining"`
    MaxPeers int    `json:"max_peers"`
    ConfigFile string `json:"config_file"`
    RootDir string  `json:"root_dir"`
    KeyFile string  `json:"key_file"`
    Name string     `json:"name"`
    LogFile string  `json:"log_file"`
    DataDir string `json:"data_dir"`
    DbName string `json:"db_name"`
    LLLPath string `json:"lll_path"`
    ContractPath string `json:"contract_path"`
    ClientIdentifier string `json:"client"`
    Version string  `json:"version"`
    Identifier string `json:"id"`
    KeyStore string `json:"keystore"`
    GenesisConfig string `json:"genesis_config"`
    DougDifficulty int `json:"difficulty"`
    LogLevel int    `json:"log_level"`
}

// set default config object
var DefaultConfig = &ChainConfig{ 
        Port : 30303,
        Mining : false,
        MaxPeers : 10,
        ConfigFile : "config",
        RootDir : path.Join(usr.HomeDir, ".monkchain2"),
        DbName : "database",
        KeyFile : path.Join(GoPath, "src", "github.com", "eris-ltd", "thelonious", "monk", "keys.txt"),
        Name : "decerver-monkchain",
        LogFile: "",
        DataDir: path.Join(homeDir(), ".eris-eth"),
        //LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
        LLLPath: "NETCALL",
       // ContractPath: path.Join(GoPath, "src", "github.com", "eris-ltd", "thelonious", "monk", "contracts"),
        ContractPath: path.Join(GoPath, "src", "github.com", "eris-ltd", "eris-std-lib"),
        ClientIdentifier: "Ethereum(deCerver)",
        Version: "0.5.17",
        Identifier: "",
        KeyStore: "db",
        GenesisConfig: path.Join(GoPath, "src", "github.com", "eris-ltd", "thelonious", "monk", "genesis-std.json"),
        DougDifficulty: 7,
        LogLevel: 5,
}


// can these methods be functions in decerver that take the modules as argument?
func (mod *MonkModule) WriteConfig(config_file string){
    b, err := json.Marshal(mod.monk.Config)
    if err != nil{
        fmt.Println("error marshalling config:", err)
        return
    }
    var out bytes.Buffer
    json.Indent(&out, b, "", "\t")
    ioutil.WriteFile(config_file, out.Bytes(), 0600)
}
func (mod *MonkModule) ReadConfig(config_file string){
    b, err := ioutil.ReadFile(config_file)
    if err != nil{
        fmt.Println("could not read config", err)
        fmt.Println("resorting to defaults")
        mod.WriteConfig(config_file)
        return
    }
    var config ChainConfig
    err = json.Unmarshal(b, &config)
    if err != nil{
        fmt.Println("error unmarshalling config from file:", err)
        fmt.Println("resorting to defaults")
        mod.monk.Config = DefaultConfig
        return
    }
    mod.monk.Config = &config
}
func (mod *MonkModule) SetConfig(config interface{}) error{
    if s, ok := config.(string); ok{
        mod.ReadConfig(s)
    } else if s, ok := config.(ChainConfig); ok{
        mod.monk.Config = &s
    } else {
        return errors.New("could not set config")
    }
    return nil
}

// configure an ethereum node
func (monk *Monk) EthConfig() {
    cfg := monk.Config
    if cfg.LLLPath != ""{
	    monkutil.PathToLLL = cfg.LLLPath
    }
    monkchain.ContractPath = cfg.ContractPath
    if cfg.GenesisConfig != ""{
        monkchain.GenesisConfig = cfg.GenesisConfig
        fmt.Println("monkchain gen:", monkchain.GenesisConfig)
    }
    monkchain.DougDifficulty = monkutil.BigPow(2, cfg.DougDifficulty)
    monkutil.ReadConfig(path.Join(cfg.RootDir, "config"), cfg.RootDir, "monkchain")
    // data dir, logfile, log level, debug file
    InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, "")
}
