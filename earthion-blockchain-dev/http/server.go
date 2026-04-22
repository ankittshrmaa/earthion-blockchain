package http

import (
	"fmt"
	"net/http"
	"strings"

	"earthion/core"
	"earthion/wallet"
)

// Server is the HTTP API server
type Server struct {
	addr   string
	handler *Handler
	mux    *http.ServeMux
	server *http.Server
}

// NewServer creates a new HTTP server
func NewServer(addr string, bc *core.Blockchain, wal *wallet.Wallet) *Server {
	s := &Server{
		addr:   addr,
		handler: NewHandler(bc, wal),
		mux:    http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// Chain endpoints
	s.mux.HandleFunc("/api/chain/height", s.handler.ChainHeight)
	s.mux.HandleFunc("/api/chain/validate", s.handler.Validate)
	s.mux.HandleFunc("/api/chain/utxo", s.handler.UTXO)

	// Block endpoints
	s.mux.HandleFunc("/api/blocks", s.handler.GetBlocks)
	s.mux.HandleFunc("/api/blocks/", s.handleBlockByHash)
	s.mux.HandleFunc("/api/blocks/index/", s.handleBlockByIndex)

	// Wallet endpoints
	s.mux.HandleFunc("/api/wallet/address", s.handler.GetAddress)
	s.mux.HandleFunc("/api/wallet/balance", s.handler.GetBalance)
	s.mux.HandleFunc("/api/wallet/send", s.handler.Send)

	// Mining endpoints
	s.mux.HandleFunc("/api/mining/mine", s.handler.Mine)
	s.mux.HandleFunc("/api/mining/reward", s.handler.GetReward)

	// Stats
	s.mux.HandleFunc("/api/stats", s.handler.Stats)

	// Health
	s.mux.HandleFunc("/health", s.handler.Health)

	// Root
	s.mux.HandleFunc("/", s.root)
}

// handleBlockByHash routes to get block by hash
func (s *Server) handleBlockByHash(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/blocks/")
	if path == "" || path == "/" {
		http.Error(w, "missing hash", http.StatusBadRequest)
		return
	}
	s.handler.GetBlockByHash(w, r)
}

// handleBlockByIndex routes to get block by index
func (s *Server) handleBlockByIndex(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/blocks/index/")
	if path == "" || path == "/" {
		http.Error(w, "missing index", http.StatusBadRequest)
		return
	}
	s.handler.GetBlockByIndex(w, r)
}

// root is the root endpoint
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"Earthion HTTP API","version":"1.0.0","docs":"/docs"}`)
		return
	}
	http.NotFound(w, r)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}
	fmt.Printf("[http] Server listening on %s\n", s.addr)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}