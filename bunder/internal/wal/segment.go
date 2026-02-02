package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const defaultSegmentSize = 64 * 1024 * 1024 // 64MB; segment rotates when full

// Segment is a single WAL file (wal-*.log). Records are length(4)+encoded record.
type Segment struct {
	file *os.File
	size int64
	max  int64
	mu   sync.RWMutex
}

func newSegment(dir string, id uint64) (*Segment, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	name := filepath.Join(dir, fmt.Sprintf("wal-%016x.log", id))
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	info, _ := f.Stat()
	return &Segment{file: f, size: info.Size(), max: defaultSegmentSize}, nil
}

func openSegment(dir string, id uint64) (*Segment, error) {
	name := filepath.Join(dir, fmt.Sprintf("wal-%016x.log", id))
	f, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	info, _ := f.Stat()
	return &Segment{file: f, size: info.Size(), max: defaultSegmentSize}, nil
}

func (s *Segment) write(record *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := record.Encode()
	if err != nil {
		return err
	}
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := s.file.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := s.file.Write(data); err != nil {
		return err
	}
	s.size += 4 + int64(len(data))
	return nil
}

func (s *Segment) sync() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.file.Sync()
}

func (s *Segment) isFull() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size >= s.max
}

func (s *Segment) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		_ = s.file.Sync()
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

func (s *Segment) readRecords() ([]*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, err := s.file.Seek(0, 0); err != nil {
		return nil, err
	}
	var out []*Record
	var lenBuf [4]byte
	for {
		n, err := s.file.Read(lenBuf[:])
		if err != nil || n == 0 {
			break
		}
		if n != 4 {
			return nil, fmt.Errorf("incomplete length")
		}
		recLen := binary.LittleEndian.Uint32(lenBuf[:])
		if recLen == 0 || recLen > 10*1024*1024 {
			return nil, fmt.Errorf("invalid record length %d", recLen)
		}
		data := make([]byte, recLen)
		n, err = s.file.Read(data)
		if err != nil || n != int(recLen) {
			return nil, fmt.Errorf("incomplete record")
		}
		rec, err := DecodeRecord(data)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}
