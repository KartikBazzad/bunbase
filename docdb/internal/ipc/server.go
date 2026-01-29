package ipc

import (
	"net"
	"os"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/pool"
	"github.com/panjf2000/ants/v2"
)

type Server struct {
	cfg         *config.Config
	logger      *logger.Logger
	pool        *pool.Pool
	handler     *Handler
	listener    net.Listener
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
	connections map[net.Conn]bool
	connMu      sync.Mutex
	connPool    *ants.Pool // Optional: bounds concurrent connection handlers (nil = unlimited)
}

func NewServer(cfg *config.Config, log *logger.Logger) (*Server, error) {
	p := pool.NewPool(cfg, log)
	h := NewHandler(p, cfg, log) // Phase E.9: Pass config and logger for debug mode

	return &Server{
		cfg:         cfg,
		logger:      log,
		pool:        p,
		handler:     h,
		running:     false,
		connections: make(map[net.Conn]bool),
	}, nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if err := s.pool.Start(); err != nil {
		return err
	}

	if err := os.RemoveAll(s.cfg.IPC.SocketPath); err != nil {
		s.logger.Warn("Failed to remove old socket: %v", err)
	}

	listener, err := net.Listen("unix", s.cfg.IPC.SocketPath)
	if err != nil {
		return err
	}

	s.listener = listener
	s.running = true

	if s.cfg.IPC.MaxConnections > 0 {
		connPool, err := ants.NewPool(s.cfg.IPC.MaxConnections, ants.WithPanicHandler(func(v any) {
			s.logger.Error("IPC connection handler panic: %v", v)
		}))
		if err == nil {
			s.connPool = connPool
		}
	}

	s.logger.Info("IPC server listening on %s", s.cfg.IPC.SocketPath)

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.pool.Stop()
	s.running = false
	s.mu.Unlock()

	// Close all active connections to unblock any waiting reads
	s.connMu.Lock()
	for conn := range s.connections {
		conn.Close()
	}
	s.connMu.Unlock()

	s.wg.Wait()

	if s.connPool != nil {
		_ = s.connPool.ReleaseTimeout(3 * time.Second)
		s.connPool = nil
	}

	s.logger.Info("IPC server stopped")
	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			s.logger.Error("Accept error: %v", err)
			continue
		}

		s.connMu.Lock()
		s.connections[conn] = true
		s.connMu.Unlock()

		s.wg.Add(1)
		if s.connPool != nil {
			conn := conn
			if err := s.connPool.Submit(func() {
				defer s.wg.Done()
				s.handleConnection(conn)
			}); err != nil {
				// If submission fails, balance the WaitGroup and close connection
				s.wg.Done()
				conn.Close()
				s.connMu.Lock()
				delete(s.connections, conn)
				s.connMu.Unlock()
				s.logger.Error("Failed to submit connection handler to pool: %v", err)
			}
		} else {
			go func() {
				defer s.wg.Done()
				s.handleConnection(conn)
			}()
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
	}()

	s.logger.Debug("New connection from %s", conn.RemoteAddr())

	for {
		data, err := readFrame(conn)
		if err != nil {
			if err != net.ErrClosed {
				s.logger.Debug("Connection closed: %v", err)
			}
			return
		}

		frame, err := DecodeRequest(data)
		if err != nil {
			s.logger.Error("Failed to decode request: %v", err)
			continue
		}

		response := s.handler.Handle(frame)
		responseData, err := EncodeResponse(response)
		if err != nil {
			s.logger.Error("Failed to encode response: %v", err)
			continue
		}

		if err := writeFrame(conn, responseData); err != nil {
			s.logger.Error("Failed to write response: %v", err)
			return
		}
	}
}
