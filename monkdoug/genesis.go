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
	// MetaGenDoug
	Address   string `json:"address"`    // bytes
	DougPath  string `json:"doug"`       // path to doug contract
	ModelName string `json:"model"`      // name of the gendoug access model
	NoGenDoug bool   `json:"no-gendoug"` // turn off gendoug

	hexAddr      string
	byteAddr     []byte
	contractPath string

	// Global GenDoug Singles
	Consensus    string `json:"consensus"` // stake, robin, eth
	Difficulty   int    `json:"difficulty"`
	PublicMine   int    `json:"public:mine"`
	PublicCreate int    `json:"public:create"`
	PublicTx     int    `json:"public:tx"`
	MaxGasTx     string `json:"maxgastx"`
	BlockTime    int    `json:"blocktime"`

	// Accounts (permissions and stake)
	Accounts []*Account `json:"accounts"`

	model PermModel
}

func (g *GenesisConfig) Model() PermModel {
	return g.model
}

func (g *GenesisConfig) SetModel(m PermModel) {
	g.model = m
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
	g.model = NewPermModel(g)

	return g
}

// Deploy the genesis block
// Converts the GenesisConfiginfo into a populated and functional doug contract in the genesis block
// if NoGenDoug, simply bankroll the accounts
func (g *GenesisConfig) Deploy(block *monkchain.Block) {
	block.Difficulty = monkutil.BigPow(2, g.Difficulty)
    
    defer func(b *monkchain.Block){
        b.State().Update()
        b.State().Sync()
    }(block)

	if g.NoGenDoug {
		// no genesis doug, deploy simple
		for _, account := range g.Accounts {
			// direct state modification to create accounts and balances
			AddAccount(account.byteAddr, account.Balance, block)
		}
		return
	}

	fmt.Println("###DEPLOYING DOUG", g.Address, g.DougPath)

	// dummy keys for signing
	//keys := monkcrypto.GenerateNewKeyPair()
	keys, err := monkcrypto.NewKeyPairFromSec([]byte("11111111112222222222333333333322"))
	if err != nil {
		log.Fatal("da fuq?", err)
	}
	fmt.Println(keys.Address())

	// create the genesis doug
	codePath := path.Join(g.contractPath, g.DougPath)
	genAddr := []byte(g.Address)
	MakeApplyTx(codePath, genAddr, nil, keys, block)

	// set the global vars
	g.model.SetValue(genAddr, []string{"setvar", "consensus", g.Consensus}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "difficulty", "0x" + monkutil.Bytes2Hex(big.NewInt(int64(g.Difficulty)).Bytes())}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "public:mine", "0x" + strconv.Itoa(g.PublicMine)}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "public:create", "0x" + strconv.Itoa(g.PublicCreate)}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "public:tx", "0x" + strconv.Itoa(g.PublicTx)}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "maxgastx", g.MaxGasTx}, keys, block)
	g.model.SetValue(genAddr, []string{"setvar", "blocktime", "0x" + strconv.Itoa(g.BlockTime)}, keys, block)

	fmt.Println("done genesis. setting perms...")

	// set balances and permissions
	for _, account := range g.Accounts {
		// direct state modification to create accounts and balances
		AddAccount(account.byteAddr, account.Balance, block)
		if g.model != nil {
			// issue txs to set perms according to the model
			g.model.SetPermissions(account.byteAddr, account.Permissions, block, keys)
			if account.Permissions["mine"] != 0 {
				g.model.SetValue(g.byteAddr, []string{"addminer", account.Name, "0x" + account.Address, "0x" + strconv.Itoa(account.Stake)}, keys, block)
			}
			fmt.Println("setting perms for", account.Address)
		}
	}
	block.Sign(keys.PrivateKey)
}

// set balance of an account (does not commit)
func AddAccount(addr []byte, balance string, block *monkchain.Block) {
	account := block.State().GetAccount(addr)
	account.Balance = monkutil.Big(balance) //monkutil.BigPow(2, 200)
	block.State().UpdateStateObject(account)
}

// return a new permissions model
// TODO: cleaner differentiation between consensus and storage access models
func NewPermModel(g *GenesisConfig) (model PermModel) {
	modelName := g.ModelName
	if g.NoGenDoug {
		modelName = "default"
	}
	switch modelName {
	case "fake":
		// simplified genesis permission structure
		//model = NewFakeModel(dougAddr)
	case "dennis":
		// gendoug-v1
		//model = NewGenDougModel(dougAddr)
	case "std":
		// gendoug-v2
		model = NewStdLibModel(g)
	case "yes":
		// everyone allowed
		model = NewYesModel(g)
	case "no":
		// noone allowed
		model = NewNoModel(g)
	case "eth":
		model = NewEthModel(g)
	default:
		model = NewYesModel(g) //NewEthModel(g)
	}
	return
}
