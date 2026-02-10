package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/kartikbazzad/bunbase/bunder-manager/internal/manager"
)

// ProxyKVRequest is the JSON payload for CmdProxyKV.
type ProxyKVRequest struct {
	Method    string `json:"method"`
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
	Body      string `json:"body"` // base64-encoded body (empty for GET)
}

// ProxyKVResponse is the JSON payload for CmdProxyKV response.
type ProxyKVResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"` // base64-encoded response body
	Error  string `json:"error,omitempty"`
}

// Server is the TCP RPC server for KV proxy (internal use by platform).
// It reuses the same HTTP mux as the main server so routing logic is shared.
type Server struct {
	addr     string
	manager  *manager.InstanceManager
	ln       net.Listener
	wg       sync.WaitGroup
	quit     chan struct{}
	httpMux  *http.ServeMux
}

// NewServer creates a new RPC server. httpMux is the same mux used for HTTP (e.g. KV routes).
func NewServer(addr string, m *manager.InstanceManager, httpMux *http.ServeMux) *Server {
	return &Server{
		addr:    addr,
		manager: m,
		httpMux: httpMux,
		quit:    make(chan struct{}),
	}
}

// Start starts the TCP listener.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("[Bunder RPC] TCP server listening on %s", s.addr)
	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	close(s.quit)
	if s.ln != nil {
		s.ln.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("[Bunder RPC] Accept error: %v", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(conn)
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		data, err := readLengthPrefixed(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("[Bunder RPC] Read error: %v", err)
			}
			return
		}
		req, err := DecodeRequest(data)
		if err != nil {
			log.Printf("[Bunder RPC] Decode request error: %v", err)
			return
		}
		resp := s.handleRequest(req)
		respData, err := EncodeResponse(resp)
		if err != nil {
			log.Printf("[Bunder RPC] Encode response error: %v", err)
			return
		}
		if err := writeLengthPrefixed(conn, respData); err != nil {
			log.Printf("[Bunder RPC] Write error: %v", err)
			return
		}
	}
}

func (s *Server) handleRequest(req *RequestFrame) *ResponseFrame {
	resp := &ResponseFrame{RequestID: req.RequestID}
	if req.Command != CmdProxyKV {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"unknown command %d"}`, req.Command))
		return resp
	}
	var proxyReq ProxyKVRequest
	if err := json.Unmarshal(req.Payload, &proxyReq); err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"invalid payload: %v"}`, err))
		return resp
	}
	var body []byte
	if proxyReq.Body != "" {
		body, _ = base64.StdEncoding.DecodeString(proxyReq.Body)
	}
	// Construct URL path: /kv/{projectID}{path}
	// path should already include /kv/{key} or /keys or /health
	urlStr := "http://localhost/kv/" + proxyReq.ProjectID + proxyReq.Path
	httpReq := httptest.NewRequest(proxyReq.Method, urlStr, bytes.NewReader(body))
	if len(body) > 0 {
		httpReq.Header.Set("Content-Type", "application/octet-stream")
	}
	rec := httptest.NewRecorder()
	s.httpMux.ServeHTTP(rec, httpReq)
	status := rec.Code
	respBody := rec.Body.Bytes()
	resp.Status = StatusOK
	out, _ := json.Marshal(ProxyKVResponse{
		Status: status,
		Body:   base64.StdEncoding.EncodeToString(respBody),
	})
	resp.Payload = out
	return resp
}

func readLengthPrefixed(conn net.Conn) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lenBuf)
	if length > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}
	return data, nil
}

func writeLengthPrefixed(conn net.Conn, data []byte) error {
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}
