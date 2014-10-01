package ethtest

import (
    "errors"
    "path"
    "github.com/eris-ltd/eth-go-mods/ethutil"
    "github.com/eris-ltd/eth-go-mods/ethchain"
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
    LLLPath string `json:"lll_path"`
    ContractPath string `json:"contract_path"`
    ClientIdentifier string `json:"client"`
    Version string  `json:"version"`
    Identifier string `json:"id"`
    KeyStore string `json:"keystore"`
    GenesisPointer string `json:"genesis_func"`
    DougDifficulty int `json:"difficulty"`
}

// set default config object
var DefaultConfig = &ChainConfig{ 
        Port : 30303,
        Mining : false,
        MaxPeers : 10,
        ConfigFile : "config",
        RootDir : path.Join(usr.HomeDir, ".ethchain"),
        KeyFile : path.Join(GoPath, "src", "github.com", "eris-ltd", "eth-go-mods", "ethtest", "keys.txt"),
        Name : "decerver-ethchain",
        LogFile: "",
        DataDir: path.Join(homeDir(), ".eris-eth"),
       // LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
        LLLPath: "NETCALL",
       // ContractPath: path.Join(GoPath, "src", "github.com", "eris-ltd", "eth-go-mods", "ethtest", "contracts"),
        ContractPath: path.Join(GoPath, "src", "github.com", "eris-ltd", "eris-std-lib"),
        ClientIdentifier: "Ethereum(deCerver)",
        Version: "0.5.17",
        Identifier: "",
        KeyStore: "db",
        GenesisPointer: "txs-by-doug",
        DougDifficulty: 15,
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
    ethchain.ContractPath = e.Config.ContractPath
    ethchain.DougDifficulty = ethutil.BigPow(2, e.Config.DougDifficulty)
    ethutil.ReadConfig(path.Join(e.Config.RootDir, "config"), e.Config.RootDir, "ethchain")
    // data dir, logfile, log level, debug file
    InitLogging(e.Config.RootDir, e.Config.LogFile, 5, "")

}
