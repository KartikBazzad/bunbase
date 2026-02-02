// Package wire defines the binary network protocol for Bundoc.
//
// Protocol Format:
//
//	[Header (5 bytes)] + [Body (JSON)]
//
// Header:
//   - OpCode (1 byte): Operation type (Insert, Find, etc.)
//   - Length (4 bytes): Uint32 Big-Endian size of Body
//
// Body:
//   - JSON encoded payload corresponding to the OpCode.
package wire

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// OpCode defines the operation type for the wire protocol.
type OpCode uint8

const (
	OpInsert OpCode = 1
	OpFind   OpCode = 2
	OpUpdate OpCode = 3
	OpDelete OpCode = 4

	// Server Responses
	// Server Responses
	OpReply     OpCode = 10
	OpError     OpCode = 11
	OpAuthReply OpCode = 14

	// Raft Consensus (Internal)
	OpRequestVote   OpCode = 12
	OpAppendEntries OpCode = 13

	// Authentication
	OpAuth OpCode = 20
)

// Header is the fixed-size message header (5 bytes)
type Header struct {
	OpCode OpCode
	Length uint32 // Length of the JSON body
}

const HeaderSize = 5

// WriteMessage writes a message (OpCode + Body) to the writer
func WriteMessage(w io.Writer, op OpCode, body interface{}) error {
	// 1. Encode Body
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	// 2. Write Header
	// OpCode (1 byte)
	if _, err := w.Write([]byte{byte(op)}); err != nil {
		return err
	}
	// Length (4 bytes, Big Endian)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(bodyBytes)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}

	// 3. Write Body
	if len(bodyBytes) > 0 {
		if _, err := w.Write(bodyBytes); err != nil {
			return err
		}
	}

	return nil
}

// ReadHeader reads and decoding the message header
func ReadHeader(r io.Reader) (Header, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Header{}, err
	}

	return Header{
		OpCode: OpCode(buf[0]),
		Length: binary.BigEndian.Uint32(buf[1:]),
	}, nil
}

// ReadBody reads the body into the provided interface
func ReadBody(r io.Reader, length uint32, v interface{}) error {
	if length == 0 {
		return nil
	}

	// Limit reader to avoid reading past body
	lr := io.LimitReader(r, int64(length))
	decoder := json.NewDecoder(lr)
	return decoder.Decode(v)
}
