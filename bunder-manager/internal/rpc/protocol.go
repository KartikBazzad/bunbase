package rpc

import (
	"encoding/binary"
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
	MaxFrameSize   = 16 * 1024 * 1024
)

// Command codes
const (
	CmdProxyKV = 1
)

// Status codes
const (
	StatusOK    = 0
	StatusError = 1
)

// RequestFrame is a single RPC request.
type RequestFrame struct {
	RequestID uint64
	Command   uint8
	Payload   []byte
}

// ResponseFrame is a single RPC response.
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

// DecodeResponse decodes a response from bytes.
func DecodeResponse(data []byte) (*ResponseFrame, error) {
	if len(data) < RequestIDSize+StatusSize+PayloadLenSize {
		return nil, ErrInvalidFrame
	}
	offset := 0
	resp := &ResponseFrame{}
	resp.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize
	resp.Status = data[offset]
	offset += StatusSize
	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += PayloadLenSize
	if offset+int(payloadLen) > len(data) {
		return nil, ErrInvalidFrame
	}
	if payloadLen > 0 {
		resp.Payload = make([]byte, payloadLen)
		copy(resp.Payload, data[offset:])
	}
	return resp, nil
}
