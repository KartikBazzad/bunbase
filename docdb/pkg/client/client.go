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
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrConnectionFailed = errors.New("failed to connect to server")
	ErrInvalidResponse  = errors.New("invalid response from server")
	ErrConnectionClosed = errors.New("connection closed")
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

func (c *Client) Create(dbID uint64, collection string, docID uint64, payload []byte) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if collection == "" {
		collection = "_default"
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
				OpType:     types.OpCreate,
				Collection: collection,
				DocID:      docID,
				Payload:    payload,
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

func (c *Client) Read(dbID uint64, collection string, docID uint64) ([]byte, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	if collection == "" {
		collection = "_default"
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:     types.OpRead,
				Collection: collection,
				DocID:      docID,
				Payload:    nil,
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

func (c *Client) Update(dbID uint64, collection string, docID uint64, payload []byte) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if collection == "" {
		collection = "_default"
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
				OpType:     types.OpUpdate,
				Collection: collection,
				DocID:      docID,
				Payload:    payload,
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

// ExecuteBatch sends a single request frame with multiple operations and returns the
// serialized response data (count + per-response length + payload). Use ParseBatchResponse
// to parse the result. Reduces round-trips when batch size > 1.
func (c *Client) ExecuteBatch(dbID uint64, ops []ipc.Operation) ([]byte, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, errors.New("ExecuteBatch requires at least one op")
	}
	copyOps := make([]ipc.Operation, len(ops))
	for i := range ops {
		copyOps[i] = ops[i]
		if copyOps[i].Collection == "" {
			copyOps[i].Collection = "_default"
		}
		if copyOps[i].OpType == types.OpCreate || copyOps[i].OpType == types.OpUpdate {
			if err := validateJSON(copyOps[i].Payload); err != nil {
				return nil, err
			}
		}
	}
	frame := &ipc.RequestFrame{
		RequestID: c.nextRequestID(),
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   uint32(len(copyOps)),
		Ops:       copyOps,
	}
	resp, err := c.sendRequest(frame)
	if err != nil {
		return nil, err
	}
	if resp.Status != types.StatusOK {
		return nil, errors.New(string(resp.Data))
	}
	return resp.Data, nil
}

// ParseBatchResponse parses the serialized response from ExecuteBatch (format: 4-byte
// count, then for each response 4-byte length + payload). Returns one slice per op.
func ParseBatchResponse(data []byte) ([][]byte, error) {
	if len(data) < 4 {
		return nil, ErrInvalidResponse
	}
	n := binary.LittleEndian.Uint32(data[0:4])
	if n == 0 {
		return [][]byte{}, nil
	}
	out := make([][]byte, 0, n)
	offset := 4
	for i := uint32(0); i < n; i++ {
		if offset+4 > len(data) {
			return nil, ErrInvalidResponse
		}
		sz := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		if offset+int(sz) > len(data) {
			return nil, ErrInvalidResponse
		}
		payload := make([]byte, sz)
		copy(payload, data[offset:offset+int(sz)])
		out = append(out, payload)
		offset += int(sz)
	}
	return out, nil
}

func (c *Client) Delete(dbID uint64, collection string, docID uint64) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if collection == "" {
		collection = "_default"
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:     types.OpDelete,
				Collection: collection,
				DocID:      docID,
				Payload:    nil,
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

func (c *Client) Patch(dbID uint64, collection string, docID uint64, ops []types.PatchOperation) error {
	if err := c.Connect(); err != nil {
		return err
	}

	if collection == "" {
		collection = "_default"
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdExecute,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:     types.OpPatch,
				Collection: collection,
				DocID:      docID,
				PatchOps:   ops,
				Payload:    nil,
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

func (c *Client) CreateCollection(dbID uint64, name string) error {
	if err := c.Connect(); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdCreateCollection,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:     types.OpCreateCollection,
				Collection: name,
				DocID:      0,
				Payload:    nil,
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

func (c *Client) DeleteCollection(dbID uint64, name string) error {
	if err := c.Connect(); err != nil {
		return err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdDeleteCollection,
		DBID:      dbID,
		OpCount:   1,
		Ops: []ipc.Operation{
			{
				OpType:     types.OpDeleteCollection,
				Collection: name,
				DocID:      0,
				Payload:    nil,
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

func (c *Client) ListCollections(dbID uint64) ([]string, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	reqID := c.nextRequestID()

	frame := &ipc.RequestFrame{
		RequestID: reqID,
		Command:   ipc.CmdListCollections,
		DBID:      dbID,
		OpCount:   0,
	}

	resp, err := c.sendRequest(frame)
	if err != nil {
		return nil, err
	}

	if resp.Status != types.StatusOK {
		return nil, errors.New(string(resp.Data))
	}

	var collections []string
	if err := json.Unmarshal(resp.Data, &collections); err != nil {
		return nil, fmt.Errorf("failed to parse collections: %w", err)
	}

	return collections, nil
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
	if c.conn == nil {
		return ErrConnectionClosed
	}
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
	if c.conn == nil {
		return nil, ErrConnectionClosed
	}
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > 16*1024*1024 {
		return nil, errors.New("frame too large")
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
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
