package ipc

import (
	"encoding/binary"
	"errors"

	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrInvalidFrame  = errors.New("invalid frame format")
	ErrFrameTooLarge = errors.New("frame too large")
)

const (
	RequestIDSize = 8
	DBIDSize      = 8
	OpCountSize   = 4

	OpTypeSize     = 1
	DocIDSize      = 8
	PayloadLenSize = 4

	MaxFrameSize = 16 * 1024 * 1024
)

const (
	CmdOpenDB  = 1
	CmdCloseDB = 2
	CmdExecute = 3
	CmdStats   = 4
)

type RequestFrame struct {
	RequestID uint64
	DBID      uint64
	Command   uint8
	OpCount   uint32
	Ops       []Operation
}

type Operation struct {
	OpType  types.OperationType
	DocID   uint64
	Payload []byte
}

type ResponseFrame struct {
	RequestID uint64
	Status    types.Status
	Data      []byte
}

func EncodeRequest(frame *RequestFrame) ([]byte, error) {
	var size uint64 = RequestIDSize + DBIDSize + 1 + OpCountSize

	for _, op := range frame.Ops {
		size += OpTypeSize + DocIDSize + PayloadLenSize + uint64(len(op.Payload))
	}

	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf := make([]byte, size)
	offset := 0

	binary.LittleEndian.PutUint64(buf[offset:], frame.RequestID)
	offset += RequestIDSize

	binary.LittleEndian.PutUint64(buf[offset:], frame.DBID)
	offset += DBIDSize

	buf[offset] = frame.Command
	offset += 1

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(frame.Ops)))
	offset += OpCountSize

	for _, op := range frame.Ops {
		buf[offset] = byte(op.OpType)
		offset += OpTypeSize

		binary.LittleEndian.PutUint64(buf[offset:], op.DocID)
		offset += DocIDSize

		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(op.Payload)))
		offset += PayloadLenSize

		if len(op.Payload) > 0 {
			copy(buf[offset:], op.Payload)
			offset += len(op.Payload)
		}
	}

	return buf, nil
}

func DecodeRequest(data []byte) (*RequestFrame, error) {
	if len(data) < RequestIDSize+DBIDSize+1+OpCountSize {
		return nil, ErrInvalidFrame
	}

	offset := 0
	frame := &RequestFrame{}

	frame.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize

	frame.DBID = binary.LittleEndian.Uint64(data[offset:])
	offset += DBIDSize

	frame.Command = data[offset]
	offset += 1

	opCount := binary.LittleEndian.Uint32(data[offset:])
	offset += OpCountSize

	frame.Ops = make([]Operation, opCount)

	for i := 0; i < int(opCount); i++ {
		if offset+OpTypeSize+DocIDSize+PayloadLenSize > len(data) {
			return nil, ErrInvalidFrame
		}

		frame.Ops[i].OpType = types.OperationType(data[offset])
		offset += OpTypeSize

		frame.Ops[i].DocID = binary.LittleEndian.Uint64(data[offset:])
		offset += DocIDSize

		payloadLen := binary.LittleEndian.Uint32(data[offset:])
		offset += PayloadLenSize

		if offset+int(payloadLen) > len(data) {
			return nil, ErrInvalidFrame
		}

		if payloadLen > 0 {
			frame.Ops[i].Payload = make([]byte, payloadLen)
			copy(frame.Ops[i].Payload, data[offset:offset+int(payloadLen)])
			offset += int(payloadLen)
		}
	}

	return frame, nil
}

func EncodeResponse(frame *ResponseFrame) ([]byte, error) {
	size := RequestIDSize + 1 + 4 + len(frame.Data)

	if size > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf := make([]byte, size)
	offset := 0

	binary.LittleEndian.PutUint64(buf[offset:], frame.RequestID)
	offset += RequestIDSize

	buf[offset] = byte(frame.Status)
	offset += 1

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(frame.Data)))
	offset += 4

	if len(frame.Data) > 0 {
		copy(buf[offset:], frame.Data)
	}

	return buf, nil
}

func DecodeResponse(data []byte) (*ResponseFrame, error) {
	if len(data) < RequestIDSize+1+4 {
		return nil, ErrInvalidFrame
	}

	offset := 0
	frame := &ResponseFrame{}

	frame.RequestID = binary.LittleEndian.Uint64(data[offset:])
	offset += RequestIDSize

	frame.Status = types.Status(data[offset])
	offset += 1

	dataLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if offset+int(dataLen) > len(data) {
		return nil, ErrInvalidFrame
	}

	if dataLen > 0 {
		frame.Data = make([]byte, dataLen)
		copy(frame.Data, data[offset:])
	}

	return frame, nil
}
