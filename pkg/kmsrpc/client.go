package kmsrpc

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

// Protocol matches bun-kms internal/rpc: length-prefixed frames,
// request (requestID, command, payload), response (requestID, status, payload).
const (
	cmdGetSecret = 1
	cmdPutSecret = 2
	statusOK     = 0
	statusError  = 1
	frameOverhead = 8 + 1 + 4
	maxFrameSize  = 16 * 1024 * 1024
)

type getSecretRequest struct {
	Name string `json:"name"`
}

type getSecretResponse struct {
	ValueB64 string `json:"value_b64"`
	Error    string `json:"error,omitempty"`
}

type putSecretRequest struct {
	Name     string `json:"name"`
	ValueB64 string `json:"value_b64"`
}

type putSecretResponse struct {
	Error string `json:"error,omitempty"`
}

// Client is a TCP RPC client for KMS GetSecret/PutSecret.
type Client struct {
	Addr      string
	conn      net.Conn
	mu        sync.Mutex
	requestID uint64
	Timeout   time.Duration
}

// New creates a new KMS RPC client. addr is TCP address (e.g. "bunkms:9092").
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

// GetSecret returns the secret value by name.
func (c *Client) GetSecret(name string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	reqID := c.requestID
	c.requestID++

	payload, err := json.Marshal(getSecretRequest{Name: name})
	if err != nil {
		return nil, err
	}
	reqFrame := make([]byte, frameOverhead+len(payload))
	binary.LittleEndian.PutUint64(reqFrame[0:], reqID)
	reqFrame[8] = cmdGetSecret
	binary.LittleEndian.PutUint32(reqFrame[9:], uint32(len(payload)))
	copy(reqFrame[13:], payload)

	if err := c.conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, err
	}
	if err := writeLengthPrefixed(c.conn, reqFrame); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("write: %w", err)
	}
	respFrame, err := readLengthPrefixed(c.conn)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("read: %w", err)
	}
	if len(respFrame) < frameOverhead {
		return nil, fmt.Errorf("short response")
	}
	if binary.LittleEndian.Uint64(respFrame[0:]) != reqID {
		return nil, fmt.Errorf("response request ID mismatch")
	}
	respStatus := respFrame[8]
	payloadLen := binary.LittleEndian.Uint32(respFrame[9:])
	if 13+int(payloadLen) > len(respFrame) {
		return nil, fmt.Errorf("invalid response payload length")
	}
	respPayload := respFrame[13 : 13+payloadLen]

	var docResp getSecretResponse
	if err := json.Unmarshal(respPayload, &docResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if respStatus == statusError && docResp.Error != "" {
		return nil, fmt.Errorf("%s", docResp.Error)
	}
	if docResp.ValueB64 == "" {
		return nil, fmt.Errorf("empty value_b64")
	}
	return base64.StdEncoding.DecodeString(docResp.ValueB64)
}

// PutSecret stores a secret by name. value is raw bytes.
func (c *Client) PutSecret(name string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	reqID := c.requestID
	c.requestID++

	payload, err := json.Marshal(putSecretRequest{
		Name:     name,
		ValueB64: base64.StdEncoding.EncodeToString(value),
	})
	if err != nil {
		return err
	}
	reqFrame := make([]byte, frameOverhead+len(payload))
	binary.LittleEndian.PutUint64(reqFrame[0:], reqID)
	reqFrame[8] = cmdPutSecret
	binary.LittleEndian.PutUint32(reqFrame[9:], uint32(len(payload)))
	copy(reqFrame[13:], payload)

	if err := c.conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		c.conn.Close()
		c.conn = nil
		return err
	}
	if err := writeLengthPrefixed(c.conn, reqFrame); err != nil {
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("write: %w", err)
	}
	respFrame, err := readLengthPrefixed(c.conn)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("read: %w", err)
	}
	if len(respFrame) < frameOverhead {
		return fmt.Errorf("short response")
	}
	if binary.LittleEndian.Uint64(respFrame[0:]) != reqID {
		return fmt.Errorf("response request ID mismatch")
	}
	respStatus := respFrame[8]
	payloadLen := binary.LittleEndian.Uint32(respFrame[9:])
	if 13+int(payloadLen) > len(respFrame) {
		return fmt.Errorf("invalid response payload length")
	}
	respPayload := respFrame[13 : 13+payloadLen]

	var docResp putSecretResponse
	if err := json.Unmarshal(respPayload, &docResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if respStatus == statusError && docResp.Error != "" {
		return fmt.Errorf("%s", docResp.Error)
	}
	return nil
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
