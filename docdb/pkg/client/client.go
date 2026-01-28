package client

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrConnectionFailed = errors.New("failed to connect to server")
	ErrInvalidResponse  = errors.New("invalid response from server")
)

type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	requestID  uint64
}

func New(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		requestID:  1,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return ErrConnectionFailed
	}

	c.conn = conn
	return nil
}

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

func (c *Client) OpenDB(name string) (uint64, error) {
	if err := c.Connect(); err != nil {
		return 0, err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdOpenDB,
		DBID:      0,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:  types.OpCreate,
				DocID:   0,
				Payload: []byte(name),
			},
		},
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return 0, err
	}

	if resp.Status != types.StatusOK {
		return 0, errors.New(string(resp.Data))
	}

	if len(resp.Data) != 8 {
		return 0, ErrInvalidResponse
	}

	dbID := binary.LittleEndian.Uint64(resp.Data)
	return dbID, nil
}

func (c *Client) CloseDB(dbID uint64) error {
	if err := c.Connect(); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdCloseDB,
		DBID:      dbID,
		OpCount:   0,
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return err
	}

	if resp.Status != types.StatusOK {
		return errors.New(string(resp.Data))
	}

	return nil
}

func (c *Client) Create(dbID uint64, docID uint64, payload []byte) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if err := validateJSON(payload); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:  types.OpCreate,
				DocID:   docID,
				Payload: payload,
			},
		},
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return err
	}

	if resp.Status != types.StatusOK {
		return errors.New(string(resp.Data))
	}

	return nil
}

func (c *Client) Read(dbID uint64, docID uint64) ([]byte, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:  types.OpRead,
				DocID:   docID,
				Payload: nil,
			},
		},
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return nil, err
	}

	if resp.Status == types.StatusNotFound {
		return nil, errors.New("document not found")
	}

	if resp.Status != types.StatusOK {
		return nil, errors.New(string(resp.Data))
	}

	return c.parseReadResponse(resp.Data)
}

func (c *Client) Update(dbID uint64, docID uint64, payload []byte) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if err := validateJSON(payload); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:  types.OpUpdate,
				DocID:   docID,
				Payload: payload,
			},
		},
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return err
	}

	if resp.Status != types.StatusOK {
		return errors.New(string(resp.Data))
	}

	return nil
}

func (c *Client) Delete(dbID uint64, docID uint64) error {
	if err := c.Connect(); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:  types.OpDelete,
				DocID:   docID,
				Payload: nil,
			},
		},
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return err
	}

	if resp.Status != types.StatusOK {
		return errors.New(string(resp.Data))
	}

	return nil
}

func (c *Client) BatchExecute(dbID uint64, ops []ipc.Operation) ([][]byte, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   uint32(len(ops)),
		Ops:       ops,
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return nil, err
	}

	if resp.Status != types.StatusOK {
		return nil, errors.New(string(resp.Data))
	}

	return c.parseBatchResponse(resp.Data)
}

func (c *Client) Stats() (*types.Stats, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdStats,
		DBID:      0,
		OpCount:   0,
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return nil, err
	}

	if resp.Status != types.StatusOK {
		return nil, errors.New(string(resp.Data))
	}

	return c.parseStats(resp.Data)
}

func (c *Client) nextRequestID() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.requestID
	c.requestID++
	return id
}

func (c *Client) sendRequest(frame *ipc.RequestFrame) (*ipc.ResponseFrame, error) {
	data, err := ipc.EncodeRequest(frame)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.writeFrame(data); err != nil {
		return nil, err
	}

	respData, err := c.readFrame()
	if err != nil {
		return nil, err
	}

	resp, err := ipc.DecodeResponse(respData)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

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

func (c *Client) readFrame() ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := c.conn.Read(lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > 16*1024*1024 {
		return nil, errors.New("frame too large")
	}

	buf := make([]byte, length)
	if _, err := c.conn.Read(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *Client) parseReadResponse(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, ErrInvalidResponse
	}

	count := binary.LittleEndian.Uint32(data[0:])
	if count != 1 {
		return nil, ErrInvalidResponse
	}

	offset := 4
	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if uint32(len(data[offset:])) != payloadLen {
		return nil, ErrInvalidResponse
	}

	return data[offset:], nil
}

func (c *Client) parseBatchResponse(data []byte) ([][]byte, error) {
	if len(data) < 4 {
		return nil, ErrInvalidResponse
	}

	count := binary.LittleEndian.Uint32(data[0:])
	offset := 4

	responses := make([][]byte, count)

	for i := 0; i < int(count); i++ {
		if offset+4 > len(data) {
			return nil, ErrInvalidResponse
		}

		payloadLen := binary.LittleEndian.Uint32(data[offset:])
		offset += 4

		if offset+int(payloadLen) > len(data) {
			return nil, ErrInvalidResponse
		}

		responses[i] = data[offset : offset+int(payloadLen)]
		offset += int(payloadLen)
	}

	return responses, nil
}

func (c *Client) parseStats(data []byte) (*types.Stats, error) {
	if len(data) != 40 {
		return nil, ErrInvalidResponse
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

func EncodeBytes(data []byte) map[string]any {
	return map[string]any{
		"_type":    "bytes",
		"encoding": "base64",
		"data":     base64.StdEncoding.EncodeToString(data),
	}
}

func DecodeBytes(obj map[string]any) ([]byte, error) {
	if obj["_type"] != "bytes" {
		return nil, fmt.Errorf("not a bytes wrapper")
	}
	if obj["encoding"] != "base64" {
		return nil, fmt.Errorf("unsupported encoding: %v", obj["encoding"])
	}
	dataStr, ok := obj["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data field is not a string")
	}
	return base64.StdEncoding.DecodeString(dataStr)
}

func validateJSON(payload []byte) error {
	if len(payload) == 0 {
		return types.ErrInvalidJSON
	}
	if !utf8.Valid(payload) {
		return types.ErrInvalidJSON
	}
	var v interface{}
	return json.Unmarshal(payload, &v)
}
