package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// RecordType represents the type of WAL record
type RecordType byte

const (
	RecordTypeInvalid    RecordType = iota
	RecordTypeInsert                // Document insert
	RecordTypeUpdate                // Document update
	RecordTypeDelete                // Document delete
	RecordTypeCommit                // Transaction commit
	RecordTypeAbort                 // Transaction abort
	RecordTypeCheckpoint            // Checkpoint marker
)

// LSN (Log Sequence Number) uniquely identifies a WAL record
type LSN uint64

// Record represents a single WAL record
type Record struct {
	LSN       LSN        // Log Sequence Number
	TxnID     uint64     // Transaction ID
	Type      RecordType // Record type
	Key       []byte     // Document key
	Value     []byte     // Document value (or delta)
	PrevLSN   LSN        // Previous LSN for this transaction
	Timestamp int64      // Timestamp (Unix nanoseconds)
}

// RecordHeader layout:
// - CRC32 (4 bytes) - checksum of record
// - LSN (8 bytes)
// - TxnID (8 bytes)
// - Type (1 byte)
// - PrevLSN (8 bytes)
// - Timestamp (8 bytes)
// - KeyLen (4 bytes)
// - ValueLen (4 bytes)
// Total: 45 bytes
const RecordHeaderSize = 45

// Encode serializes a WAL record to bytes
func (r *Record) Encode() ([]byte, error) {
	keyLen := len(r.Key)
	valueLen := len(r.Value)
	totalSize := RecordHeaderSize + keyLen + valueLen

	buf := make([]byte, totalSize)
	offset := 4 // Skip CRC32, will write it last

	// Write LSN
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(r.LSN))
	offset += 8

	// Write TxnID
	binary.LittleEndian.PutUint64(buf[offset:offset+8], r.TxnID)
	offset += 8

	// Write Type
	buf[offset] = byte(r.Type)
	offset++

	// Write PrevLSN
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(r.PrevLSN))
	offset += 8

	// Write Timestamp
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(r.Timestamp))
	offset += 8

	// Write KeyLen
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(keyLen))
	offset += 4

	// Write ValueLen
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(valueLen))
	offset += 4

	// Write Key
	copy(buf[offset:offset+keyLen], r.Key)
	offset += keyLen

	// Write Value
	copy(buf[offset:offset+valueLen], r.Value)

	// Calculate and write CRC32 (excluding the CRC field itself)
	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:4], crc)

	return buf, nil
}

// Decode deserializes a WAL record from bytes
func Decode(data []byte) (*Record, error) {
	if len(data) < RecordHeaderSize {
		return nil, fmt.Errorf("invalid record: too short (got %d bytes, need at least %d)", len(data), RecordHeaderSize)
	}

	// Verify CRC32
	expectedCRC := binary.LittleEndian.Uint32(data[0:4])
	actualCRC := crc32.ChecksumIEEE(data[4:])
	if expectedCRC != actualCRC {
		return nil, fmt.Errorf("invalid record: CRC mismatch (expected %d, got %d)", expectedCRC, actualCRC)
	}

	offset := 4

	// Read LSN
	lsn := LSN(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Read TxnID
	txnID := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Read Type
	recordType := RecordType(data[offset])
	offset++

	// Read PrevLSN
	prevLSN := LSN(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Read Timestamp
	timestamp := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Read KeyLen
	keyLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	// Read ValueLen
	valueLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	// Validate lengths
	if offset+keyLen+valueLen != len(data) {
		return nil, fmt.Errorf("invalid record: length mismatch")
	}

	// Read Key
	key := make([]byte, keyLen)
	copy(key, data[offset:offset+keyLen])
	offset += keyLen

	// Read Value
	value := make([]byte, valueLen)
	copy(value, data[offset:offset+valueLen])

	return &Record{
		LSN:       lsn,
		TxnID:     txnID,
		Type:      recordType,
		Key:       key,
		Value:     value,
		PrevLSN:   prevLSN,
		Timestamp: timestamp,
	}, nil
}

// Size returns the size of the encoded record in bytes
func (r *Record) Size() int {
	return RecordHeaderSize + len(r.Key) + len(r.Value)
}

// String returns a human-readable representation of the record
func (r *Record) String() string {
	return fmt.Sprintf("Record{LSN:%d, TxnID:%d, Type:%d, KeyLen:%d, ValueLen:%d}",
		r.LSN, r.TxnID, r.Type, len(r.Key), len(r.Value))
}
