package ethtest

import (
    "errors"
    "path"
    "github.com/eris-ltd/eth-go-mods/ethutil"
)


type ChainConfig struct{
    Port int        `json:"port"`
    Mining bool     `json:"mining"`
    MaxPeers int    `json:"max_peers"`
    ConfigFile string `json:"config_file"`
    RootDir string  `json:"root_dir"`
    Name string     `json:"name"`
    LogFile string  `json:"log_file"`
    DataDir string `json:"data_dir"`
    LLLPath string `json:"lll_path"`
    ContractPath string `json:"contract_path"`
    ClientIdentifier string `json:"client"`
    Version string  `json:"version"`
    Identifier string `json:"id"`
    KeyStore string `json:"keystore"`
}

// set default config object
var DefaultConfig = &ChainConfig{ 
        Port : 30303,
        Mining : false,
        MaxPeers : 10,
        ConfigFile : "config",
        RootDir : path.Join(usr.HomeDir, ".ethchain"),
        Name : "decerver-ethchain",
        LogFile: "",
        DataDir: path.Join(homeDir(), ".eris-eth"),
        LLLPath: path.Join(homeDir(), "Programming/goApps/src/github.com/project-douglas/cpp-ethereum/build/lllc/lllc"),
        ContractPath: path.Join(GoPath, "src", "github.com", "eris-ltd", "eth-go-mods", "ethtest", "contracts"),
        ClientIdentifier: "Ethereum(deCerver)",
        Version: "0.5.17",
        Identifier: "",
        KeyStore: "db",
}


// can these methods be functions in decerver that take the modules as argument?
func (e *EthChain) WriteConfig(config_file string){
}
func (e *EthChain) ReadConfig(config_file string){
}
func (e *EthChain) SetConfig(config interface{}) error{
    if s, ok := config.(string); ok{
        e.ReadConfig(s)
    } else if s, ok := config.(ChainConfig); ok{
        e.Config = &s
    } else {
        return errors.New("could not set config")
    }
    return nil
}

// configure an ethereum node
func (e *EthChain) EthConfig() {
    ethutil.PathToLLL = e.Config.LLLPath
    ethutil.ReadConfig(path.Join(e.Config.RootDir, "config"), e.Config.RootDir, "ethchain")
    InitLogging(e.Config.RootDir, e.Config.LogFile, 5, "")
}
