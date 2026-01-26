package wal

import (
	"os"
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Writer struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	size    uint64
	maxSize uint64
	fsync   bool
	logger  *logger.Logger
}

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
