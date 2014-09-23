package ethutil

import (
	"fmt"
	"strings"
    "path"
    "os/exec"
    "bytes"
	"github.com/obscuren/mutan"
	"github.com/obscuren/mutan/backends"
    
	//"github.com/obscuren/serpent-go"
)

var PathToLLL = path.Join("/Users/BatBuddha/Programming/goApps/src/github.com/project-douglas/cpp-ethereum/build/lllc/lllc")

// General compile function
func Compile(script string, silent bool) (ret []byte, err error) {
    fmt.Println("script", script, script[len(script)-4:])
	if len(script) > 2 {
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
		} else {
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
		}
	}

	return nil, nil
}

// compile LLL file into evm bytecode 
func CompileLLL(filename string) ([]byte, error){
    fmt.Println("filename", filename, PathToLLL)
    cmd := exec.Command(PathToLLL, filename)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        fmt.Println("Couldn't compile!!", err)
        return nil, err
    }
    //outstr := strings.Split(out.String(), "\n")
    outstr := out.String()
    for l:=len(outstr);outstr[l-1] == '\n';l--{
        outstr = outstr[:l-1]
    }
    fmt.Println("script hex", outstr)
    //return "0x"+outstr, nil
    return Hex2Bytes(outstr), nil
}

