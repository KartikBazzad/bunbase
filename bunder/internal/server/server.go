package server

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/kartikbazzad/bunbase/bunder/internal/config"
	"github.com/kartikbazzad/bunbase/bunder/internal/data_structures"
	"github.com/kartikbazzad/bunbase/bunder/internal/pubsub"
	"github.com/kartikbazzad/bunbase/bunder/internal/ttl"
)

// Server is the Bunder daemon: opens KV store, TTL manager, pubsub; starts TCP (RESP) and HTTP listeners.
// Start() begins accepting connections; Close() stops listeners and closes the KV store.
type Server struct {
	cfg      *config.Config
	kv       *data_structures.KVStore
	ttl      *ttl.Manager
	pubsub   *pubsub.PubSubManager
	handler  *Handler
	listener net.Listener
	httpLn   net.Listener
	conns    atomic.Int64
	mu       sync.Mutex
	closed   bool
}

// NewServer creates a server: opens KV store at cfg.DataPath, optional TTL manager and Buncast pubsub.
// Call Start() to bind TCP and HTTP; call Close() to stop and close the KV store.
func NewServer(cfg *config.Config) (*Server, error) {
	kv, err := data_structures.OpenKVStore(cfg.DataPath, cfg.BufferPoolSize, cfg.Shards)
	if err != nil {
		return nil, fmt.Errorf("open kv store: %w", err)
	}
	var ttlMgr *ttl.Manager
	if cfg.TTLCheckInterval > 0 {
		ttlMgr = ttl.NewManager(func(key string) {
			_, _ = kv.Delete([]byte(key))
		}, cfg.TTLCheckInterval)
	}
	var pubsubMgr *pubsub.PubSubManager
	if cfg.BuncastEnabled {
		pubsubMgr = pubsub.NewPubSubManager(cfg.BuncastAddr, true)
	}
	handler := NewHandler(kv, ttlMgr, pubsubMgr)
	return &Server{
		cfg:     cfg,
		kv:      kv,
		ttl:     ttlMgr,
		pubsub:  pubsubMgr,
		handler: handler,
	}, nil
}

// Start binds the TCP (RESP) and HTTP listeners and begins accepting connections.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("server closed")
	}
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln
	go s.acceptLoop()
	if s.cfg.HTTPAddr != "" {
		httpLn, err := net.Listen("tcp", s.cfg.HTTPAddr)
		if err != nil {
			ln.Close()
			return fmt.Errorf("http listen: %w", err)
		}
		s.httpLn = httpLn
		httpHandler := NewHTTPHandler(s.handler)
		go http.Serve(httpLn, httpHandler)
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed {
				return
			}
			continue
		}
		if s.conns.Load() >= int64(s.cfg.MaxClients) {
			conn.Close()
			continue
		}
		s.conns.Add(1)
		go func() {
			defer s.conns.Add(-1)
			s.handler.HandleConnection(conn)
		}()
	}
}

// Close stops the TCP/HTTP listeners, TTL sweeper, pubsub, and closes the KV store.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.httpLn != nil {
		s.httpLn.Close()
		s.httpLn = nil
	}
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
	if s.ttl != nil {
		s.ttl.Stop()
	}
	if s.pubsub != nil {
		s.pubsub.Close()
	}
	if s.kv != nil {
		return s.kv.Close()
	}
	return nil
}

// ConnCount returns the current number of connections.
func (s *Server) ConnCount() int64 {
	return s.conns.Load()
}

// Addr returns the TCP listener address (e.g. "127.0.0.1:6379") after Start. Empty if not listening.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}
