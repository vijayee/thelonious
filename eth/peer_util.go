package eth

import (
	"encoding/json"

	"github.com/eris-ltd/new-thelonious/monkutil"
)

func WritePeers(path string, addresses []string) {
	if len(addresses) > 0 {
		data, _ := json.MarshalIndent(addresses, "", "    ")
		monkutil.WriteFile(path, data)
	}
}

func ReadPeers(path string) (ips []string, err error) {
	var data string
	data, err = monkutil.ReadAllFile(path)
	if err != nil {
		json.Unmarshal([]byte(data), &ips)
	}
	return
}
