package ipc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
)

var (
	ErrInvalidFrame  = errors.New("invalid frame format")
	ErrFrameTooLarge = errors.New("frame too large")
)

const (
	RequestIDSize  = 8
	CommandSize    = 1
	StatusSize     = 1
	PayloadLenSize = 4
	TopicLenSize   = 2
	MaxFrameSize   = 16 * 1024 * 1024
	MaxTopicLen    = 1024
)

// Command codes
const (
	CmdCreateTopic = 1
	CmdDeleteTopic = 2
	CmdListTopics  = 3
	CmdPublish     = 4
	CmdSubscribe   = 5
)

// Status codes
const (
	StatusOK    = 0
	StatusError = 1
)

// RequestFrame is a single IPC request.
type RequestFrame struct {
	RequestID uint64
	Command   uint8
	Payload   []byte
}

// ResponseFrame is a single IPC response.
type ResponseFrame struct {
	RequestID uint64
	Status    uint8
	Payload   []byte
}

// EncodeRequest encodes a request for sending.
func EncodeRequest(req *RequestFrame) ([]byte, error) {
	size := RequestIDSize + CommandSize + PayloadLenSize + len(req.Payload)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint64(buf[offset:], req.RequestID)
	offset += RequestIDSize
	buf[offset] = req.Command
	offset += CommandSize
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(req.Payload)))
	offset += PayloadLenSize
	if len(req.Payload) > 0 {
		copy(buf[offset:], req.Payload)
	}
	return buf, nil
}

// DecodeRequest decodes a request from bytes.
func DecodeRequest(data []byte) (*RequestFrame, error) {
	if len(data) < RequestIDSize+CommandSize+PayloadLenSize {
		return nil, ErrInvalidFrame
	}
	offset := 0
	req := &RequestFrame{}
	req.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize
	req.Command = data[offset]
	offset += CommandSize
	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += PayloadLenSize
	if offset+int(payloadLen) > len(data) {
		return nil, ErrInvalidFrame
	}
	if payloadLen > 0 {
		req.Payload = make([]byte, payloadLen)
		copy(req.Payload, data[offset:])
	}
	return req, nil
}

// EncodeResponse encodes a response for sending.
func EncodeResponse(resp *ResponseFrame) ([]byte, error) {
	size := RequestIDSize + StatusSize + PayloadLenSize + len(resp.Payload)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint64(buf[offset:], resp.RequestID)
	offset += RequestIDSize
	buf[offset] = resp.Status
	offset += StatusSize
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(resp.Payload)))
	offset += PayloadLenSize
	if len(resp.Payload) > 0 {
		copy(buf[offset:], resp.Payload)
	}
	return buf, nil
}

// Payload helpers: topic (2-byte len + bytes), publish (topic + 4-byte len + payload)

// EncodeTopicPayload encodes a single topic for CreateTopic/DeleteTopic/Subscribe.
func EncodeTopicPayload(topic string) ([]byte, error) {
	topicBytes := []byte(topic)
	if len(topicBytes) > MaxTopicLen {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, TopicLenSize+len(topicBytes))
	binary.LittleEndian.PutUint16(buf[0:], uint16(len(topicBytes)))
	copy(buf[TopicLenSize:], topicBytes)
	return buf, nil
}

// DecodeTopicPayload decodes a topic from payload.
func DecodeTopicPayload(payload []byte) (string, error) {
	if len(payload) < TopicLenSize {
		return "", ErrInvalidFrame
	}
	n := binary.LittleEndian.Uint16(payload[0:])
	if int(TopicLenSize+n) > len(payload) {
		return "", ErrInvalidFrame
	}
	return string(payload[TopicLenSize : TopicLenSize+n]), nil
}

// EncodePublishPayload encodes topic + payload for Publish.
func EncodePublishPayload(topic string, body []byte) ([]byte, error) {
	topicBytes := []byte(topic)
	if len(topicBytes) > MaxTopicLen {
		return nil, ErrFrameTooLarge
	}
	size := TopicLenSize + len(topicBytes) + 4 + len(body)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(topicBytes)))
	offset += TopicLenSize
	copy(buf[offset:], topicBytes)
	offset += len(topicBytes)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(body)))
	offset += 4
	if len(body) > 0 {
		copy(buf[offset:], body)
	}
	return buf, nil
}

// DecodePublishPayload decodes topic and body from Publish payload.
func DecodePublishPayload(payload []byte) (topic string, body []byte, err error) {
	if len(payload) < TopicLenSize {
		return "", nil, ErrInvalidFrame
	}
	topicLen := binary.LittleEndian.Uint16(payload[0:])
	offset := TopicLenSize
	if offset+int(topicLen) > len(payload) {
		return "", nil, ErrInvalidFrame
	}
	topic = string(payload[offset : offset+int(topicLen)])
	offset += int(topicLen)
	if offset+4 > len(payload) {
		return topic, nil, nil
	}
	bodyLen := binary.LittleEndian.Uint32(payload[offset:])
	offset += 4
	if offset+int(bodyLen) > len(payload) {
		return topic, nil, ErrInvalidFrame
	}
	if bodyLen > 0 {
		body = make([]byte, bodyLen)
		copy(body, payload[offset:])
	}
	return topic, body, nil
}

// EncodeMessageFrame encodes a message for streaming to a subscriber (length-prefixed frame body).
func EncodeMessageFrame(topic string, body []byte) ([]byte, error) {
	topicBytes := []byte(topic)
	if len(topicBytes) > MaxTopicLen {
		return nil, ErrFrameTooLarge
	}
	size := TopicLenSize + len(topicBytes) + PayloadLenSize + len(body)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, size)
	offset := 0
	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(topicBytes)))
	offset += TopicLenSize
	copy(buf[offset:], topicBytes)
	offset += len(topicBytes)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(body)))
	offset += PayloadLenSize
	if len(body) > 0 {
		copy(buf[offset:], body)
	}
	return buf, nil
}

// EncodeListTopicsResponse encodes ListTopics response as JSON array.
func EncodeListTopicsResponse(topics []string) ([]byte, error) {
	return json.Marshal(topics)
}

// DecodeListTopicsResponse decodes ListTopics response.
func DecodeListTopicsResponse(payload []byte) ([]string, error) {
	var topics []string
	if err := json.Unmarshal(payload, &topics); err != nil {
		return nil, err
	}
	return topics, nil
}
