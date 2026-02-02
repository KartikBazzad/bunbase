// Package wal implements Write-Ahead Logging for Bunder: append-only records with CRC32 and LSN.
package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// RecordType identifies the kind of WAL record (Set, Del, Expire).
type RecordType byte

const (
	RecordTypeInvalid RecordType = iota
	RecordTypeSet                // SET key value
	RecordTypeDel                // DEL key
	RecordTypeExpire             // EXPIRE key ttl_seconds
)

// LSN is the log sequence number.
type LSN uint64

// Record is a single WAL record for KV operations.
type Record struct {
	LSN   LSN
	Type  RecordType
	Key   []byte
	Value []byte
}

// Header: CRC32(4) + LSN(8) + Type(1) + KeyLen(4) + ValueLen(4) = 21
const recordHeaderSize = 21

// Encode serializes the record to bytes.
func (r *Record) Encode() ([]byte, error) {
	kl, vl := len(r.Key), len(r.Value)
	total := recordHeaderSize + kl + vl
	buf := make([]byte, total)
	off := 4 // skip CRC
	binary.LittleEndian.PutUint64(buf[off:off+8], uint64(r.LSN))
	off += 8
	buf[off] = byte(r.Type)
	off++
	binary.LittleEndian.PutUint32(buf[off:off+4], uint32(kl))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:off+4], uint32(vl))
	off += 4
	copy(buf[off:off+kl], r.Key)
	off += kl
	copy(buf[off:off+vl], r.Value)
	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:4], crc)
	return buf, nil
}

// Decode deserializes a record from bytes.
func DecodeRecord(data []byte) (*Record, error) {
	if len(data) < recordHeaderSize {
		return nil, fmt.Errorf("record too short: %d", len(data))
	}
	expected := binary.LittleEndian.Uint32(data[0:4])
	actual := crc32.ChecksumIEEE(data[4:])
	if expected != actual {
		return nil, fmt.Errorf("record CRC mismatch")
	}
	off := 4
	lsn := LSN(binary.LittleEndian.Uint64(data[off : off+8]))
	off += 8
	typ := RecordType(data[off])
	off++
	kl := binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	vl := binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	if int(kl)+int(vl) > len(data)-off {
		return nil, fmt.Errorf("record length overflow")
	}
	key := make([]byte, kl)
	copy(key, data[off:off+int(kl)])
	off += int(kl)
	val := make([]byte, vl)
	copy(val, data[off:off+int(vl)])
	return &Record{LSN: lsn, Type: typ, Key: key, Value: val}, nil
}
