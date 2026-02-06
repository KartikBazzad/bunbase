package rpc

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/api"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/core"
)

// GetSecretRequest is the JSON payload for CmdGetSecret.
type GetSecretRequest struct {
	Name string `json:"name"`
}

// GetSecretResponse is the JSON response for CmdGetSecret.
type GetSecretResponse struct {
	ValueB64 string `json:"value_b64"`
	Error    string `json:"error,omitempty"`
}

// PutSecretRequest is the JSON payload for CmdPutSecret.
type PutSecretRequest struct {
	Name     string `json:"name"`
	ValueB64 string `json:"value_b64"`
}

// PutSecretResponse is the JSON response for CmdPutSecret.
type PutSecretResponse struct {
	Error string `json:"error,omitempty"`
}

// Server is the TCP RPC server for KMS secrets (GetSecret / PutSecret).
type Server struct {
	addr    string
	secrets *core.SecretStore
	ln      net.Listener
	wg      sync.WaitGroup
	quit    chan struct{}
}

// NewServer creates a new RPC server backed by the given SecretStore.
func NewServer(addr string, secrets *core.SecretStore) *Server {
	return &Server{
		addr:    addr,
		secrets: secrets,
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
	log.Printf("[KMS RPC] TCP server listening on %s", s.addr)
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
				log.Printf("[KMS RPC] Accept error: %v", err)
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
				log.Printf("[KMS RPC] Read error: %v", err)
			}
			return
		}
		req, err := DecodeRequest(data)
		if err != nil {
			log.Printf("[KMS RPC] Decode request error: %v", err)
			return
		}
		resp := s.handleRequest(req)
		respData, err := EncodeResponse(resp)
		if err != nil {
			log.Printf("[KMS RPC] Encode response error: %v", err)
			return
		}
		if err := writeLengthPrefixed(conn, respData); err != nil {
			log.Printf("[KMS RPC] Write error: %v", err)
			return
		}
	}
}

func (s *Server) handleRequest(req *RequestFrame) *ResponseFrame {
	resp := &ResponseFrame{RequestID: req.RequestID}
	switch req.Command {
	case CmdGetSecret:
		var getReq GetSecretRequest
		if err := json.Unmarshal(req.Payload, &getReq); err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(GetSecretResponse{Error: "invalid payload: " + err.Error()})
			return resp
		}
		name := api.SanitizeKeyName(getReq.Name)
		if err := api.ValidateSecretName(name); err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(GetSecretResponse{Error: err.Error()})
			return resp
		}
		value, _, err := s.secrets.Get(name)
		if err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(GetSecretResponse{Error: err.Error()})
			return resp
		}
		resp.Status = StatusOK
		resp.Payload = mustMarshal(GetSecretResponse{ValueB64: base64.StdEncoding.EncodeToString(value)})
	case CmdPutSecret:
		var putReq PutSecretRequest
		if err := json.Unmarshal(req.Payload, &putReq); err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(PutSecretResponse{Error: "invalid payload: " + err.Error()})
			return resp
		}
		value, err := base64.StdEncoding.DecodeString(putReq.ValueB64)
		if err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(PutSecretResponse{Error: "invalid value_b64"})
			return resp
		}
		if err := api.ValidatePayloadSize(value); err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(PutSecretResponse{Error: err.Error()})
			return resp
		}
		name := api.SanitizeKeyName(putReq.Name)
		if err := api.ValidateSecretName(name); err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(PutSecretResponse{Error: err.Error()})
			return resp
		}
		_, err = s.secrets.Put(name, value)
		if err != nil {
			resp.Status = StatusError
			resp.Payload = mustMarshal(PutSecretResponse{Error: err.Error()})
			return resp
		}
		resp.Status = StatusOK
		resp.Payload = mustMarshal(PutSecretResponse{})
	default:
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"unknown command %d"}`, req.Command))
	}
	return resp
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
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
