package monk

import (
    "github.com/eris-ltd/thelonious/ethutil"
    "github.com/eris-ltd/thelonious/ethcrypto"
    //"github.com/eris-ltd/thelonious/ethpipe"
    "io/ioutil"
    "strings"
    "log"
    //"strconv"
)

func newKeyPair(keyMang *ethcrypto.KeyManager){
    keyPair := ethcrypto.GenerateNewKeyPair()
    keyMang.KeyRing().AddKeyPair(keyPair)
//    keyRing.NewKeyPair(keyPair.PrivateKey)
}

// private keys in plain-text hex format one per line
func  LoadKeys(filename string, keyMang *ethcrypto.KeyManager){
    keyData, err := ioutil.ReadFile(filename)
    if err != nil{
        log.Println("Could not find keys file. Creating new keypair...")        
        newKeyPair(keyMang)
    } else { 
        keys := strings.Split(string(keyData), "\n")
        for _, k := range keys{
            if len(k) == 64{
                keyPair, err := ethcrypto.NewKeyPairFromSec(ethutil.Hex2Bytes(k))
                if err == nil{
                    log.Println("adding keypair")
                    keyMang.KeyRing().AddKeyPair(keyPair)
                }
            }
        }
    }
    if keyMang.KeyRing().Len() == 0{
        newKeyPair(keyMang)
    }
    keyMang.SetCursor(1)
    logger.Infoln("Keys loaded: ", keyMang.KeyRing().Len())
}

/*
func CheckZeroBalance(pipe *ethpipe.Pipe, keyMang *ethcrypto.KeyManager){
    keys := keyMang.KeyRing()
    master := ethutil.Bytes2Hex(keys.GetKeyPair(keys.Len()-1).PrivateKey)
    logger.Infoln("master has ", pipe.Balance(keys.GetKeyPair(keys.Len()-1).Address()) )
    for i:=0; i<keys.Len();i++{
        k := keys.GetKeyPair(i).Address()
        val := pipe.Balance(k)
        logger.Infoln("key ", i, " ", ethutil.Bytes2Hex(k), " ", val)
        v, _ := strconv.Atoi(val)
        if v < 100 {
            _, err := pipe.Transact(master, ethutil.Bytes2Hex(k), "10000000000000000000", "1000", "1000", "")
            if err != nil{
                logger.Infoln("Error transfering funds to ", ethutil.Bytes2Hex(k))
            }
        }
    }
}*/

