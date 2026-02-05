package ipc

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"

	"github.com/kartikbazzad/bunbase/buncast/internal/broker"
	"github.com/kartikbazzad/bunbase/buncast/internal/config"
	"github.com/kartikbazzad/bunbase/buncast/internal/logger"
)

const frameLenSize = 4

// Server is the Buncast IPC server (Unix socket).
type Server struct {
	cfg         *config.Config
	logger      *logger.Logger
	broker      *broker.Broker
	handler     *Handler
	listener    net.Listener
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
	connections map[net.Conn]bool
	connMu      sync.Mutex
}

// NewServer creates a new IPC server.
func NewServer(cfg *config.Config, log *logger.Logger, b *broker.Broker) (*Server, error) {
	h := NewHandler(b, cfg, log)
	return &Server{
		cfg:         cfg,
		logger:      log,
		broker:      b,
		handler:     h,
		running:     false,
		connections: make(map[net.Conn]bool),
	}, nil
}

// Start starts the IPC server (Unix socket).
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
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

	s.logger.Info("IPC server listening on %s", s.cfg.IPC.SocketPath)
	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

// Stop stops the IPC server and closes all connections.
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
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

func readFrame(conn io.Reader) ([]byte, error) {
	lenBuf := make([]byte, frameLenSize)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lenBuf)
	if length > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(conn io.Writer, data []byte) error {
	lenBuf := make([]byte, frameLenSize)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}

func (s *Server) handleConnection(conn net.Conn) {
	var activeSession *SubscribeSession
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("connection handler panic: %v", r)
		}
		// Unsubscribe if there's an active subscription session
		if activeSession != nil {
			s.broker.Unsubscribe(activeSession.Topic, activeSession.Sub)
		}
		conn.Close()
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
	}()

	for {
		data, err := readFrame(conn)
		if err != nil {
			if err != io.EOF && err != net.ErrClosed {
				s.logger.Debug("Connection read: %v", err)
			}
			return
		}

		req, err := DecodeRequest(data)
		if err != nil {
			s.logger.Error("Failed to decode request: %v", err)
			resp := &ResponseFrame{RequestID: 0, Status: StatusError, Payload: ErrorPayload(err.Error())}
			if enc, e := EncodeResponse(resp); e == nil {
				_ = writeFrame(conn, enc)
			}
			continue
		}

		resp, session, err := s.handler.Handle(conn, req)
		if err != nil {
			s.logger.Error("Handle error: %v", err)
			continue
		}

		respData, err := EncodeResponse(resp)
		if err != nil {
			s.logger.Error("Failed to encode response: %v", err)
			continue
		}
		if err := writeFrame(conn, respData); err != nil {
			s.logger.Error("Failed to write response: %v", err)
			return
		}

		// Subscribe: connection stays open; broker writes messages via connSubscriber.Send()
		// DO NOT read from connection here - it will consume messages meant for the client!
		// The broker writes messages directly to conn, and the client reads them.
		// We need to keep the handler running (don't return) but stop reading frames.
		// We'll wait for connection close by monitoring write errors in connSubscriber.Send()
		if session != nil {
			activeSession = session
			// Don't return - keep the handler alive so the connection stays open
			// Stop reading frames - the client reads messages, we just monitor for close
			// Wait for close signal from connSubscriber when write fails
			<-session.CloseChan
			return
		}
	}
}
