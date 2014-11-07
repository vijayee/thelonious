package monk

import (
    "testing"
    "github.com/eris-ltd/deCerver-interfaces/modules"
)

func receiveModule(m modules.Module){
}

func receiveBlockchain(m modules.Blockchain){
}

func TestModule(t *testing.T){
    tester("module satisfaction", func(mod *MonkModule){
        receiveModule(mod)        
        receiveBlockchain(mod)
        receiveBlockchain(mod.monk)
    }, 0)
}
