package monk

import (
    "testing"
    "time"
    "github.com/eris-ltd/decerver-interfaces/modules"
)

func receiveModule(m modules.Module){
}

func receiveBlockchain(m modules.Blockchain){
}

// Static type checking to ensure the module and blockchain interfaces are satisfied
func TestModule(t *testing.T){
    tester("module satisfaction", func(mod *MonkModule){
        receiveModule(mod)        
        receiveBlockchain(mod)
        receiveBlockchain(mod.monk)
    }, 0)
}


func TestSubscribe(t *testing.T){
    tester("subscribe/unsuscribe", func(mod *MonkModule){
        mod.Init()
        name := "testNewBlock"
        ch := mod.Subscribe(name, "newBlock", "")
        go func(){
            for{
                a, more := <- ch
                if !more{
                    break
                }
                if _, ok := a.Resource.(*modules.Block); !ok{
                    t.Error("Event resource not a block!")
                }
            }
        }()
        mod.Start()
        time.Sleep(4*time.Second)
        mod.UnSubscribe("testNewBlock")
    }, 0)
}
