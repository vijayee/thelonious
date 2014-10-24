# EthTest

--------------------------------------------------
This used to be deCerver/chain, and offers a simple interface to the blockchain (possibly still for use in deCerver)

It also offers a set of tests for testing contract creation, execution, storage, etc.

Tests are in `storage.go` and `genesis.go` and the framework is defined in `test.go`.

Within `monk`, run a test with `go run tests/main.go -t testname`, where `testname` is one of:
   *  basic: start a node, start mining, stop after a few seconds
   *  run: run a node, mine, and keep running
   *  tx: send a simple tx
   *  traverse: traverse back to genesis block
   *  genesis: print the genesis accounts
   *  genesis-msg: msg the contract in the genesis block
   *  get-storage: get storage from a contract addr
   *  msg-storage: msg a contract and have it store a value
   *  validate: accept/reject blocks based on permissions in genesis doug


Note testing is not really formalized yet, and still requires manually parsing some output. There's also no nice way to set parameters other than mucking in code. Will come as necessary.

Run a node simply with `go run tests/main.go -t run`
