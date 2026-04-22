package p2p

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// P2P Server
// =============================================================================

// Config holds server configuration
type Config struct {
	ListenAddr      string        // Address to listen on (e.g., ":8333")
	MaxConnections int          // Maximum number of connections
	MaxInbound     int          // Maximum inbound connections
	MaxOutbound    int          // Maximum outbound connections
	ConnectionTimeout time.Duration // Connection timeout
	ReadTimeout    time.Duration // Read deadline
	WriteTimeout  time.Duration // Write deadline
	PingInterval   time.Duration // How often to ping peers
	BlockTimeout  time.Duration // Timeout for block requests
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		ListenAddr:        ":8333",
		MaxConnections:   125,
		MaxInbound:      100,
		MaxOutbound:     25,
		ConnectionTimeout: 30 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:   30 * time.Second,
		PingInterval:   2 * time.Minute,
		BlockTimeout:  3 * time.Minute,
	}
}

// Server is the P2P server that accepts connections
type Server struct {
	cfg Config

	// Local node identity
	localNode *LocalNode

	// Network
	listener net.Listener
	quit     chan struct{}
	wg       sync.WaitGroup

	// Connection management
	connCh   chan *ConnInfo // New connections channel
	peers    *PeerPool
	inbound  int32 // Current inbound connections
	outbound int32 // Current outbound connections

	// Services
	chain       ChainInterface
	chainSyncer *ChainSyncer
	dosDetector *DoSDetector
	onBlock     func(*Peer, []byte) // Block received callback
	onTX        func(*Peer, []byte) // Transaction received callback

	// State
	running atomic.Bool
	started atomic.Bool
}

// ConnInfo contains information about a new connection
type ConnInfo struct {
	Conn  net.Conn
	Dir   Direction
	NodeID NodeID // For outbound, the target; for inbound, will be set after handshake
	Err   error
}

// NewServer creates a new P2P server
func NewServer(cfg Config, chain ChainInterface) *Server {
	return &Server{
		cfg:    cfg,
		chain:  chain,
		peers:  NewPeerPool(),
		connCh: make(chan *ConnInfo, 10),
		quit:   make(chan struct{}),
	}
}

// NewServerWithServices creates a server with all services initialized
func NewServerWithServices(cfg Config, chain ChainInterface) *Server {
	s := &Server{
		cfg:    cfg,
		chain:  chain,
		peers:  NewPeerPool(),
		connCh: make(chan *ConnInfo, 10),
		quit:   make(chan struct{}),
	}

	// Initialize chain syncer
	if chain != nil {
		s.chainSyncer = NewChainSyncer(nil, chain)
	}

	// Initialize DoS detector
	s.dosDetector = NewDoSDetector(DefaultDoSConfig())

	return s
}

// Start starts the P2P server
func (s *Server) Start(localNode *LocalNode) error {
	if s.running.Swap(true) {
		return fmt.Errorf("server already running")
	}

	s.localNode = localNode

	// Create listener
	var err error
	s.listener, err = net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		s.running.Store(false)
		return fmt.Errorf("listen on %s: %w", s.cfg.ListenAddr, err)
	}

	// Start accept loop
	s.wg.Add(1)
	go s.acceptLoop()

	// Start connection handler
	s.wg.Add(1)
	go s.handleConnections()

	s.started.Store(true)
	fmt.Printf("[p2p] Server listening on %s\n", s.cfg.ListenAddr)
	return nil
}

// Stop stops the P2P server
func (s *Server) Stop() {
	if !s.running.Swap(false) {
		return
	}

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close quit channel
	close(s.quit)

	// Disconnect all peers
	s.peers.DisconnectAll()

	// Wait for goroutines
	s.wg.Wait()
	s.started.Store(false)
	fmt.Println("[p2p] Server stopped")
}

// AcceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		// Set accept deadline
		// TODO: SetDeadline on listener(time.Now().Add(s.cfg.ConnectionTimeout))

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
			}

			// Log non-temporary errors
			if opErr, ok := err.(net.Error); !ok || !opErr.Temporary() {
				fmt.Printf("[p2p] Accept error: %v\n", err)
				time.Sleep(time.Second)
			}
			continue
		}

		// Check inbound limit
		if atomic.LoadInt32(&s.inbound) >= int32(s.cfg.MaxInbound) {
			conn.Close()
			continue
		}

		// Handle connection in goroutine
		s.wg.Add(1)
		go func() {
			s.handleInbound(conn)
			s.wg.Done()
		}()
	}
}

// handleInbound handles an incoming connection
func (s *Server) handleInbound(conn net.Conn) {
	addr := conn.RemoteAddr().String()

	// Check DoS protection
	if s.dosDetector != nil {
		allowed, reason := s.dosDetector.CheckConnection(addr)
		if !allowed {
			fmt.Printf("[p2p] Connection rejected from %s: %s\n", addr, reason)
			conn.Close()
			return
		}
	}

	atomic.AddInt32(&s.inbound, 1)
	defer atomic.AddInt32(&s.inbound, -1)

	fmt.Printf("[p2p] Inbound connection from %s\n", addr)

	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))

	// Send connection info to handler (NodeID set after handshake)
	select {
	case s.connCh <- &ConnInfo{Conn: conn, Dir: DirInbound}:
	case <-s.quit:
		conn.Close()
	}
}

// ConnectToPeer initiates an outbound connection
func (s *Server) ConnectToPeer(addr string, nodeID NodeID) error {
	// Check outbound limit
	if atomic.LoadInt32(&s.outbound) >= int32(s.cfg.MaxOutbound) {
		return fmt.Errorf("max outbound connections reached")
	}

	// Check if already connected
	if _, ok := s.peers.GetByAddr(addr); ok {
		return fmt.Errorf("already connected to %s", addr)
	}

	// Dial
	conn, err := net.DialTimeout("tcp", addr, s.cfg.ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	atomic.AddInt32(&s.outbound, 1)

	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))

	fmt.Printf("[p2p] Outbound connection to %s\n", addr)

	// Send to handler (peer created after handshake)
	select {
	case s.connCh <- &ConnInfo{Conn: conn, Dir: DirOutbound, NodeID: nodeID}:
	case <-s.quit:
		conn.Close()
		atomic.AddInt32(&s.outbound, -1)
		return fmt.Errorf("server shutting down")
	}

	return nil
}

// handleConnections processes new connections
func (s *Server) handleConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		case info := <-s.connCh:
			s.processConnection(info)
		}
	}
}

// processConnection processes a new connection
func (s *Server) processConnection(info *ConnInfo) {
	peer := NewPeer(info.Conn, info.Dir, info.NodeID)

	// Perform handshake
	if err := s.doHandshake(peer); err != nil {
		fmt.Printf("[p2p] Handshake failed with %s: %v\n", peer.RemoteAddress(), err)
		info.Conn.Close()
		return
	}

	// Add to peer pool
	if err := s.peers.Add(peer); err != nil {
		fmt.Printf("[p2p] Failed to add peer %s: %v\n", peer.RemoteAddress(), err)
		info.Conn.Close()
		return
	}

	// Decrement outbound counter on success
	if info.Dir == DirOutbound {
		atomic.AddInt32(&s.outbound, -1)
	}

	// Start message handlers
	go s.readLoop(peer)
	go s.writeLoop(peer)

	// Trigger chain sync if needed
	if s.onBlock != nil {
		s.requestChain(peer)
	}
}

// doHandshake performs the version handshake
func (s *Server) doHandshake(p *Peer) error {
	// Get current chain height
	bestHeight := 0
	if s.chain != nil {
		bestHeight = s.chain.GetBestHeight()
	}

	if p.Direction == DirOutbound {
		// Send version
		version := &MessageVersion{
			Version:     ProtocolVersion,
			Services:   1, // Full node
			Timestamp:  time.Now().Unix(),
			BestHeight: bestHeight,
			ID:         s.localNode.ID,
		}

		if err := WriteMessage(p.GetConnection(), &Message{Type: MsgVersion, Payload: version.Serialize()}); err != nil {
			return fmt.Errorf("send version: %w", err)
		}
		p.SetState(StateHandshake)

		// Read version
		msg, err := ReadMessage(p.GetConnection())
		if err != nil {
			return fmt.Errorf("read version: %w", err)
		}
		if msg.Type != MsgVersion {
			return fmt.Errorf("expected version, got %s", msg.Type)
		}

		peerVersion := DeserializeMessageVersion(msg.Payload)
		p.SetVersion(peerVersion)

		// Send verack
		if err := WriteMessage(p.GetConnection(), &Message{Type: MsgVerAck}); err != nil {
			return fmt.Errorf("send verack: %w", err)
		}

		// Read verack
		msg, err = ReadMessage(p.GetConnection())
		if err != nil {
			return fmt.Errorf("read verack: %w", err)
		}
		if msg.Type != MsgVerAck {
			return fmt.Errorf("expected verack, got %s", msg.Type)
		}
	} else {
		// Inbound: read version first
		msg, err := ReadMessage(p.GetConnection())
		if err != nil {
			return fmt.Errorf("read version: %w", err)
		}
		if msg.Type != MsgVersion {
			return fmt.Errorf("expected version, got %s", msg.Type)
		}

		peerVersion := DeserializeMessageVersion(msg.Payload)
		p.SetVersion(peerVersion)

		// Get current chain height
		bestHeight := 0
		if s.chain != nil {
			bestHeight = s.chain.GetBestHeight()
		}

		// Send version
		version := &MessageVersion{
			Version:     ProtocolVersion,
			Services:   1,
			Timestamp:  time.Now().Unix(),
			BestHeight: bestHeight,
			ID:         s.localNode.ID,
		}

		if err := WriteMessage(p.GetConnection(), &Message{Type: MsgVersion, Payload: version.Serialize()}); err != nil {
			return fmt.Errorf("send version: %w", err)
		}
		p.SetState(StateHandshake)

		// Send verack
		if err := WriteMessage(p.GetConnection(), &Message{Type: MsgVerAck}); err != nil {
			return fmt.Errorf("send verack: %w", err)
		}

		// Read verack
		msg, err = ReadMessage(p.GetConnection())
		if err != nil {
			return fmt.Errorf("read verack: %w", err)
		}
		if msg.Type != MsgVerAck {
			return fmt.Errorf("expected verack, got %s", msg.Type)
		}
	}

	p.SetState(StateReady)
	return nil
}

// requestChain requests the chain from a peer
func (s *Server) requestChain(p *Peer) {
	// TODO: Implement chain synchronization
}

// readLoop reads messages from a peer
func (s *Server) readLoop(p *Peer) {
	p.wg.Add(1)
	defer p.wg.Done()

	for {
		select {
		case <-p.quit:
			return
		default:
		}

		// Set deadline
		p.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout))

		msg, err := ReadMessage(p.GetConnection())
		if err != nil {
			fmt.Printf("[p2p] Read error from %s: %v\n", p.RemoteAddress(), err)
			p.Close()
			return
		}

		p.AddBytesRecv(int64(len(msg.Payload)))

		// Handle message
		s.handleMessage(p, msg)
	}
}

// writeLoop writes messages to a peer
func (s *Server) writeLoop(p *Peer) {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.quit:
			return
		case msg := <-p.MessageChan():
			if err := WriteMessage(p.GetConnection(), msg); err != nil {
				fmt.Printf("[p2p] Write error to %s: %v\n", p.RemoteAddress(), err)
				p.Close()
				return
			}
			p.AddBytesSent(int64(len(msg.Payload)))
		case <-ticker.C:
			// Send periodic ping
			msg := &Message{Type: MsgPing}
			if err := WriteMessage(p.GetConnection(), msg); err != nil {
				fmt.Printf("[p2p] Ping error to %s: %v\n", p.RemoteAddress(), err)
				p.Close()
				return
			}
		}
	}
}

// handleMessage handles an incoming message
func (s *Server) handleMessage(p *Peer, msg *Message) {
	switch msg.Type {
	case MsgVersion:
		// Already handled in handshake
	case MsgVerAck:
		// Already handled in handshake
	case MsgPing:
		// Respond with pong (include nonce for latency measurement)
		nonce := make([]byte, 8)
		if len(msg.Payload) >= 8 {
			copy(nonce, msg.Payload[:8])
		}
		WriteMessage(p.GetConnection(), &Message{Type: MsgPong, Payload: nonce})
	case MsgPong:
		// Update ping latency
		p.UpdateLastPing()
	case MsgGetAddr:
		// Send our known addresses (from peer pool)
		s.handleGetAddr(p)
	case MsgAddr:
		// Handle address advertisement
		s.handleAddr(p, msg.Payload)
	case MsgInv:
		// Handle inventory - request unknown blocks/txs
		s.handleInv(p, msg.Payload)
	case MsgGetData:
		// Handle request for blocks/transactions
		s.handleGetData(p, msg.Payload)
	case MsgNotFound:
		// Handle requested item not found
		s.handleNotFound(p, msg.Payload)
	case MsgGetBlocks:
		// Handle getblocks request
		if s.chainSyncer != nil {
			s.chainSyncer.HandleGetBlocks(p, msg)
		}
	case MsgGetHeaders:
		// Handle getheaders request
		if s.chainSyncer != nil {
			s.chainSyncer.HandleGetHeaders(p, msg)
		}
	case MsgBlock:
		// Handle block
		if s.onBlock != nil {
			s.onBlock(p, msg.Payload)
		}
		// Also handle via chain syncer if in IBD
		if s.chainSyncer != nil {
			s.chainSyncer.HandleBlock(p, msg.Payload)
		}
	case MsgTX:
		// Handle transaction
		if s.onTX != nil {
			s.onTX(p, msg.Payload)
		}
	case MsgMemPool:
		// Handle mempool request
		s.handleMemPool(p)
	default:
		fmt.Printf("[p2p] Unknown message type: %s from %s\n", msg.Type, p.RemoteAddress())
	}
}

// handleGetAddr handles getaddr request
func (s *Server) handleGetAddr(p *Peer) {
	// Get all known peers from peer pool
	peers := s.peers.List()

	// Build address message
	addrMsg := &AddrMessage{
		Count:     int64(len(peers)),
		AddrList:  make([]AddrEntry, 0, len(peers)),
	}

	for _, peer := range peers {
		addrMsg.AddrList = append(addrMsg.AddrList, AddrEntry{
			Timestamp: peer.connectedAt,
			Services:  peer.Services,
			Addr:      peer.RemoteAddress(),
		})
	}

	response := &Message{
		Type:    MsgAddr,
		Payload: addrMsg.Serialize(),
	}

	select {
	case p.MessageChan() <- response:
	default:
	}
}

// handleAddr handles addr message
func (s *Server) handleAddr(p *Peer, payload []byte) {
	addrMsg := DeserializeAddrMessage(payload)
	if addrMsg == nil {
		return
	}

	// Process new addresses (simplified - in production would add to discovery)
	for _, entry := range addrMsg.AddrList {
		fmt.Printf("[p2p] Received address: %s\n", entry.Addr)
	}
}

// handleInv handles inventory message
func (s *Server) handleInv(p *Peer, payload []byte) {
	inv := DeserializeInvMessage(payload)
	if inv == nil {
		return
	}

	// Filter to unknown items and request them
	var blockHashes [][]byte
	var txHashes [][]byte

	for _, v := range inv.Vectors {
		hash := make([]byte, 32)
		copy(hash, v.Hash[:])

		if v.Type == InvTypeBlock {
			if !p.IsBlockKnown(hash) {
				p.MarkBlockKnown(hash)
				blockHashes = append(blockHashes, hash)
			}
		} else if v.Type == InvTypeTX {
			if !p.IsTxKnown(hash) {
				p.MarkTxKnown(hash)
				txHashes = append(txHashes, hash)
			}
		}
	}

	// Request blocks
	if len(blockHashes) > 0 && s.chainSyncer != nil {
		s.chainSyncer.AddBlockRequest(blockHashes, p)
	}

	// Request transactions
	if len(txHashes) > 0 {
		msg := &Message{
			Type:    MsgGetData,
			Payload: s.serializeGetDataRequest(txHashes, 1), // 1 = TX type
		}
		select {
		case p.MessageChan() <- msg:
		default:
		}
	}
}

// handleGetData handles getdata request
func (s *Server) handleGetData(p *Peer, payload []byte) {
	// Deserialize getdata request
	if len(payload) < 1 {
		return
	}

	count := int(payload[0])
	if count*33+1 > len(payload) {
		return
	}

	var blockHashes [][]byte
	offset := 1

	for i := 0; i < count; i++ {
		itemType := payload[offset]
		hash := make([]byte, 32)
		copy(hash, payload[offset+1:offset+33])

		if itemType == InvTypeBlock {
			blockHashes = append(blockHashes, hash)
		}
		// TODO: Handle TX requests

		offset += 33
	}

	// Send blocks
	for _, hash := range blockHashes {
		if s.chain != nil {
			block, err := s.chain.GetBlock(hash)
			if err == nil && block != nil {
				msg := &Message{
					Type:    MsgBlock,
					Payload: block.Serialize(),
				}
				select {
				case p.MessageChan() <- msg:
				default:
				}
			}
		}
	}
}

// handleNotFound handles notfound message
func (s *Server) handleNotFound(p *Peer, payload []byte) {
	fmt.Printf("[p2p] Peer %s could not fulfill getdata request\n", p.RemoteAddress())
}

// handleMemPool handles mempool request
func (s *Server) handleMemPool(p *Peer) {
	// TODO: Return transactions in mempool
	// For now, send empty inventory
	inv := NewInvMessage()

	msg := &Message{
		Type:    MsgInv,
		Payload: inv.Serialize(),
	}

	select {
	case p.MessageChan() <- msg:
	default:
	}
}

// serializeGetDataRequest creates getdata payload
func (s *Server) serializeGetDataRequest(hashes [][]byte, itemType uint8) []byte {
	count := len(hashes)
	size := 1 + count*33
	buf := make([]byte, size)

	buf[0] = byte(count)
	offset := 1

	for _, hash := range hashes {
		buf[offset] = itemType
		offset++
		copy(buf[offset:offset+32], hash)
		offset += 32
	}

	return buf
}

// =============================================================================
// Public Interface
// =============================================================================

// Broadcast sends a message to all peers
func (s *Server) Broadcast(msgType MessageType, payload []byte) {
	msg := &Message{Type: msgType, Payload: payload}
	s.peers.Broadcast(msg, nil)
}

// SendTo sends a message to a specific peer
func (s *Server) SendTo(nodeID NodeID, msgType MessageType, payload []byte) error {
	p, ok := s.peers.Get(nodeID)
	if !ok {
		return fmt.Errorf("peer not found")
	}

	msg := &Message{Type: msgType, Payload: payload}
	select {
	case p.MessageChan() <- msg:
		return nil
	default:
		return fmt.Errorf("channel full")
	}
}

// Peers returns the peer pool
func (s *Server) Peers() *PeerPool {
	return s.peers
}

// IsRunning returns the server state
func (s *Server) IsRunning() bool {
	return s.running.Load()
}

// Addr returns the listening address
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}