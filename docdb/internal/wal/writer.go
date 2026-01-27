// Package wal implements Write-Ahead Log for durability.
//
// WAL provides:
//   - Append-only writes (no overwrites)
//   - Binary record format with CRC32 checksums
//   - Optional fsync on every write (durability)
//   - Size tracking (for rotation warnings)
//   - Crash recovery (via reader)
//
// WAL Format (per record):
//
//	[8 bytes: record_len] [8 bytes: tx_id] [8 bytes: db_id]
//	[1 byte: op_type] [8 bytes: doc_id] [4 bytes: payload_len]
//	[N bytes: payload] [4 bytes: crc32]
//
// Durability Guarantees:
//   - If fsync enabled: Record is on disk after Write() returns
//   - If fsync disabled: Record is in OS buffer (may be lost on crash)
//   - CRC32 detects corruption on replay
//   - Truncation at first corrupt record (partial recovery)
//
// Thread Safety: All methods are thread-safe (mu protects file).
package wal

import (
	"os"
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

// Writer manages append-only WAL file.
//
// It provides:
//   - Atomic record writes (with optional fsync)
//   - CRC32 checksum calculation
//   - Size tracking (for rotation warnings)
//   - Thread-safe file operations
//
// Thread Safety: All methods are thread-safe via mu.
type Writer struct {
	mu      sync.Mutex     // Protects all file operations
	file    *os.File       // Open WAL file handle (append mode)
	path    string         // WAL file path
	size    uint64         // Current file size (in bytes)
	maxSize uint64         // Maximum size before warning (0 = unlimited)
	fsync   bool           // If true, fsync after each write
	logger  *logger.Logger // Structured logging
}

// NewWriter creates a new WAL writer.
//
// Parameters:
//   - path: WAL file path (will be created if doesn't exist)
//   - maxSize: Maximum file size before logging warning (0 = no limit)
//   - fsync: If true, call file.Sync() after each write (slower, more durable)
//   - log: Logger instance
//
// Returns:
//   - Initialized WAL writer ready for Open()
//
// Note: Writer is not opened until Open() is called.
func NewWriter(path string, maxSize uint64, fsync bool, log *logger.Logger) *Writer {
	return &Writer{
		path:    path,
		maxSize: maxSize,
		fsync:   fsync,
		logger:  log,
	}
}

func (w *Writer) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.OpenFile(w.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	w.file = file
	w.size = uint64(info.Size())

	return nil
}

func (w *Writer) Write(txID, dbID, docID uint64, opType types.OperationType, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	encoded, err := EncodeRecord(txID, dbID, docID, opType, payload)
	if err != nil {
		return err
	}

	if w.maxSize > 0 && w.size+uint64(len(encoded)) > w.maxSize {
		w.logger.Warn("WAL file approaching size limit, rotation not implemented in v0")
	}

	n, err := w.file.Write(encoded)
	if err != nil {
		return ErrFileWrite
	}

	w.size += uint64(n)

	if w.fsync {
		if err := w.file.Sync(); err != nil {
			return ErrFileSync
		}
	}

	return nil
}

func (w *Writer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	return w.file.Sync()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	if err := w.file.Sync(); err != nil {
		return err
	}

	if err := w.file.Close(); err != nil {
		return err
	}

	w.file = nil
	return nil
}

func (w *Writer) Size() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}
