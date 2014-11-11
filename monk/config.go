package monk

import (
    "os"
    "io"
    "fmt"
    "path"
    "bytes"
    "reflect"
    "io/ioutil"
    "encoding/json"
    "github.com/eris-ltd/thelonious/monkutil"
    "github.com/eris-ltd/thelonious/monkchain"
)

var ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")

type ChainConfig struct{
    Port int        `json:"port"`
    Mining bool     `json:"mining"`
    MaxPeers int    `json:"max_peers"`
    ConfigFile string `json:"config_file"`
    RootDir string  `json:"root_dir"`
    Name string     `json:"name"`
    LogFile string  `json:"log_file"`
    DataDir string `json:"data_dir"`
    DbName string `json:"db_name"`
    LLLPath string `json:"lll_path"`
    ContractPath string `json:"contract_path"`
    ClientIdentifier string `json:"client"`
    Version string  `json:"version"`
    Identifier string `json:"id"`
    KeySession string  `json:"key_session"`
    KeyStore string `json:"key_store"`
    KeyCursor int `json:"key_cursor"`
    KeyFile string  `json:"key_file"`
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
        KeySession: "generous",
        Name : "decerver-monkchain",
        LogFile: "",
        //LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
        LLLPath: "NETCALL",
        ContractPath: path.Join(ErisLtd, "eris-std-lib"),
        ClientIdentifier: "Ethereum(deCerver)",
        Version: "0.5.17",
        Identifier: "chainId",
        KeyStore: "file",
        KeyCursor: 0,
        KeyFile: path.Join(ErisLtd, "thelonious", "monk", "keys.txt"),
        GenesisConfig: path.Join(ErisLtd, "thelonious", "monk", "genesis-std.json"),
        DougDifficulty: 7,
        LogLevel: 5,
}


// can these methods be functions in decerver that take the modules as argument?
func (mod *MonkModule) WriteConfig(config_file string){
    b, err := json.Marshal(mod.monk.config)
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
        //mod.monk.config = DefaultConfig
        return
    }
    mod.monk.config = &config
}

func (mod *MonkModule) SetConfig(field string, value interface{}) error{
    cv := reflect.ValueOf(mod.monk.config).Elem()
    f := cv.FieldByName(field)
    kind := f.Kind()

    k := reflect.ValueOf(value).Kind()
    if kind != k{
        return fmt.Errorf("Invalid kind. Expected %s, received %s", kind, k)
    }
    
    if kind == reflect.String{
        f.SetString(value.(string))
    } else if kind == reflect.Int{
        f.SetInt(int64(value.(int)))
    } else if kind == reflect.Bool{
        f.SetBool(value.(bool))
    }
    return nil
}

// this will probably never be used
func (mod *MonkModule) SetConfigObj(config interface{}) error{
    if c, ok := config.(*ChainConfig); ok{
        mod.monk.config = c
    } else{
        return fmt.Errorf("Invalid config object")
    }
    return nil
}

// configure an ethereum node
func (monk *Monk) EthConfig() {
    cfg := monk.config
    if cfg.LLLPath != ""{
	    monkutil.PathToLLL = cfg.LLLPath
    }
    monkchain.ContractPath = cfg.ContractPath
    if cfg.GenesisConfig != ""{
        monkchain.GenesisConfig = cfg.GenesisConfig
        fmt.Println("monkchain gen:", monkchain.GenesisConfig)
    }
    monkchain.DougDifficulty = monkutil.BigPow(2, cfg.DougDifficulty)

    // check on data dir
    // create keys
    _, err := os.Stat(cfg.RootDir)
    if err != nil{
        os.Mkdir(cfg.RootDir, 0777)
        _, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession)+".prv")
        if err != nil{
            Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
        }
    }
    // eth-go uses a global monkutil.Config object. This will set it up for us, but we do our config of course our way
    // it also uses rakyl/globalconf, but fuck that for now
    monkutil.Config = &monkutil.ConfigManager{ExecPath: cfg.RootDir, Debug: true, Paranoia: true}
    // data dir, logfile, log level, debug file
    // TODO: enhance this with more pkg level control
    InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, "")
}

// common golang, really?
func Copy(src, dst string){
    r, err := os.Open(src)
    if err != nil{
        fmt.Println(src, err)
        logger.Errorln(err)
        return
    }
    defer r.Close()

    w, err := os.Create(dst)
    if err != nil{
        fmt.Println(err)
        logger.Errorln(err)
        return
    }
    defer w.Close()

    _, err = io.Copy(w, r)
    if err != nil{
        fmt.Println(err)
        logger.Errorln(err)
        return
    }
}
