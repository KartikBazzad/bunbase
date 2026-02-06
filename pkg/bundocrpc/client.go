package bundocrpc

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Protocol matches bundoc-server/internal/rpc: length-prefixed frames,
// request (requestID, command, payload), response (requestID, status, payload).
const (
	cmdProxyDocument = 1
	statusOK         = 0
	statusError      = 1
	maxFrameSize     = 16 * 1024 * 1024
)

type proxyDocumentRequest struct {
	Method    string `json:"method"`
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
	Body      string `json:"body"`
}

type proxyDocumentResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error,omitempty"`
}

// Client is a TCP RPC client for the bundoc document proxy.
type Client struct {
	Addr      string
	conn      net.Conn
	mu        sync.Mutex
	requestID uint64
	Timeout   time.Duration
}

// New creates a new RPC client. addr is TCP address (e.g. "bundoc-data:9091").
func New(addr string) *Client {
	return &Client{
		Addr:    addr,
		Timeout: 10 * time.Second,
	}
}

func (c *Client) connectLocked() error {
	if c.conn != nil {
		if err := c.conn.SetDeadline(time.Time{}); err != nil {
			c.conn.Close()
			c.conn = nil
		} else {
			return nil
		}
	}
	conn, err := net.DialTimeout("tcp", c.Addr, c.Timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// ProxyRequest sends a document proxy request and returns status code, body, and error.
// path is the suffix after /v1/projects/{projectID}, e.g. /databases/(default)/documents/users.
func (c *Client) ProxyRequest(method, projectID, path string, body []byte) (status int, respBody []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return 0, nil, fmt.Errorf("connect: %w", err)
	}
	reqID := c.requestID
	c.requestID++

	bodyB64 := ""
	if len(body) > 0 {
		bodyB64 = base64.StdEncoding.EncodeToString(body)
	}
	payload, err := json.Marshal(proxyDocumentRequest{
		Method:    method,
		ProjectID: projectID,
		Path:      path,
		Body:      bodyB64,
	})
	if err != nil {
		return 0, nil, err
	}

	reqFrame := make([]byte, 8+1+4+len(payload))
	binary.LittleEndian.PutUint64(reqFrame[0:], reqID)
	reqFrame[8] = cmdProxyDocument
	binary.LittleEndian.PutUint32(reqFrame[9:], uint32(len(payload)))
	copy(reqFrame[13:], payload)

	if err := c.conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		c.conn.Close()
		c.conn = nil
		return 0, nil, err
	}
	if err := writeLengthPrefixed(c.conn, reqFrame); err != nil {
		c.conn.Close()
		c.conn = nil
		return 0, nil, fmt.Errorf("write: %w", err)
	}
	respFrame, err := readLengthPrefixed(c.conn)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return 0, nil, fmt.Errorf("read: %w", err)
	}
	if len(respFrame) < 8+1+4 {
		return 0, nil, fmt.Errorf("short response")
	}
	respID := binary.LittleEndian.Uint64(respFrame[0:])
	if respID != reqID {
		return 0, nil, fmt.Errorf("response request ID mismatch")
	}
	respStatus := respFrame[8]
	payloadLen := binary.LittleEndian.Uint32(respFrame[9:])
	if 13+int(payloadLen) > len(respFrame) {
		return 0, nil, fmt.Errorf("invalid response payload length")
	}
	respPayload := respFrame[13 : 13+payloadLen]

	var docResp proxyDocumentResponse
	if err := json.Unmarshal(respPayload, &docResp); err != nil {
		return 0, nil, fmt.Errorf("decode response: %w", err)
	}
	if respStatus == statusError && docResp.Error != "" {
		return 0, nil, fmt.Errorf("%s", docResp.Error)
	}
	if docResp.Body != "" {
		respBody, _ = base64.StdEncoding.DecodeString(docResp.Body)
	}
	return docResp.Status, respBody, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func readLengthPrefixed(conn net.Conn) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lenBuf)
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large")
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
