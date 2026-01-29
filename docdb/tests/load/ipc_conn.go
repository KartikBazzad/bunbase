package load

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

// ipcConnection handles IPC communication for load testing.
type ipcConnection struct {
	conn      net.Conn
	mu        sync.Mutex
	requestID uint64
}

// newIPCConnection creates a new IPC connection.
func newIPCConnection(socketPath string) (*ipcConnection, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}

	return &ipcConnection{
		conn:      conn,
		requestID: 1,
	}, nil
}

// nextRequestID returns the next request ID.
func (c *ipcConnection) nextRequestID() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.requestID
	c.requestID++
	return id
}

// sendRequest sends a request and returns the response.
func (c *ipcConnection) sendRequest(frame *ipc.RequestFrame) (*ipc.ResponseFrame, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := ipc.EncodeRequest(frame)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	if err := c.writeFrame(data); err != nil {
		return nil, fmt.Errorf("failed to write frame: %w", err)
	}

	respData, err := c.readFrame()
	if err != nil {
		return nil, fmt.Errorf("failed to read frame: %w", err)
	}

	resp, err := ipc.DecodeResponse(respData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return resp, nil
}

// writeFrame writes a frame to the connection.
func (c *ipcConnection) writeFrame(data []byte) error {
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

// readFrame reads a frame from the connection.
func (c *ipcConnection) readFrame() ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := c.conn.Read(lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > 16*1024*1024 {
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}

	buf := make([]byte, length)
	if _, err := c.conn.Read(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// close closes the connection.
func (c *ipcConnection) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// parseStats parses stats from response data.
func parseStats(data []byte) (*types.Stats, error) {
	if len(data) != 40 {
		return nil, fmt.Errorf("invalid stats response length: %d", len(data))
	}

	stats := &types.Stats{
		TotalDBs:       int(binary.LittleEndian.Uint64(data[0:])),
		ActiveDBs:      int(binary.LittleEndian.Uint64(data[8:])),
		TotalTxns:      binary.LittleEndian.Uint64(data[16:]),
		WALSize:        binary.LittleEndian.Uint64(data[24:]),
		MemoryUsed:     binary.LittleEndian.Uint64(data[32:]),
		MemoryCapacity: 0,
	}

	return stats, nil
}
