package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// SegmentID uniquely identifies a WAL segment file
type SegmentID uint64

// DefaultSegmentSize is the default maximum size for a WAL segment (64MB)
const DefaultSegmentSize = 64 * 1024 * 1024

// Segment represents a single WAL segment file
type Segment struct {
	ID       SegmentID
	file     *os.File
	size     int64
	maxSize  int64
	startLSN LSN
	endLSN   LSN
	mu       sync.RWMutex
}

// NewSegment creates a new WAL segment
func NewSegment(dir string, id SegmentID, startLSN LSN) (*Segment, error) {
	filename := filepath.Join(dir, fmt.Sprintf("wal-%016x.log", id))

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAL segment: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat WAL segment: %w", err)
	}

	return &Segment{
		ID:       id,
		file:     file,
		size:     info.Size(),
		maxSize:  DefaultSegmentSize,
		startLSN: startLSN,
		endLSN:   startLSN,
	}, nil
}

// OpenSegment opens an existing WAL segment
func OpenSegment(dir string, id SegmentID) (*Segment, error) {
	filename := filepath.Join(dir, fmt.Sprintf("wal-%016x.log", id))

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL segment: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat WAL segment: %w", err)
	}

	// For opened segments, we'll need to scan to determine start/end LSN
	// For now, simplified
	return &Segment{
		ID:       id,
		file:     file,
		size:     info.Size(),
		maxSize:  DefaultSegmentSize,
		startLSN: LSN(0),
		endLSN:   LSN(0),
	}, nil
}

// Write writes a record to the segment
func (s *Segment) Write(record *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encode record
	data, err := record.Encode()
	if err != nil {
		return err
	}

	// Write record length first (4 bytes) for easier reading
	lenBuf := make([]byte, 4)
	lenBuf[0] = byte(len(data))
	lenBuf[1] = byte(len(data) >> 8)
	lenBuf[2] = byte(len(data) >> 16)
	lenBuf[3] = byte(len(data) >> 24)

	if _, err := s.file.Write(lenBuf); err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	// Write record data
	if _, err := s.file.Write(data); err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	// Update size and LSN
	s.size += int64(4 + len(data))
	s.endLSN = record.LSN

	return nil
}

// Sync flushes the segment to disk
func (s *Segment) Sync() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}
	return nil
}

// IsFull returns true if the segment has reached its maximum size
func (s *Segment) IsFull() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size >= s.maxSize
}

// Size returns the current size of the segment
func (s *Segment) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// Close closes the segment file
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		if err := s.file.Sync(); err != nil {
			return err
		}
		return s.file.Close()
	}
	return nil
}

// ReadRecords reads all records from the segment
func (s *Segment) ReadRecords() ([]*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Seek to beginning
	if _, err := s.file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("%w: %v", util.ErrDiskReadFailed, err)
	}

	var records []*Record
	lenBuf := make([]byte, 4)

	for {
		// Read record length
		n, err := s.file.Read(lenBuf)
		if err != nil || n == 0 {
			break // EOF or error
		}
		if n != 4 {
			return nil, fmt.Errorf("%w: incomplete length header", util.ErrWALCorrupt)
		}

		// Parse length
		recordLen := int(lenBuf[0]) | int(lenBuf[1])<<8 | int(lenBuf[2])<<16 | int(lenBuf[3])<<24
		if recordLen == 0 || recordLen > 10*1024*1024 { // Sanity check: max 10MB per record
			return nil, fmt.Errorf("%w: invalid record length %d", util.ErrWALCorrupt, recordLen)
		}

		// Read record data
		data := make([]byte, recordLen)
		n, err = s.file.Read(data)
		if err != nil || n != recordLen {
			return nil, fmt.Errorf("%w: incomplete record data", util.ErrWALCorrupt)
		}

		// Decode record
		record, err := Decode(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", util.ErrWALCorrupt, err)
		}

		records = append(records, record)
	}

	return records, nil
}

// GetPath returns the file path of the segment
func (s *Segment) GetPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.file != nil {
		return s.file.Name()
	}
	return ""
}
