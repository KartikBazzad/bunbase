// Package client provides a Go client for the Buncast IPC server (Unix socket).
package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Client communicates with the Buncast service via Unix socket IPC.
type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	requestID  uint64
}

// New creates a new Buncast client.
func New(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		requestID:  1,
	}
}

// Connect establishes a connection to the Buncast server.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("buncast: connect: %w", err)
	}

	c.conn = conn
	return nil
}

// Close closes the connection to the Buncast server.
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

// Protocol constants (must match internal/ipc/protocol.go)
const (
	requestIDSize  = 8
	commandSize    = 1
	statusSize     = 1
	payloadLenSize = 4
	topicLenSize   = 2
	maxFrameSize   = 16 * 1024 * 1024
)

const (
	cmdCreateTopic = 1
	cmdDeleteTopic = 2
	cmdListTopics  = 3
	cmdPublish     = 4
	cmdSubscribe   = 5
)

const (
	statusOK    = 0
	statusError = 1
)

func (c *Client) writeFrame(data []byte) error {
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := c.conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := c.conn.Write(data)
	return err
}

func (c *Client) readFrame() ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lenBuf)
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) sendRequest(command uint8, payload []byte) (status uint8, respPayload []byte, err error) {
	if err = c.Connect(); err != nil {
		return 0, nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	reqID := c.requestID
	c.requestID++

	size := requestIDSize + commandSize + payloadLenSize + len(payload)
	if size > maxFrameSize {
		return 0, nil, fmt.Errorf("request too large")
	}

	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint64(buf[offset:], reqID)
	offset += requestIDSize
	buf[offset] = command
	offset += commandSize
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(payload)))
	offset += payloadLenSize
	if len(payload) > 0 {
		copy(buf[offset:], payload)
	}

	if err = c.writeFrame(buf); err != nil {
		return 0, nil, err
	}

	respData, err := c.readFrame()
	if err != nil {
		return 0, nil, err
	}

	if len(respData) < requestIDSize+statusSize+payloadLenSize {
		return 0, nil, fmt.Errorf("invalid response frame")
	}

	offset = 0
	_ = binary.LittleEndian.Uint64(respData[offset:])
	offset += requestIDSize
	status = respData[offset]
	offset += statusSize
	payloadLen := binary.LittleEndian.Uint32(respData[offset:])
	offset += payloadLenSize
	if offset+int(payloadLen) > len(respData) {
		return 0, nil, fmt.Errorf("invalid response payload length")
	}
	if payloadLen > 0 {
		respPayload = make([]byte, payloadLen)
		copy(respPayload, respData[offset:])
	}
	return status, respPayload, nil
}

func encodeTopicPayload(topic string) ([]byte, error) {
	topicBytes := []byte(topic)
	if len(topicBytes) > 1024 {
		return nil, fmt.Errorf("topic too long")
	}
	buf := make([]byte, topicLenSize+len(topicBytes))
	binary.LittleEndian.PutUint16(buf[0:], uint16(len(topicBytes)))
	copy(buf[topicLenSize:], topicBytes)
	return buf, nil
}

func encodePublishPayload(topic string, body []byte) ([]byte, error) {
	topicBytes := []byte(topic)
	if len(topicBytes) > 1024 {
		return nil, fmt.Errorf("topic too long")
	}
	size := topicLenSize + len(topicBytes) + 4 + len(body)
	if size > maxFrameSize {
		return nil, fmt.Errorf("payload too large")
	}
	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(topicBytes)))
	offset += topicLenSize
	copy(buf[offset:], topicBytes)
	offset += len(topicBytes)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(body)))
	offset += 4
	if len(body) > 0 {
		copy(buf[offset:], body)
	}
	return buf, nil
}

// CreateTopic creates a topic (idempotent).
func (c *Client) CreateTopic(topic string) error {
	payload, err := encodeTopicPayload(topic)
	if err != nil {
		return err
	}
	status, respPayload, err := c.sendRequest(cmdCreateTopic, payload)
	if err != nil {
		return err
	}
	if status != statusOK {
		return decodeError(respPayload)
	}
	return nil
}

// DeleteTopic deletes a topic.
func (c *Client) DeleteTopic(topic string) error {
	payload, err := encodeTopicPayload(topic)
	if err != nil {
		return err
	}
	status, respPayload, err := c.sendRequest(cmdDeleteTopic, payload)
	if err != nil {
		return err
	}
	if status != statusOK {
		return decodeError(respPayload)
	}
	return nil
}

// ListTopics returns all topic names.
func (c *Client) ListTopics() ([]string, error) {
	status, respPayload, err := c.sendRequest(cmdListTopics, nil)
	if err != nil {
		return nil, err
	}
	if status != statusOK {
		return nil, decodeError(respPayload)
	}
	var topics []string
	if err := json.Unmarshal(respPayload, &topics); err != nil {
		return nil, fmt.Errorf("buncast: decode list topics: %w", err)
	}
	return topics, nil
}

// Publish publishes a message to a topic.
func (c *Client) Publish(topic string, payload []byte) error {
	body, err := encodePublishPayload(topic, payload)
	if err != nil {
		return err
	}
	status, respPayload, err := c.sendRequest(cmdPublish, body)
	if err != nil {
		return err
	}
	if status != statusOK {
		return decodeError(respPayload)
	}
	return nil
}

func decodeError(payload []byte) error {
	var v struct {
		Error string `json:"error"`
	}
	if len(payload) > 0 && json.Unmarshal(payload, &v) == nil && v.Error != "" {
		return fmt.Errorf("buncast: %s", v.Error)
	}
	return fmt.Errorf("buncast: error (status)")
}

// Message is a message received when subscribing.
type Message struct {
	Topic   string
	Payload []byte
}

// Subscribe subscribes to a topic and calls fn for each message until the connection is closed or fn returns a non-nil error.
// Subscribe blocks. The server streams length-prefixed message frames after the initial OK response.
func (c *Client) Subscribe(topic string, fn func(msg *Message) error) error {
	if err := c.Connect(); err != nil {
		return err
	}

	payload, err := encodeTopicPayload(topic)
	if err != nil {
		return err
	}

	c.mu.Lock()
	status, respPayload, err := c.sendRequest(cmdSubscribe, payload)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	if status != statusOK {
		c.mu.Unlock()
		return decodeError(respPayload)
	}
	// After Subscribe response, server streams message frames; we don't release the lock for sendRequest
	// but sendRequest already released. So we're still holding the lock. We need to read frames in a loop.
	// Actually sendRequest does Lock/Unlock, so after sendRequest we're unlocked. Now we need to read
	// length-prefixed message frames from the connection. We must not call sendRequest again on this
	// connection while we're in Subscribe. So we need a dedicated connection for Subscribe or we need
	// to document that Subscribe holds the connection and no other method should be called concurrently.
	// Simplest: document that Subscribe blocks and uses the connection exclusively. So after sendRequest
	// we loop reading frames (length + body), decode topic + payload, call fn. If fn returns error we close and return.
	c.mu.Unlock()

	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		length := binary.LittleEndian.Uint32(lenBuf)
		if length > maxFrameSize {
			return fmt.Errorf("message frame too large")
		}
		frame := make([]byte, length)
		if _, err := io.ReadFull(c.conn, frame); err != nil {
			return err
		}
		// Decode: TopicLen(2) + Topic + PayloadLen(4) + Payload
		if len(frame) < topicLenSize {
			continue
		}
		topicLen := binary.LittleEndian.Uint16(frame[0:])
		offset := topicLenSize
		if offset+int(topicLen) > len(frame) {
			continue
		}
		msgTopic := string(frame[offset : offset+int(topicLen)])
		offset += int(topicLen)
		if offset+4 > len(frame) {
			continue
		}
		payloadLen := binary.LittleEndian.Uint32(frame[offset:])
		offset += 4
		if offset+int(payloadLen) > len(frame) {
			continue
		}
		var msgPayload []byte
		if payloadLen > 0 {
			msgPayload = make([]byte, payloadLen)
			copy(msgPayload, frame[offset:])
		}
		if err := fn(&Message{Topic: msgTopic, Payload: msgPayload}); err != nil {
			return err
		}
	}
}
