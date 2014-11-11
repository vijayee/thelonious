// +build !windows !cgo

package monkutil

import (
	"fmt"
	"strings"
    "errors"
	"github.com/obscuren/mutan"
	"github.com/obscuren/mutan/backends"
	"github.com/project-douglas/lllc-server"
    
	//"github.com/obscuren/serpent-go"
)

// this can be overwritten by higher-level constructs
// monk/config.go will reset it from config file
var PathToLLL = ExpandHomePath("~/cpp-ethereum/build/lllc/lllc")

// General compile function
// compiles lll or mu according to extension on script
// script must be a file name!
func Compile(script string, silent bool) (ret []byte, err error) {
	if len(script) > 2 {
        //fmt.Println("script", script, script[len(script)-4:])
        l := len(script)
        if script[l-4:] == ".lll"{
            //fmt.Println("LLL")
            byteCode, err := CompileLLL(script, false)
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
func CompileLLL(filename string, literal bool) ([]byte, error){
    //fmt.Println("filename", filename, PathToLLL)
    // if we don't have the lllc locally, use the server
    if PathToLLL == "NETCALL"{
            //url := "http://ps.erisindustries.com/compile"
            lllcserver.URL = "http://lllc.erisindustries.com/compile"
            resp, err := lllcserver.CompileLLLClient([]string{filename}, literal)
            // check for internal error
            if err != nil{
                return nil, err    
            }
            // check for compilation error
            if len(resp.Error) > 0 && resp.Error[0] != ""{
                return nil, errors.New(resp.Error[0]) 
            }
            return resp.Bytecode[0], nil
    }
    lllcserver.PathToLLL = PathToLLL
    // TODO: compile literals if running locally
    return lllcserver.CompileLLLWrapper(filename)
}

// strings and hex only
func PackTxDataArgs(args ... string) []byte{
    //fmt.Println("pack data:", args)
    ret := *new([]byte)
    for _, s := range args{
        if s[:2] == "0x"{
            t := s[2:]
            if len(t) % 2 == 1{
                t = "0"+t
            }
            x := Hex2Bytes(t)
            //fmt.Println(x)
            l := len(x)
            ret = append(ret, LeftPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            ret = append(ret, RightPadBytes(x, 32*((l + 31)/32))...)
        }
    }
   return ret
}

// strings and hex only
func PackTxDataArgs2(args ... string) []byte{
    //fmt.Println("pack data:", args)
    ret := *new([]byte)
    for _, s := range args{
        if s[:2] == "0x"{
            t := s[2:]
            if len(t) % 2 == 1{
                t = "0"+t
            }
            x := Hex2Bytes(t)
            //fmt.Println(x)
            l := len(x)
            ret = append(ret, LeftPadBytes(x, 32*((l + 31)/32))...)
        }else{
            x := []byte(s)
            l := len(x)
            ret = append(ret, LeftPadBytes(x, 32*((l + 31)/32))...)
        }
    }
   return ret
}
