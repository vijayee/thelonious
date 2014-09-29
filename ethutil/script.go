// +build !windows !cgo

package ethutil

import (
	"fmt"
	"strings"
    "errors"
    "path"
	"github.com/obscuren/mutan"
	"github.com/obscuren/mutan/backends"
	"github.com/project-douglas/lllc-server"
    
	//"github.com/obscuren/serpent-go"
)

// this can be overwritten by higher-level constructs
// ethtest/config.go will reset it from config file
var PathToLLL = path.Join("/Users/BatBuddha/Programming/goApps/src/github.com/project-douglas/cpp-ethereum/build/lllc/lllc")

// General compile function
// compiles lll or mu according to extension on script
// script must be a file name!
func Compile(script string, silent bool) (ret []byte, err error) {
	if len(script) > 2 {
        fmt.Println("script", script, script[len(script)-4:])
        l := len(script)
        if script[l-4:] == ".lll"{
            fmt.Println("LLL")
            byteCode, err := CompileLLL(script)
            if err != nil{
                return nil, err                
            }
            return byteCode, nil
            
             
        /*
		line := strings.Split(script, "\n")[0]
		if len(line) > 1 && line[0:2] == "#!" {
			switch line {
			case "#!serpent":
				byteCode, err := serpent.Compile(script)
				if err != nil {
					return nil, err
				}

				return byteCode, nil
			}
            */
		} else if script[l-2:] == ".mu"{
			compiler := mutan.NewCompiler(backend.NewEthereumBackend())
			compiler.Silent = silent
			byteCode, errors := compiler.Compile(strings.NewReader(script))
			if len(errors) > 0 {
				var errs string
				for _, er := range errors {
					if er != nil {
						errs += er.Error()
					}
				}
				return nil, fmt.Errorf("%v", errs)
			}

			return byteCode, nil
		} else{
            //
        }
	}

	return nil, nil
}

// compile LLL file into evm bytecode 
func CompileLLL(filename string) ([]byte, error){
    fmt.Println("filename", filename, PathToLLL)
    // if we don't have the lllc locally, use the server
    if PathToLLL == "NETCALL"{
            //url := "http://ps.erisindustries.com/compile"
            lllcserver.URL = "http://162.218.65.211:9999/compile"
            resp, err := lllcserver.CompileLLLClient([]string{filename})
            // check for internal error
            if err != nil{
                return nil, err    
            }
            // check for compilation error
            if resp.Error[0] != ""{
                return nil, errors.New(resp.Error[0]) 
            }
            return resp.Bytecode[0], nil
    }
    lllcserver.PathToLLL = PathToLLL
    return lllcserver.CompileLLLWrapper(filename)
}

