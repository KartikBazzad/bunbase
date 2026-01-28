package functions

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Client communicates with the functions service via Unix socket IPC
type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	requestID  uint64
}

// NewClient creates a new functions service client
func NewClient(socketPath string) (*Client, error) {
	return &Client{
		socketPath: socketPath,
		requestID:  1,
	}, nil
}

// Connect connects to the functions service
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to functions service: %w", err)
	}

	c.conn = conn
	return nil
}

// Close closes the connection
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

// Protocol constants
const (
	RequestIDSize  = 8
	CommandSize    = 1
	StatusSize     = 1
	PayloadLenSize = 4
	MaxFrameSize   = 16 * 1024 * 1024
)

// Command types
const (
	CmdInvoke            = 0x01
	CmdGetLogs           = 0x02
	CmdGetMetrics        = 0x03
	CmdRegisterFunction  = 0x04
	CmdDeployFunction    = 0x05
)

// Status codes
const (
	StatusOK       = 0x00
	StatusError    = 0x01
	StatusNotFound = 0x02
)

// nextRequestID generates the next request ID
func (c *Client) nextRequestID() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.requestID
	c.requestID++
	return id
}

// writeFrame writes a length-prefixed frame to the connection
func (c *Client) writeFrame(data []byte) error {
	// Write length prefix (4 bytes, little endian)
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := c.conn.Write(lenBuf); err != nil {
		return fmt.Errorf("failed to write frame length: %w", err)
	}

	// Write frame data
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write frame data: %w", err)
	}

	return nil
}

// readFrame reads a length-prefixed frame from the connection
func (c *Client) readFrame() ([]byte, error) {
	// Read length prefix (4 bytes, little endian)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read frame length: %w", err)
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > MaxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}

	// Read frame data
	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, fmt.Errorf("failed to read frame data: %w", err)
	}

	return data, nil
}

// sendRequest sends a request and receives a response
func (c *Client) sendRequest(command uint8, payload []byte) (*ResponseFrame, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	requestID := c.nextRequestID()

	// Build request frame: RequestID (8) + Command (1) + PayloadLen (4) + Payload
	frameSize := RequestIDSize + CommandSize + PayloadLenSize + len(payload)
	if frameSize > MaxFrameSize {
		return nil, fmt.Errorf("request too large")
	}

	frame := make([]byte, frameSize)
	offset := 0

	// RequestID (8 bytes, little endian)
	binary.LittleEndian.PutUint64(frame[offset:], requestID)
	offset += RequestIDSize

	// Command (1 byte)
	frame[offset] = command
	offset += CommandSize

	// Payload length (4 bytes, little endian)
	binary.LittleEndian.PutUint32(frame[offset:], uint32(len(payload)))
	offset += PayloadLenSize

	// Payload
	if len(payload) > 0 {
		copy(frame[offset:], payload)
	}

	// Send frame
	if err := c.writeFrame(frame); err != nil {
		return nil, err
	}

	// Read response
	respData, err := c.readFrame()
	if err != nil {
		return nil, err
	}

	// Parse response frame: RequestID (8) + Status (1) + PayloadLen (4) + Payload
	if len(respData) < RequestIDSize+StatusSize+PayloadLenSize {
		return nil, fmt.Errorf("invalid response frame")
	}

	offset = 0
	respRequestID := binary.LittleEndian.Uint64(respData[offset:])
	offset += RequestIDSize

	status := respData[offset]
	offset += StatusSize

	payloadLen := binary.LittleEndian.Uint32(respData[offset:])
	offset += PayloadLenSize

	if offset+int(payloadLen) > len(respData) {
		return nil, fmt.Errorf("invalid response payload length")
	}

	var respPayload []byte
	if payloadLen > 0 {
		respPayload = respData[offset : offset+int(payloadLen)]
	}

	return &ResponseFrame{
		RequestID: fmt.Sprintf("%d", respRequestID),
		Status:    int(status),
		Payload:   respPayload,
	}, nil
}

// RequestFrame represents a request frame (for reference)
type RequestFrame struct {
	RequestID string
	Command   int
	Payload   []byte
}

// ResponseFrame represents a response frame
type ResponseFrame struct {
	RequestID string
	Status    int
	Payload   []byte
}

// RegisterFunctionRequest represents a function registration request
type RegisterFunctionRequest struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	Handler string `json:"handler"`
}

// RegisterFunctionResponse represents a function registration response
type RegisterFunctionResponse struct {
	FunctionID string `json:"function_id"`
	Name       string `json:"name"`
	Runtime    string `json:"runtime"`
	Handler    string `json:"handler"`
	Status     string `json:"status"`
}

// DeployFunctionRequest represents a function deployment request
type DeployFunctionRequest struct {
	FunctionID string `json:"function_id"`
	Version    string `json:"version"`
	BundlePath string `json:"bundle_path"`
}

// DeployFunctionResponse represents a function deployment response
type DeployFunctionResponse struct {
	DeploymentID string `json:"deployment_id"`
	FunctionID   string `json:"function_id"`
	Version       string `json:"version"`
	Status        string `json:"status"`
}

// RegisterFunction registers a function in the functions service
func (c *Client) RegisterFunction(name, runtime, handler string) (*RegisterFunctionResponse, error) {
	req := RegisterFunctionRequest{
		Name:    name,
		Runtime: runtime,
		Handler: handler,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.sendRequest(CmdRegisterFunction, payload)
	if err != nil {
		return nil, err
	}

	if resp.Status != StatusOK {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(resp.Payload, &errorResp); err == nil {
			return nil, fmt.Errorf("functions service error: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("functions service error: %s", string(resp.Payload))
	}

	var result RegisterFunctionResponse
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeployFunction deploys a function version
func (c *Client) DeployFunction(functionID, version, bundlePath string) (*DeployFunctionResponse, error) {
	req := DeployFunctionRequest{
		FunctionID: functionID,
		Version:    version,
		BundlePath: bundlePath,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.sendRequest(CmdDeployFunction, payload)
	if err != nil {
		return nil, err
	}

	if resp.Status != StatusOK {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(resp.Payload, &errorResp); err == nil {
			return nil, fmt.Errorf("functions service error: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("functions service error: %s", string(resp.Payload))
	}

	var result DeployFunctionResponse
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}
