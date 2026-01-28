package ipc

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var (
	ErrInvalidFrame  = errors.New("invalid frame format")
	ErrFrameTooLarge = errors.New("frame too large")
)

const (
	RequestIDSize = 8
	CommandSize   = 1
	StatusSize    = 1
	PayloadLenSize = 4
	MaxFrameSize   = 16 * 1024 * 1024
)

// Commands
const (
	CmdInvoke            = 0x01
	CmdGetLogs           = 0x02
	CmdGetMetrics        = 0x03
	CmdRegisterFunction  = 0x04
	CmdDeployFunction    = 0x05
)

// Status values
const (
	StatusOK    = 0x00
	StatusError = 0x01
)

// RequestFrame represents an IPC request frame
type RequestFrame struct {
	RequestID uint64
	Command   uint8
	Payload   []byte // JSON
}

// ResponseFrame represents an IPC response frame
type ResponseFrame struct {
	RequestID uint64
	Status    uint8
	Payload   []byte // JSON
}

// EncodeRequest encodes a request frame
func EncodeRequest(frame *RequestFrame) ([]byte, error) {
	size := RequestIDSize + CommandSize + PayloadLenSize + len(frame.Payload)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf := make([]byte, size)
	offset := 0

	binary.LittleEndian.PutUint64(buf[offset:], frame.RequestID)
	offset += RequestIDSize

	buf[offset] = frame.Command
	offset += CommandSize

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(frame.Payload)))
	offset += PayloadLenSize

	if len(frame.Payload) > 0 {
		copy(buf[offset:], frame.Payload)
	}

	return buf, nil
}

// DecodeRequest decodes a request frame
func DecodeRequest(data []byte) (*RequestFrame, error) {
	if len(data) < RequestIDSize+CommandSize+PayloadLenSize {
		return nil, ErrInvalidFrame
	}

	offset := 0
	frame := &RequestFrame{}

	frame.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize

	frame.Command = data[offset]
	offset += CommandSize

	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += PayloadLenSize

	if offset+int(payloadLen) > len(data) {
		return nil, ErrInvalidFrame
	}

	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		copy(frame.Payload, data[offset:offset+int(payloadLen)])
	}

	return frame, nil
}

// EncodeResponse encodes a response frame
func EncodeResponse(frame *ResponseFrame) ([]byte, error) {
	size := RequestIDSize + StatusSize + PayloadLenSize + len(frame.Payload)
	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf := make([]byte, size)
	offset := 0

	binary.LittleEndian.PutUint64(buf[offset:], frame.RequestID)
	offset += RequestIDSize

	buf[offset] = frame.Status
	offset += StatusSize

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(frame.Payload)))
	offset += PayloadLenSize

	if len(frame.Payload) > 0 {
		copy(buf[offset:], frame.Payload)
	}

	return buf, nil
}

// DecodeResponse decodes a response frame
func DecodeResponse(data []byte) (*ResponseFrame, error) {
	if len(data) < RequestIDSize+StatusSize+PayloadLenSize {
		return nil, ErrInvalidFrame
	}

	offset := 0
	frame := &ResponseFrame{}

	frame.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize

	frame.Status = data[offset]
	offset += StatusSize

	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += PayloadLenSize

	if offset+int(payloadLen) > len(data) {
		return nil, ErrInvalidFrame
	}

	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		copy(frame.Payload, data[offset:])
	}

	return frame, nil
}

// readFrame reads a length-prefixed frame from a connection
func readFrame(conn net.Conn) ([]byte, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	// Read frame data
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

// writeFrame writes a length-prefixed frame to a connection
func writeFrame(conn net.Conn, data []byte) error {
	// Write length prefix
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}

	// Write frame data
	if _, err := conn.Write(data); err != nil {
		return err
	}

	return nil
}
