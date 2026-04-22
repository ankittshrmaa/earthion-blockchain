package p2p

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"earthion/core"
)

// =============================================================================
// Chain Synchronization
// =============================================================================

// ChainSyncer handles Initial Block Download (IBD) and ongoing sync
type ChainSyncer struct {
	localNode *LocalNode
	chain     ChainInterface

	// IBD state
	mu           sync.Mutex
	isIBD        bool
	ibdStarted   time.Time
	ibdPeer      *Peer
	knownBest   int      // Best height we've verified
	requestQueue chan *SyncRequest
	quit         chan struct{}

	// Parallel download
	parallelBlocks int // Number of blocks to request at once
}

// ChainInterface defines the interface for chain operations
type ChainInterface interface {
	GetBestHash() []byte
	GetBestHeight() int
	GetBlock(hash []byte) (*core.Block, error)
	AddBlock(block *core.Block) error
	GetBlockHash(height int) ([]byte, error)
	GetBlockHeight(hash []byte) (int, error)
}

// SyncRequest represents a block sync request
type SyncRequest struct {
	Hashes   [][]byte
	Priority int
	Peer     *Peer
}

// NewChainSyncer creates a new chain sync handler
func NewChainSyncer(local *LocalNode, chain ChainInterface) *ChainSyncer {
	return &ChainSyncer{
		localNode:       local,
		chain:         chain,
		requestQueue:  make(chan *SyncRequest, 100),
		quit:           make(chan struct{}),
		parallelBlocks: 16, // Request 16 blocks at a time
	}
}

// StartIBD starts Initial Block Download from a peer
func (cs *ChainSyncer) StartIBD(peer *Peer) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.isIBD {
		return // Already in IBD
	}

	cs.isIBD = true
	cs.ibdStarted = time.Now()
	cs.ibdPeer = peer

	fmt.Printf("[sync] Starting IBD from %s\n", peer.RemoteAddress())

	// Send getblocks message
	locators := cs.getBlockLocators()
	msg := &Message{
		Type:    MsgGetBlocks,
		Payload: (&GetBlocksMessage{
			Version:     uint32(ProtocolVersion),
			BlockLocators: locators,
		}).Serialize(),
	}

	select {
	case peer.MessageChan() <- msg:
	default:
	}

	// Start request processor
	go cs.processRequests()
}

// StopIBD stops IBD mode
func (cs *ChainSyncer) StopIBD() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.isIBD {
		return
	}

	cs.isIBD = false
	cs.ibdPeer = nil

	fmt.Printf("[sync] IBD complete. Took %v\n", time.Since(cs.ibdStarted))
}

// IsIBD returns if we're currently in IBD
func (cs *ChainSyncer) IsIBD() bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.isIBD
}

// HandleInv handles inventory message during sync
func (cs *ChainSyncer) HandleInv(peer *Peer, inv *InvMessage) error {
	if !cs.IsIBD() {
		return nil // Not syncing
	}

	// Check for blocks
	for _, v := range inv.Vectors {
		if v.Type == InvTypeBlock {
			// Queue block fetch
			hash := make([]byte, 32)
			copy(hash, v.Hash[:])

			cs.requestQueue <- &SyncRequest{
				Hashes:   [][]byte{hash},
				Priority: 1,
				Peer:     peer,
			}
		}
	}

	return nil
}

// HandleBlock handles an incoming block
func (cs *ChainSyncer) HandleBlock(peer *Peer, blockData []byte) error {
	// Deserialize block
	block, err := core.DeserializeBlock(blockData)
	if err != nil {
		return fmt.Errorf("deserialize block: %w", err)
	}

	// Skip verification during sync (already validated by sender)
	// In production, add proper validation here

	// Add to chain
	if err := cs.chain.AddBlock(block); err != nil {
		return fmt.Errorf("add block: %w", err)
	}

	cs.mu.Lock()
	cs.knownBest = block.Index
	cs.mu.Unlock()

	// Check if IBD is done
	if cs.IsIBD() && cs.chain.GetBestHeight() >= cs.knownBest {
		// More blocks may be needed - send another request
		cs.requestMoreBlocks(peer)
	}

	return nil
}

// requestMoreBlocks requests more blocks during IBD
func (cs *ChainSyncer) requestMoreBlocks(peer *Peer) {
	// Get locator
	locators := cs.getBlockLocators()

	msg := &Message{
		Type:    MsgGetBlocks,
		Payload: (&GetBlocksMessage{
			Version:     uint32(ProtocolVersion),
			BlockLocators: locators,
		}).Serialize(),
	}

	select {
	case peer.MessageChan() <- msg:
	default:
	}
}

// getBlockLocators returns block locators for sync
func (cs *ChainSyncer) getBlockLocators() [][]byte {
	height := cs.chain.GetBestHeight()
	locators := make([][]byte, 0)

	// Start from current tip and go back exponentially
	i := height
	for i >= 0 && len(locators) < 20 {
		if hash, err := cs.chain.GetBlockHash(i); err == nil {
			locators = append(locators, hash)
		} else {
			break
		}

		// Skip back exponentially
		if i > 10 {
			i -= 10
		} else if i > 1 {
			i -= 1
		} else {
			break
		}
	}

	// Add genesis hash
	genesis, _ := cs.chain.GetBlockHash(0)
	if genesis != nil {
		locators = append(locators, genesis)
	}

	return locators
}

// processRequests processes block download requests during IBD
func (cs *ChainSyncer) processRequests() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-cs.quit:
			return
		case req := <-cs.requestQueue:
			// Request blocks from peer
			if req.Peer == nil || req.Peer.GetState() != StateReady {
				continue
			}

			// Create GetData message for requested blocks
			msg := &Message{
				Type:    MsgGetData,
				Payload: cs.serializeGetDataRequest(req.Hashes),
			}

			select {
			case req.Peer.MessageChan() <- msg:
				fmt.Printf("[sync] Requesting %d blocks from %s\n", len(req.Hashes), req.Peer.RemoteAddress())
			default:
				// Channel full, re-queue
				cs.requestQueue <- req
			}
		case <-ticker.C:
			// Check timeout
			cs.mu.Lock()
			isIBD := cs.isIBD
			started := cs.ibdStarted
			cs.mu.Unlock()

			if isIBD && time.Since(started) > 10*time.Minute {
				fmt.Printf("[sync] IBD timeout after %v\n", time.Since(started))
				cs.StopIBD()
			}
		}
	}
}

// serializeGetDataRequest serializes a getdata request for blocks
func (cs *ChainSyncer) serializeGetDataRequest(hashes [][]byte) []byte {
	// Format: count (1 byte) + (type (1) + hash (32)) per item
	count := len(hashes)
	size := 1 + count*33
	buf := make([]byte, size)

	buf[0] = byte(count)
	offset := 1

	for _, hash := range hashes {
		buf[offset] = InvTypeBlock // Block inventory type
		offset++
		copy(buf[offset:offset+32], hash)
		offset += 32
	}

	return buf
}

// HandleGetBlocks handles getblocks request from peer
func (cs *ChainSyncer) HandleGetBlocks(peer *Peer, msg *Message) error {
	gbm := DeserializeGetBlocksMessage(msg.Payload)
	if gbm == nil {
		return fmt.Errorf("invalid getblocks message")
	}

	// Find blocks from locator and send inventory
	// Simplified: send blocks from our tip
	height := cs.chain.GetBestHeight()

	inv := NewInvMessage()

	// Send up to 500 block hashes (Bitcoin protocol limit)
	count := 500
	if height > count {
		startHeight := height - count
		for i := startHeight; i <= height; i++ {
			if hash, err := cs.chain.GetBlockHash(i); err == nil {
				inv.AddBlock(hash)
			}
		}
	} else {
		for i := 0; i <= height; i++ {
			if hash, err := cs.chain.GetBlockHash(i); err == nil {
				inv.AddBlock(hash)
			}
		}
	}

	response := &Message{
		Type:    MsgInv,
		Payload: inv.Serialize(),
	}

	select {
	case peer.MessageChan() <- response:
	default:
	}

	return nil
}

// HandleGetHeaders handles getheaders request from peer
func (cs *ChainSyncer) HandleGetHeaders(peer *Peer, msg *Message) error {
	gbm := DeserializeGetBlocksMessage(msg.Payload)
	if gbm == nil {
		return fmt.Errorf("invalid getheaders message")
	}

	// TODO: Implement proper header serialization
	// For now, return empty headers response
	return nil
}

// AddBlockRequest adds a block download request to the queue
func (cs *ChainSyncer) AddBlockRequest(hashes [][]byte, peer *Peer) {
	cs.requestQueue <- &SyncRequest{
		Hashes:   hashes,
		Priority: 1,
		Peer:     peer,
	}
}

// RequestQueue returns the request queue for external access
func (cs *ChainSyncer) RequestQueue() chan *SyncRequest {
	return cs.requestQueue
}

// Stop stops the chain syncer
func (cs *ChainSyncer) Stop() {
	close(cs.quit)
}

// =============================================================================
// Ongoing Sync (Headers-First)
// =============================================================================

// SyncManager manages chain synchronization
type SyncManager struct {
	localNode *LocalNode
	peers     *PeerPool
	chain     ChainInterface

	// State
	mu           sync.RWMutex
	syncing      bool
	syncPeer     *Peer
	lastInv      time.Time
	syncStart    time.Time
}

// NewSyncManager creates a new sync manager
func NewSyncManager(local *LocalNode, peers *PeerPool, chain ChainInterface) *SyncManager {
	return &SyncManager{
		localNode: local,
		peers:     peers,
		chain:     chain,
	}
}

// StartSync starts syncing with a peer
func (sm *SyncManager) StartSync(peer *Peer) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.syncing {
		return
	}

	sm.syncing = true
	sm.syncPeer = peer
	sm.syncStart = time.Now()

	// Request headers first
	locators := sm.getLocators()
	msg := &Message{
		Type:    MsgGetHeaders,
		Payload: (&GetHeadersMessage{
			Version:     uint32(ProtocolVersion),
			BlockLocators: locators,
		}).Serialize(),
	}

	select {
	case peer.MessageChan() <- msg:
	default:
	}

	fmt.Printf("[sync] Starting sync with %s\n", peer.RemoteAddress())
}

// HandleHeaders handles incoming headers
func (sm *SyncManager) HandleHeaders(peer *Peer, headers []*core.BlockHeader) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if peer != sm.syncPeer {
		return nil // Not from sync peer
	}

	// Process headers
	if len(headers) == 0 {
		// Sync complete
		sm.syncing = false
		sm.syncPeer = nil
		fmt.Printf("[sync] Headers sync complete\n")
		return nil
	}

	// Find common ancestor
	common := sm.findCommonAncestor(headers)
	if common == nil {
		// No common ancestor - potential reorganization
		return sm.handleReorg(peer, headers)
	}

	// Request blocks from common point
	return nil
}

// findCommonAncestor finds the common ancestor
func (sm *SyncManager) findCommonAncestor(headers []*core.BlockHeader) *core.BlockHeader {
	for _, h := range headers {
		existing, err := sm.chain.GetBlockHash(int(h.Index))
		if err == nil && hex.EncodeToString(existing) == hex.EncodeToString(h.PrevHash) {
			return h
		}
	}
	return nil
}

// handleReorg handles chain reorganization
func (sm *SyncManager) handleReorg(peer *Peer, headers []*core.BlockHeader) error {
	fmt.Printf("[sync] Chain reorganization detected\n")
	// TODO: Implement reorg handling
	return nil
}

// getLocators returns block locators
func (sm *SyncManager) getLocators() [][]byte {
	height := sm.chain.GetBestHeight()
	locators := make([][]byte, 0)

	// Simple: get last 10 block hashes
	for i := height; i >= height-10 && i >= 0; i-- {
		if hash, err := sm.chain.GetBlockHash(i); err == nil {
			locators = append(locators, hash)
		}
	}

	return locators
}

// AnnounceBlock announces a new block to the network
func (sm *SyncManager) AnnounceBlock(block *core.Block) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.syncing {
		return // Not syncing, broadcast to all
	}

	// Create inv message
	inv := NewInvMessage()
	inv.AddBlock(block.Hash)

	msg := &Message{
		Type:    MsgInv,
		Payload: inv.Serialize(),
	}

	// Broadcast except sync peer
	sm.peers.Broadcast(msg, sm.syncPeer)
}

// =============================================================================
// Compact Block Sync (BIP152)
// =============================================================================

// CompactBlock represents a compact block (BIP152)
type CompactBlock struct {
	Header       core.BlockHeader
	Nonce        uint64
	ShortIDs    []uint64
	NumPrefilled int
	Prefilled   []PrefilledTx
}

// PrefilledTx represents a prefilled transaction in compact block
type PrefilledTx struct {
	Index int
	Tx    *core.Transaction
}

// Serialize serializes compact block
func (cb *CompactBlock) Serialize() []byte {
	// Simplified - full impl per BIP152
	data := cb.Header.Serialize()
	// Add nonce, shortids, prefilled
	return data
}

// DeserializeCompactBlock deserializes compact block
func DeserializeCompactBlock(data []byte) (*CompactBlock, error) {
	// Simplified
	header, err := core.DeserializeBlockHeader(data[:80])
	if err != nil {
		return nil, err
	}

	return &CompactBlock{
		Header: header,
	}, nil
}