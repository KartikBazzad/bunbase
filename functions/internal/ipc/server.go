package ipc

import (
	"net"
	"os"
	"sync"

	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

// Server provides Unix socket IPC for API server integration
type Server struct {
	cfg         *config.Config
	logger      *logger.Logger
	router      *router.Router
	scheduler   *scheduler.Scheduler
	handler     *Handler
	listener    net.Listener
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
	connections map[net.Conn]bool
	connMu      sync.Mutex
}

// NewServer creates a new IPC server
func NewServer(cfg *config.Config, r *router.Router, s *scheduler.Scheduler, log *logger.Logger) (*Server, error) {
	h := NewHandler(r, s, log)
	return &Server{
		cfg:         cfg,
		logger:      log,
		router:      r,
		scheduler:   s,
		handler:     h,
		running:     false,
		connections: make(map[net.Conn]bool),
	}, nil
}

// SetDependencies sets dependencies needed for function registration/deployment
func (s *Server) SetDependencies(meta *metadata.Store, workerScript string) {
	s.handler.SetDependencies(meta, s.cfg, workerScript)
}

// Start starts the IPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Remove old socket
	if err := os.RemoveAll(s.cfg.SocketPath); err != nil {
		s.logger.Warn("Failed to remove old socket: %v", err)
	}

	listener, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		return err
	}

	s.listener = listener
	s.running = true

	s.logger.Info("IPC server listening on %s", s.cfg.SocketPath)

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop stops the IPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.running = false
	s.mu.Unlock()

	// Close all active connections
	s.connMu.Lock()
	for conn := range s.connections {
		conn.Close()
	}
	s.connMu.Unlock()

	s.wg.Wait()

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
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		conn.Close()
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
	}()

	s.logger.Debug("New IPC connection from %s", conn.RemoteAddr())

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
