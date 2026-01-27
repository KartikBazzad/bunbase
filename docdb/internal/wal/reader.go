// Package wal implements WAL reading for recovery.
//
// Reader provides:
//   - Sequential WAL record reading
//   - CRC32 checksum validation
//   - Corruption detection (returns error)
//   - Truncated file handling (returns EOF)
//
// Recovery Process:
//  1. Read record length (8 bytes)
//  2. Read remaining record bytes
//  3. Validate CRC32 checksum
//  4. Return record or error
//  5. Truncate at first error (caller responsibility)
//
// Error Handling:
//   - io.EOF: End of file (normal)
//   - ErrCorruptRecord: CRC32 mismatch or invalid length
//   - ErrFileRead: File read error
//
// Thread Safety: NOT thread-safe (single reader per file).
package wal

import (
	"io"
	"os"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

// Reader manages sequential WAL record reading.
//
// It provides:
//   - Forward-only iteration (no random access)
//   - CRC32 validation on each record
//   - Automatic EOF handling
//
// Thread Safety: Not thread-safe (single reader per file).
// Multiple readers require separate instances.
type Reader struct {
	file   *os.File       // Open WAL file handle
	path   string         // WAL file path
	logger *logger.Logger // Structured logging
}

// NewReader creates a new WAL reader.
//
// Parameters:
//   - path: WAL file path (must exist)
//   - log: Logger instance
//
// Returns:
//   - Initialized WAL reader ready for Open()
//
// Note: Reader is not opened until Open() is called.
func NewReader(path string, log *logger.Logger) *Reader {
	return &Reader{
		path:   path,
		logger: log,
	}
}

func (r *Reader) Open() error {
	file, err := os.Open(r.path)
	if err != nil {
		return ErrFileOpen
	}

	r.file = file
	return nil
}

func (r *Reader) Next() (*types.WALRecord, error) {
	if r.file == nil {
		return nil, ErrFileRead
	}

	lenBuf := make([]byte, RecordLenSize)
	_, err := io.ReadFull(r.file, lenBuf)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, ErrCorruptRecord
	}

	recordLen := byteOrder.Uint64(lenBuf)

	if recordLen < RecordLenSize || recordLen > MaxPayloadSize+RecordOverhead {
		return nil, ErrCorruptRecord
	}

	remaining := recordLen - RecordLenSize
	buf := make([]byte, remaining)

	_, err = io.ReadFull(r.file, buf)
	if err != nil {
		return nil, ErrCorruptRecord
	}

	fullRecord := make([]byte, recordLen)
	copy(fullRecord[:RecordLenSize], lenBuf)
	copy(fullRecord[RecordLenSize:], buf)

	record, err := DecodeRecord(fullRecord)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r *Reader) Close() error {
	if r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil
	return err
}
