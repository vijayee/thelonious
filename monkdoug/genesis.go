package monkdoug

import (
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkutil"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"reflect"
	"strconv"
)

/*
   Configure a new genesis block from genesis.json
   Deploy the genesis block
*/

type Account struct {
	Address     string         `json:"address"`
	byteAddr    []byte         //convenience, but not from json
	Name        string         `json:"name"`
	Balance     string         `json:"balance"`
	Permissions map[string]int `json:"permissions"`
	Stake       int            `json:"stake"`
}

type GenesisConfig struct {
	/*
	   MetaGenDoug
	*/
	// 20 ASCI bytes of gendoug addr
	Address string `json:"address"`
	// Path to lll doug contract
	DougPath string `json:"doug"`
	// Should gendoug be unique (set true in production)
	Unique bool `json:"unique"`
	// A private key to seed uniqueness (otherwise is random)
	PrivateKey string `json:"private-key"`
	// Name of the gendoug access model (yes, no, std, vm, eth)
	ModelName string `json:"model"`
	// Turn off gendoug
	NoGenDoug bool `json:"no-gendoug"`

	/*
	   Global GenDoug Singles
	*/
	// Consensus/difficulty mechanism (stake, robin, constant, eth)
	Consensus string `json:"consensus"`
	// Starting difficulty level
	Difficulty int `json:"difficulty"`
	// Allow anyone to mine
	PublicMine int `json:"public:mine"`
	// Allow anyone to create contracts
	PublicCreate int `json:"public:create"`
	// Allow anyone to transact
	PublicTx int `json:"public:tx"`
	// Max gas per tx
	MaxGasTx string `json:"maxgastx"`
	// Target block time (shaky...)
	BlockTime int `json:"blocktime"`

	// Paths to lll consensus contracts (if ModelName = vm)
	Vm *VmConsensus `json:"vm"`

	// Accounts (permissions and stake)
	Accounts []*Account `json:"accounts"`

	// for convenience, not filled in by json
	hexAddr      string
	byteAddr     []byte
	contractPath string

	// Gendoug based protocol interface
	// for verifying blocks/txs
	protocol  monkchain.Protocol
	consensus monkchain.Consensus

	// Signed genesis block (hex)
	chainId string
}

// A protocol level call executed through the vm
type SysCall struct {
	// Path to lll code for this function
	CodePath string `json:"code-path"`
	// Should we use doug's state or our own
	// TODO: this the kind of thing that may require us
	// to have the genesis.json and not just the genesis block!
	// Phase it out!
	Doug bool `json:"doug"`
	// Addr of this contract
	Addr     string `json:"addr"`
	byteAddr []byte
}

type VmConsensus struct {
	// Name of a suite of contracts
	SuiteName string `json:"suite-name"`
	// Path to lll permission verify contract
	PermissionVerify SysCall `json:"permission-verify"`
	// Path to lll block verify contract
	BlockVerify SysCall `json:"block-verify"`
	// Path to lll tx verify contract
	TxVerify SysCall `json:"tx-verify"`
	// Path to lll compute difficulty contract
	// Calculate difficulty for block from parent (and storage)
	ComputeDifficulty SysCall `json:"compute-difficulty"`
	// Path to lll participate contract
	// Determine if a coinbase should participate in consensus
	ComputeParticipate SysCall `json:"compute-participate"`
	// Participate/Pledge contract
	Participate SysCall `json:"participate"`
	// Contract to run at the beginning of a block
	PreCall SysCall `json:"precall"`
	// Contract to run at the end of a block
	PostCall SysCall `json:"postcall"`
	// Other contracts for arbitrary functionality
	Other []SysCall `json:"other"`
}

func (g *GenesisConfig) Model() monkchain.Protocol {
	return g.protocol
}

func (g *GenesisConfig) SetModel() {
	g.protocol = NewProtocol(g)
}

// Load the genesis block info from genesis.json
func LoadGenesis(file string) *GenesisConfig {
	fmt.Println("reading ", file)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("err reading genesis.json", err)
		os.Exit(0)
	}

	g := new(GenesisConfig)
	err = json.Unmarshal(b, g)
	if err != nil {
		fmt.Println("error unmarshalling genesis.json", err)
		os.Exit(0)
	}

	// move address into accounts, in bytes
	for _, acc := range g.Accounts {
		acc.byteAddr = monkutil.UserHex2Bytes(acc.Address)
	}

	g.byteAddr = []byte(g.Address)
	g.hexAddr = monkutil.Bytes2Hex(g.byteAddr)
	g.contractPath = path.Join(ErisLtd, "eris-std-lib")

	// set doug model
	g.protocol = NewProtocol(g)

	return g
}

// Deploy the genesis block
// Converts the GenesisConfiginfo into a populated and functional doug contract in the genesis block
// if NoGenDoug, simply bankroll the accounts
// TODO: offer an EPM version
func (g *GenesisConfig) Deploy(block *monkchain.Block) []byte {
	block.Difficulty = monkutil.BigPow(2, g.Difficulty)

	defer func(b *monkchain.Block) {
		b.State().Update()
		b.State().Sync()
	}(block)

	if g.NoGenDoug {
		// no genesis doug, deploy simple
		for _, account := range g.Accounts {
			// direct state modification to create accounts and balances
			AddAccount(account.byteAddr, account.Balance, block)
		}
		// TODO: make sure defer happens first!
		return block.Hash()
	}

	fmt.Println("###DEPLOYING DOUG", g.Address, g.DougPath)

	// Keys for creating valid txs and for signing
	// the final gendoug
	// Must be unique for production use!
	var keys *monkcrypto.KeyPair
	var err error
	if g.Unique {
		if g.PrivateKey != "" {
			// TODO: some kind of encryption here ...
			decoded := monkutil.Hex2Bytes(g.PrivateKey)
			keys, err = monkcrypto.NewKeyPairFromSec(decoded)
			if err != nil {
				log.Fatal("Invalid private key", err)
			}
		} else {
			keys = monkcrypto.GenerateNewKeyPair()
		}
	} else {
		static := []byte("11111111112222222222333333333322")
		keys, err = monkcrypto.NewKeyPairFromSec(static)
		if err != nil {
			log.Fatal("Invalid static private", err)
		}
	}
	fmt.Println(keys.Address())

	// create the genesis doug
	codePath := path.Join(g.contractPath, g.DougPath)
	genAddr := []byte(g.Address)
	MakeApplyTx(codePath, genAddr, nil, keys, block)

	// set the global vars
	SetValue(genAddr, []string{"setvar", "consensus", g.Consensus}, keys, block)
	SetValue(genAddr, []string{"setvar", "difficulty", "0x" + monkutil.Bytes2Hex(big.NewInt(int64(g.Difficulty)).Bytes())}, keys, block)
	SetValue(genAddr, []string{"setvar", "public:mine", "0x" + strconv.Itoa(g.PublicMine)}, keys, block)
	SetValue(genAddr, []string{"setvar", "public:create", "0x" + strconv.Itoa(g.PublicCreate)}, keys, block)
	SetValue(genAddr, []string{"setvar", "public:tx", "0x" + strconv.Itoa(g.PublicTx)}, keys, block)
	SetValue(genAddr, []string{"setvar", "maxgastx", g.MaxGasTx}, keys, block)
	SetValue(genAddr, []string{"setvar", "blocktime", "0x" + strconv.Itoa(g.BlockTime)}, keys, block)

	// set balances and permissions
	for _, account := range g.Accounts {
		// direct state modification to create accounts and balances
		AddAccount(account.byteAddr, account.Balance, block)
		if g.protocol != nil {
			// issue txs to set perms according to the model
			SetPermissions(genAddr, account.byteAddr, account.Permissions, block, keys)
			if account.Permissions["mine"] != 0 {
				SetValue(g.byteAddr, []string{"addminer", account.Name, "0x" + account.Address, "0x" + strconv.Itoa(account.Stake)}, keys, block)
			}
			fmt.Println("setting perms for", account.Address)
		}
	}

	// set verification contracts for "vm" consensus
	if g.ModelName == "vm" {
		if g.Vm == nil {
			log.Fatal("Model=vm requires non-nil VmConsensus obj")
		}

		suite := suites[g.Vm.SuiteName]

		// loop through g.Vm fields
		// deploy the non-nil ones
		// fall back to suite (if set) or nothing (default)
		m := g.protocol.(*Protocol).consensus.(*VmModel)
		gvm := reflect.ValueOf(g.Vm).Elem()
		svm := reflect.ValueOf(suite).Elem()
		typeOf := gvm.Type()
		// First is suite name, last is list of SysCalls (deal with later)
		for i := 1; i < gvm.NumField()-1; i++ {
			def := true
			f := gvm.Field(i)
			name := typeOf.Field(i).Name
			addr := monkutil.LeftPadBytes([]byte(name), 20)
			tag := typeOf.Field(i).Tag.Get("json")
			useDoug := false
			// value of f is a SysCall struct
			v := f.FieldByName("CodePath")
			val := v.String()
			// if val exists, overwrite suite defaults with config values
			if val != "" { //v.IsValid(){
				val = v.String()
				def = false
			} else if suite != nil {
				// field is set by suite
				c := svm.FieldByName(name)
				v := c.FieldByName("CodePath")
				val = v.String()
				if val != "" { //v.IsValid() {
					val = v.String()
					f = c
					def = false
				}
			}
			if !def {
				if a := f.FieldByName("Addr").String(); a != "" {
					if len(a) > 20 && len(monkutil.UserHex2Bytes(a)) == 20 {
						addr = monkutil.UserHex2Bytes(a)
					} else {
						addr = monkutil.LeftPadBytes([]byte(a), 20)
					}
				}
				if a := f.FieldByName("Doug").String(); a != "" {
					useDoug = true
				}
				codePath := path.Join(g.contractPath, val)
				_, _, err := MakeApplyTx(codePath, addr, nil, keys, block)
				if err == nil {
					s := SysCall{
						Addr:     monkutil.Bytes2Hex(addr),
						byteAddr: addr,
						Doug:     useDoug,
						CodePath: codePath,
					}
					m.contract[tag] = s
				}
				SetValue(genAddr, []string{"setvar", tag, "0x" + monkutil.Bytes2Hex(addr)}, keys, block)
			}

		}
		//TODO handle final element in Vm struct (list of SysCalls)
	}

	// This is the chainID (65 bytes)
	chainId := block.Sign(keys.PrivateKey)
	g.chainId = monkutil.Bytes2Hex(chainId)
	return chainId
}

// set balance of an account (does not commit)
func AddAccount(addr []byte, balance string, block *monkchain.Block) {
	account := block.State().GetAccount(addr)
	account.Balance = monkutil.Big(balance) //monkutil.BigPow(2, 200)
	block.State().UpdateStateObject(account)
}

//
func NewProtocol(g *GenesisConfig) monkchain.Protocol {
	consensus := NewPermModel(g)
	p := &Protocol{g: g, consensus: consensus}
	return p
}

// Return a new permissions model
// Only "std" and "vm" care about gendoug
// NoGendoug defaults to the "yes" model
func NewPermModel(g *GenesisConfig) (model monkchain.Consensus) {
	modelName := g.ModelName
	if g.NoGenDoug {
		modelName = "yes"
	}
	switch modelName {
	case "std":
		// gendoug-v2
		// uses eris-std-lib/gotests/vars for reading
		// from gendoug
		model = NewStdLibModel(g)
	case "vm":
		// run processing through the vm
		model = NewVmModel(g)
	case "yes":
		// everyone allowed everything
		model = NewYesModel(g)
	case "no":
		// noone allowed anything
		model = NewNoModel(g)
	case "eth":
		// ethereum
		g.NoGenDoug = true
		model = NewEthModel(g)
	default:
		// default to yes
		model = NewYesModel(g)
	}
	return
}

// A default genesis.json
// TODO: make a lookup-able suite of these
var DefaultGenesis = GenesisConfig{
	NoGenDoug:  true,
	Difficulty: 15,
	Accounts: []*Account{
		&Account{
			Address:  "0xbbbd0256041f7aed3ce278c56ee61492de96d001",
			byteAddr: monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"),
			Balance:  "1000000000000000000000000000000000000",
		},
	},
}

// Contract suites for vm based protocol
var suites = map[string]*VmConsensus{
	"std": &VmConsensus{
		SuiteName:          "std",
		PermissionVerify:   SysCall{"", true, "", nil},
		BlockVerify:        SysCall{"Protocol/block-verify.lll", true, "", nil},
		TxVerify:           SysCall{"Protocol/tx-verify.lll", true, "", nil},
		ComputeDifficulty:  SysCall{"", true, "", nil},
		ComputeParticipate: SysCall{"", true, "", nil},
		Participate:        SysCall{"", true, "", nil},
		PreCall:            SysCall{"", true, "", nil},
		PostCall:           SysCall{"", true, "", nil},
	},
}
