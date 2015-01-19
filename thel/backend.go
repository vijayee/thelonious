package thel

import (
	"fmt"
	"net"
	"sync"

	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/event"
	ethlogger "github.com/eris-ltd/new-thelonious/logger"
	"github.com/eris-ltd/new-thelonious/p2p"
	"github.com/eris-ltd/new-thelonious/pow/ezp"
	"github.com/eris-ltd/new-thelonious/rpc"
	"github.com/eris-ltd/new-thelonious/theldb"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/whisper"
)

const (
	seedNodeAddress = "poc-8.ethdev.com:30303"
)

type Config struct {
	Name       string
	Version    string
	Identifier string
	KeyStore   string
	DataDir    string
	LogFile    string
	LogLevel   int
	KeyRing    string

	MaxPeers   int
	Port       string
	NATType    string
	PMPGateway string

	Shh  bool
	Dial bool

	KeyManager *crypto.KeyManager
}

var logger = ethlogger.NewLogger("SERV")

type Thelonious struct {
	// Channel for shutting down the ethereum
	shutdownChan chan bool
	quit         chan bool

	// DB interface
	db        thelutil.Database
	blacklist p2p.Blacklist

	//*** SERVICES ***
	// State manager for processing new blocks and managing the over all states
	blockProcessor *core.BlockProcessor
	txPool         *core.TxPool
	chainManager   *core.ChainManager
	blockPool      *BlockPool
	whisper        *whisper.Whisper

	net      *p2p.Server
	eventMux *event.TypeMux
	txSub    event.Subscription
	blockSub event.Subscription

	RpcServer  *rpc.JsonRpcServer
	keyManager *crypto.KeyManager

	clientIdentity p2p.ClientIdentity
	logger         ethlogger.LogSystem

	synclock  sync.Mutex
	syncGroup sync.WaitGroup

	Mining bool
}

func New(config *Config) (*Thelonious, error) {
	// Boostrap database
	logger := ethlogger.New(config.DataDir, config.LogFile, config.LogLevel)
	db, err := theldb.NewLDBDatabase("blockchain")
	if err != nil {
		return nil, err
	}

	// Perform database sanity checks
	d, _ := db.Get([]byte("ProtocolVersion"))
	protov := thelutil.NewValue(d).Uint()
	if protov != ProtocolVersion && protov != 0 {
		return nil, fmt.Errorf("Database version mismatch. Protocol(%d / %d). `rm -rf %s`", protov, ProtocolVersion, thelutil.Config.ExecPath+"/database")
	}

	// Create new keymanager
	var keyManager *crypto.KeyManager
	switch config.KeyStore {
	case "db":
		keyManager = crypto.NewDBKeyManager(db)
	case "file":
		keyManager = crypto.NewFileKeyManager(config.DataDir)
	default:
		return nil, fmt.Errorf("unknown keystore type: %s", config.KeyStore)
	}
	// Initialise the keyring
	keyManager.Init(config.KeyRing, 0, false)

	// Create a new client id for this instance. This will help identifying the node on the network
	clientId := p2p.NewSimpleClientIdentity(config.Name, config.Version, config.Identifier, keyManager.PublicKey())

	saveProtocolVersion(db)
	//thelutil.Config.Db = db

	eth := &Thelonious{
		shutdownChan:   make(chan bool),
		quit:           make(chan bool),
		db:             db,
		keyManager:     keyManager,
		clientIdentity: clientId,
		blacklist:      p2p.NewBlacklist(),
		eventMux:       &event.TypeMux{},
		logger:         logger,
	}

	eth.chainManager = core.NewChainManager(db, eth.EventMux())
	eth.txPool = core.NewTxPool(eth.EventMux())
	eth.blockProcessor = core.NewBlockProcessor(db, eth.txPool, eth.chainManager, eth.EventMux())
	eth.chainManager.SetProcessor(eth.blockProcessor)
	eth.whisper = whisper.New()

	hasBlock := eth.chainManager.HasBlock
	insertChain := eth.chainManager.InsertChain
	eth.blockPool = NewBlockPool(hasBlock, insertChain, ezp.Verify)

	ethProto := EthProtocol(eth.txPool, eth.chainManager, eth.blockPool)
	protocols := []p2p.Protocol{ethProto, eth.whisper.Protocol()}

	nat, err := p2p.ParseNAT(config.NATType, config.PMPGateway)
	if err != nil {
		return nil, err
	}
	fmt.Println(nat)

	eth.net = &p2p.Server{
		Identity:  clientId,
		MaxPeers:  config.MaxPeers,
		Protocols: protocols,
		Blacklist: eth.blacklist,
		NAT:       p2p.UPNP(),
		NoDial:    !config.Dial,
	}

	if len(config.Port) > 0 {
		eth.net.ListenAddr = ":" + config.Port
	}

	return eth, nil
}

func (s *Thelonious) KeyManager() *crypto.KeyManager {
	return s.keyManager
}

func (s *Thelonious) Logger() ethlogger.LogSystem {
	return s.logger
}

func (s *Thelonious) ClientIdentity() p2p.ClientIdentity {
	return s.clientIdentity
}

func (s *Thelonious) ChainManager() *core.ChainManager {
	return s.chainManager
}

func (s *Thelonious) BlockProcessor() *core.BlockProcessor {
	return s.blockProcessor
}

func (s *Thelonious) TxPool() *core.TxPool {
	return s.txPool
}

func (s *Thelonious) BlockPool() *BlockPool {
	return s.blockPool
}

func (s *Thelonious) Whisper() *whisper.Whisper {
	return s.whisper
}

func (s *Thelonious) EventMux() *event.TypeMux {
	return s.eventMux
}
func (self *Thelonious) Db() thelutil.Database {
	return self.db
}

func (s *Thelonious) IsMining() bool {
	return s.Mining
}

func (s *Thelonious) IsListening() bool {
	// XXX TODO
	return false
}

func (s *Thelonious) PeerCount() int {
	return s.net.PeerCount()
}

func (s *Thelonious) Peers() []*p2p.Peer {
	return s.net.Peers()
}

func (s *Thelonious) MaxPeers() int {
	return s.net.MaxPeers
}

// Start the ethereum
func (s *Thelonious) Start(seed bool) error {
	err := s.net.Start()
	if err != nil {
		return err
	}

	// Start services
	s.txPool.Start()
	s.blockPool.Start()

	if s.whisper != nil {
		s.whisper.Start()
	}

	// broadcast transactions
	s.txSub = s.eventMux.Subscribe(core.TxPreEvent{})
	go s.txBroadcastLoop()

	// broadcast mined blocks
	s.blockSub = s.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go s.blockBroadcastLoop()

	// TODO: read peers here
	if seed {
		logger.Infof("Connect to seed node %v", seedNodeAddress)
		if err := s.SuggestPeer(seedNodeAddress); err != nil {
			return err
		}
	}

	logger.Infoln("Server started")
	return nil
}

func (self *Thelonious) SuggestPeer(addr string) error {
	netaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logger.Errorf("couldn't resolve %s:", addr, err)
		return err
	}

	self.net.SuggestPeer(netaddr.IP, netaddr.Port, nil)
	return nil
}

func (s *Thelonious) Stop() {
	// Close the database
	defer s.db.Close()

	close(s.quit)

	s.txSub.Unsubscribe()    // quits txBroadcastLoop
	s.blockSub.Unsubscribe() // quits blockBroadcastLoop

	if s.RpcServer != nil {
		s.RpcServer.Stop()
	}
	s.txPool.Stop()
	s.eventMux.Stop()
	s.blockPool.Stop()
	if s.whisper != nil {
		s.whisper.Stop()
	}

	logger.Infoln("Server stopped")
	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Thelonious) WaitForShutdown() {
	<-s.shutdownChan
}

// now tx broadcasting is taken out of txPool
// handled here via subscription, efficiency?
func (self *Thelonious) txBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.txSub.Chan() {
		event := obj.(core.TxPreEvent)
		self.net.Broadcast("eth", TxMsg, event.Tx.RlpData())
	}
}

func (self *Thelonious) blockBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.blockSub.Chan() {
		switch ev := obj.(type) {
		case core.NewMinedBlockEvent:
			self.net.Broadcast("eth", NewBlockMsg, ev.Block.RlpData(), ev.Block.Td)
		}
	}
}

func saveProtocolVersion(db thelutil.Database) {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protocolVersion := thelutil.NewValue(d).Uint()

	if protocolVersion == 0 {
		db.Put([]byte("ProtocolVersion"), thelutil.NewValue(ProtocolVersion).Bytes())
	}
}
