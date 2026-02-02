package wal

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// WAL is the write-ahead log for Bunder KV operations.
// Records are appended to segment files (wal-*.log); segments rotate at 64MB.
type WAL struct {
	dir       string
	segment   *Segment
	segmentID uint64
	lsn       atomic.Uint64
	mu        sync.RWMutex
}

// NewWAL creates a new WAL in the given directory.
func NewWAL(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	seg, err := newSegment(dir, 0)
	if err != nil {
		return nil, err
	}
	w := &WAL{dir: dir, segment: seg, segmentID: 0}
	w.lsn.Store(1)
	return w, nil
}

// Append appends a record and returns its LSN.
func (w *WAL) Append(record *Record) (LSN, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	lsn := LSN(w.lsn.Add(1))
	record.LSN = lsn
	if w.segment.isFull() {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	if err := w.segment.write(record); err != nil {
		return 0, err
	}
	return lsn, nil
}

func (w *WAL) rotate() error {
	if err := w.segment.close(); err != nil {
		return err
	}
	w.segmentID++
	seg, err := newSegment(w.dir, w.segmentID)
	if err != nil {
		return err
	}
	w.segment = seg
	return nil
}

// Sync flushes the WAL to disk.
func (w *WAL) Sync() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.segment.sync()
}

// Close closes the WAL.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.segment.close()
}

// ReadAllRecords reads all records from all segment files (for recovery).
func (w *WAL) ReadAllRecords() ([]*Record, error) {
	w.mu.RLock()
	dir := w.dir
	w.mu.RUnlock()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []*Record
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) < 4 || name[:4] != "wal-" {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		seg := &Segment{file: f, max: defaultSegmentSize}
		recs, err := seg.readRecords()
		f.Close()
		if err != nil {
			return nil, err
		}
		all = append(all, recs...)
	}
	return all, nil
}

// CurrentLSN returns the current LSN (next will be CurrentLSN+1).
func (w *WAL) CurrentLSN() LSN {
	return LSN(w.lsn.Load())
}
