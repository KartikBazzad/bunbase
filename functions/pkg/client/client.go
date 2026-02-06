package client

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/kartikbazzad/bunbase/functions/internal/ipc"
)

var (
	ErrConnectionFailed = errors.New("failed to connect to server")
	ErrInvalidResponse  = errors.New("invalid response from server")
)

// Client is a client for the functions service IPC (Unix socket or TCP).
type Client struct {
	network string // "unix" or "tcp"
	address string // socket path or "host:port"
	conn    net.Conn
	mu      sync.Mutex
	requestID uint64
}

// New creates a new IPC client. addr is either a Unix socket path or "tcp://host:port" for TCP.
func New(addr string) *Client {
	network := "unix"
	address := addr
	if len(addr) >= 7 && (addr[:7] == "tcp://") {
		network = "tcp"
		address = addr[7:]
	}
	return &Client{
		network:   network,
		address:   address,
		requestID: 1,
	}
}

// Connect connects to the server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, err := net.Dial(c.network, c.address)
	if err != nil {
		return ErrConnectionFailed
	}

	c.conn = conn
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}

// InvokeRequest represents an invocation request
type InvokeRequest struct {
	FunctionID string
	Method     string
	Path       string
	Headers    map[string]string
	Query      map[string]string
	Body       []byte
}

// InvokeResponse represents an invocation response
type InvokeResponse struct {
	Success       bool
	Status        int
	Headers       map[string]string
	Body          []byte
	Error         string
	ExecutionTime int64 // milliseconds
	ExecutionID   string
}

// Invoke invokes a function
func (c *Client) Invoke(req *InvokeRequest) (*InvokeResponse, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	requestID := c.nextRequestID()

	// Encode body as base64
	bodyBase64 := ""
	if len(req.Body) > 0 {
		bodyBase64 = base64.StdEncoding.EncodeToString(req.Body)
	}

	// Build payload
	payload := map[string]interface{}{
		"function_id": req.FunctionID,
		"method":       req.Method,
		"path":         req.Path,
		"headers":      req.Headers,
		"query":        req.Query,
		"body":         bodyBase64,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Build request frame
	frame := &ipc.RequestFrame{
		RequestID: requestID,
		Command:   ipc.CmdInvoke,
		Payload:   payloadJSON,
	}

	// Send request
	requestData, err := ipc.EncodeRequest(frame)
	if err != nil {
		return nil, err
	}

	if err := c.writeFrame(requestData); err != nil {
		return nil, err
	}

	// Read response
	responseData, err := c.readFrame()
	if err != nil {
		return nil, err
	}

	responseFrame, err := ipc.DecodeResponse(responseData)
	if err != nil {
		return nil, err
	}

	if responseFrame.RequestID != requestID {
		return nil, ErrInvalidResponse
	}

	if responseFrame.Status != ipc.StatusOK {
		var errorResp map[string]string
		if err := json.Unmarshal(responseFrame.Payload, &errorResp); err == nil {
			return nil, fmt.Errorf("%s", errorResp["error"])
		}
		return nil, fmt.Errorf("server error")
	}

	// Parse response payload
	var respPayload map[string]interface{}
	if err := json.Unmarshal(responseFrame.Payload, &respPayload); err != nil {
		return nil, err
	}

	result := &InvokeResponse{
		Success: respPayload["success"].(bool),
	}

	if !result.Success {
		if errStr, ok := respPayload["error"].(string); ok {
			result.Error = errStr
		}
		return result, nil
	}

	// Parse success response
	if status, ok := respPayload["status"].(float64); ok {
		result.Status = int(status)
	}

	if headers, ok := respPayload["headers"].(map[string]interface{}); ok {
		result.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				result.Headers[k] = str
			}
		}
	}

	if bodyStr, ok := respPayload["body"].(string); ok && bodyStr != "" {
		body, err := base64.StdEncoding.DecodeString(bodyStr)
		if err == nil {
			result.Body = body
		}
	}

	if execTime, ok := respPayload["execution_time_ms"].(float64); ok {
		result.ExecutionTime = int64(execTime)
	}

	if execID, ok := respPayload["execution_id"].(string); ok {
		result.ExecutionID = execID
	}

	return result, nil
}

// readFrame reads a length-prefixed frame
func (c *Client) readFrame() ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > ipc.MaxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

// writeFrame writes a length-prefixed frame
func (c *Client) writeFrame(data []byte) error {
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := c.conn.Write(lenBuf); err != nil {
		return err
	}

	if _, err := c.conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *Client) nextRequestID() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.requestID
	c.requestID++
	return id
}
