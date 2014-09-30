package ethtest

import (
    "github.com/eris-ltd/eth-go-mods/ethchain"
    "path"
)

func (t *Test) TestCallStack(){
    t.tester("callstack", func(eth *EthChain){
        eth.Start()
        eth.DeployContract(path.Join(ethchain.ContractPath, "lll/callstack.lll"), "lll")
        t.callback("op: callstack", eth, func(){
            PrettyPrintChainAccounts(eth)
        })

    }, 0)
}
