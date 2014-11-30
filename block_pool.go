package thelonious

import (
	"bytes"
	"container/list"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/thelonious/monkutil"
	//"github.com/eris-ltd/thelonious/monkwire"
)

var poollogger = monklog.NewLogger("BPOOL")

type block struct {
	from      *Peer
	peer      *Peer
	block     *monkchain.Block
	reqAt     time.Time
	requested int
}

type BlockPool struct {
	mut sync.Mutex

	eth *Thelonious

	hashPool [][]byte
	pool     map[string]*block

	td   *big.Int
	quit chan bool

	fetchingHashes    bool
	downloadStartedAt time.Time

	ChainLength, BlocksProcessed int

	peer *Peer
}

func NewBlockPool(eth *Thelonious) *BlockPool {
	return &BlockPool{
		eth:  eth,
		pool: make(map[string]*block),
		td:   monkutil.Big0,
		quit: make(chan bool),
	}
}

func (self *BlockPool) Len() int {
	return len(self.hashPool)
}

func (self *BlockPool) Reset() {
	self.mut.Lock()
	defer self.mut.Unlock()
	self.pool = make(map[string]*block)
	self.hashPool = nil
}

func (self *BlockPool) HasLatestHash() bool {
	self.mut.Lock()
	defer self.mut.Unlock()

	return self.pool[string(self.eth.ChainManager().CurrentBlock().Hash())] != nil
}

func (self *BlockPool) HasCommonHash(hash []byte) bool {
	return self.eth.ChainManager().GetBlock(hash) != nil
}

func (self *BlockPool) Blocks() (blocks monkchain.Blocks) {
	self.mut.Lock()
	defer self.mut.Unlock()
	for _, item := range self.pool {
		if item.block != nil {
			blocks = append(blocks, item.block)
		}
	}

	return
}

func (self *BlockPool) AddHash(hash []byte, peer *Peer) {
	self.mut.Lock()
	defer self.mut.Unlock()

	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{peer, nil, nil, time.Now(), 0}

		self.hashPool = append([][]byte{hash}, self.hashPool...)
	}
}

func (self *BlockPool) Add(b *monkchain.Block, peer *Peer) {
	self.mut.Lock()
	defer self.mut.Unlock()

	hash := string(b.Hash())

	// Note this doesn't check the working tree
	// Leave it to TestChain to ignore blocks already in forks
	// Also, we can one day use the information on which/howmany peers
	//  give us which blocks, in the td calculation. Hold on to your hats!
	if self.pool[hash] == nil && !self.eth.ChainManager().HasBlock(b.Hash()) {
		poollogger.Infof("Got unrequested block (%x...)\n", hash[0:4])

		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, peer, b, time.Now(), 0}

		if !self.eth.ChainManager().HasBlock(b.PrevHash) && self.pool[string(b.PrevHash)] == nil && !self.fetchingHashes {
			poollogger.Infof("Unknown block, requesting parent (%x...)\n", b.PrevHash[0:4])
			//peer.QueueMessage(monkwire.NewMessage(monkwire.MsgGetBlockHashesTy, []interface{}{b.Hash(), uint32(256)}))
		}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}

	self.BlocksProcessed++
}

func (self *BlockPool) Remove(hash []byte) {
	self.mut.Lock()
	defer self.mut.Unlock()

	self.hashPool = monkutil.DeleteFromByteSlice(self.hashPool, hash)
	delete(self.pool, string(hash))
}

func (self *BlockPool) ProcessCanonical(f func(block *monkchain.Block)) (procAmount int) {
	blocks := self.Blocks()

	monkchain.BlockBy(monkchain.Number).Sort(blocks)
	fmt.Println("Len block pool in process canonical: ", len(blocks))
	if len(blocks) > 0 {
		fmt.Println("first blocks num:", blocks[0].Number)
		fmt.Println("last blocks num:", blocks[len(blocks)-1].Number)
	} else {
		fmt.Println("no blocks in pool!")
	}
	for _, block := range blocks {
		if self.eth.ChainManager().HasBlock(block.PrevHash) {
			procAmount++

			f(block)

			self.Remove(block.Hash())
		} else {
			fmt.Println("not processed as we don't have prevhash")
		}

	}

	return
}

func (self *BlockPool) DistributeHashes() {
	self.mut.Lock()
	defer self.mut.Unlock()

	var (
		peerLen = self.eth.PeerCount()
		amount  = 256 * peerLen
		dist    = make(map[*Peer][][]byte)
	)

	num := int(math.Min(float64(amount), float64(len(self.pool))))
	for i, j := 0, 0; i < len(self.hashPool) && j < num; i++ {
		hash := self.hashPool[i]
		item := self.pool[string(hash)]

		if item != nil && item.block == nil {
			var peer *Peer
			lastFetchFailed := time.Since(item.reqAt) > 5*time.Second

			// Handle failed requests
			if lastFetchFailed && item.requested > 5 && item.peer != nil {
				if item.requested < 100 {
					// Select peer the hash was retrieved off
					peer = item.from
				} else {
					// Remove it
					self.hashPool = monkutil.DeleteFromByteSlice(self.hashPool, hash)
					delete(self.pool, string(hash))
				}
			} else if lastFetchFailed || item.peer == nil {
				// Find a suitable, available peer
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					if peer == nil && len(dist[p]) < amount/peerLen {
						peer = p
					}
				})
			}

			if peer != nil {
				item.reqAt = time.Now()
				item.peer = peer
				item.requested++

				dist[peer] = append(dist[peer], hash)
			}
		}
	}

	for peer, hashes := range dist {
		peer.FetchBlocks(hashes)
	}

	if len(dist) > 0 {
		self.downloadStartedAt = time.Now()
	}
}

func (self *BlockPool) Start() {
	go self.downloadThread()
	go self.chainThread()
}

func (self *BlockPool) Stop() {
	close(self.quit)
}

func (self *BlockPool) downloadThread() {
	serviceTimer := time.NewTicker(100 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-serviceTimer.C:
			// Check if we're catching up. If not distribute the hashes to
			// the peers and download the blockchain
			self.fetchingHashes = false
			self.eth.peerMut.Lock()
			eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
				if p.statusKnown && p.FetchingHashes() {
					self.fetchingHashes = true
				}
			})
			self.eth.peerMut.Unlock()

			self.DistributeHashes()

			self.mut.Lock()
			if self.ChainLength < len(self.hashPool) {
				self.ChainLength = len(self.hashPool)
			}
			self.mut.Unlock()
		}
	}
}

// Sort blocks in pool by number
// Find first with prevhash in canonical
// Find first consecutive chain
// TestChain (add blocks to workingTree, remove if any fail)
// InsertChain (add to canonical
//      or      sum difficulties of fork
//      and     possibly cause re-org
func (self *BlockPool) chainThread() {
	procTimer := time.NewTicker(500 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-procTimer.C:
			// We'd need to make sure that the pools are properly protected by a mutex
			blocks := self.Blocks()
			monkchain.BlockBy(monkchain.Number).Sort(blocks)

			// Find first block with prevhash in canonical
			for i, block := range blocks {
				if self.eth.ChainManager().HasBlock(block.PrevHash) {
					blocks = blocks[i:]
					break
				}
			}

			// Find first conescutive chain
			if len(blocks) > 0 {
				// Find chain of blocks
				if self.eth.ChainManager().HasBlock(blocks[0].PrevHash) {
					for i, block := range blocks[1:] {
						// NOTE: The Ith element in this loop refers to the previous block in
						// outer "blocks"
						if bytes.Compare(block.PrevHash, blocks[i].Hash()) != 0 {
							blocks = blocks[:i]
							break
						}
					}
				} else {
					blocks = nil
				}
			}

			// TODO figure out whether we were catching up
			// If caught up and just a new block has been propagated:
			// sm.eth.EventMux().Post(NewBlockEvent{block})
			// otherwise process and don't emit anything
			if len(blocks) > 0 {
				chainManager := self.eth.ChainManager()

				// sling blocks into a list
				bchain := monkchain.NewChain(blocks)
				// validate the chain
				_, err := chainManager.TestChain(bchain)

				// If validation failed, we flush the pool
				// and punish the peer
				if err != nil && !monkchain.IsTDError(err) {
					poollogger.Debugln(err)

					self.Reset()
					//self.punishPeer()
				} else {
					// Validation was successful
					// Sum-difficulties, insert chain
					// Possibly re-org
					chainManager.InsertChain(bchain)
					// Remove all blocks from pool
					for _, block := range blocks {
						self.Remove(block.Hash())
					}
				}
			}

			/* Do not propagate to the network on catchups
			if amount == 1 {
				block := self.eth.ChainManager().CurrentBlock
				self.eth.Broadcast(monkwire.MsgBlockTy, []interface{}{block.Value().Val})
			}*/
		}
	}
}

func (self *BlockPool) punishPeer() {
	/*
	                        TODO: fix this peer handling!
						if self.peer != nil && self.peer.conn != nil {
							poollogger.Debugf("Punishing peer for supplying bad chain (%v)\n", self.peer.conn.RemoteAddr())

						// This peer gave us bad hashes and made us fetch a bad chain, therefor he shall be punished.
						//self.eth.BlacklistPeer(self.peer)
						//self.peer.StopWithReason(DiscBadPeer)
	                        self.peer.Stop()
	                        self.td = monkutil.Big0
	                        self.peer = nil
						}*/

}
