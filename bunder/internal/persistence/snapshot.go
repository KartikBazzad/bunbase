// Package persistence provides snapshot (RDB-like) and AOF (append-only file) for Bunder.
package persistence

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Snapshot writes a full point-in-time dump of KV entries to a single file (RDB-like).
// Format: count(8) + for each key-value keyLen(4)+key+valLen(4)+value.
type Snapshot struct {
	path string
	mu   sync.Mutex
}

// NewSnapshot creates a snapshot writer to the given directory.
func NewSnapshot(dir string) *Snapshot {
	return &Snapshot{path: dir}
}

// Write writes a full snapshot: count(8) + for each key: keyLen(4)+key + valLen(4)+value.
func (s *Snapshot) Write(entries []SnapshotEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.path, 0755); err != nil {
		return err
	}
	name := filepath.Join(s.path, fmt.Sprintf("snapshot-%d.rdb", time.Now().Unix()))
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(len(entries)))
	if _, err := f.Write(buf); err != nil {
		return err
	}
	for _, e := range entries {
		binary.LittleEndian.PutUint32(buf[0:4], uint32(len(e.Key)))
		if _, err := f.Write(buf[:4]); err != nil {
			return err
		}
		if _, err := f.Write(e.Key); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[0:4], uint32(len(e.Value)))
		if _, err := f.Write(buf[:4]); err != nil {
			return err
		}
		if _, err := f.Write(e.Value); err != nil {
			return err
		}
	}
	return f.Sync()
}

// SnapshotEntry is a key-value pair for snapshot.
type SnapshotEntry struct {
	Key   []byte
	Value []byte
}

// Read reads a snapshot file and returns entries.
func ReadSnapshot(path string) ([]SnapshotEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lenBuf [8]byte
	if _, err := f.Read(lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint64(lenBuf[:])
	var out []SnapshotEntry
	for i := uint64(0); i < n; i++ {
		var kl, vl uint32
		if _, err := f.Read(lenBuf[:4]); err != nil {
			return nil, err
		}
		kl = binary.LittleEndian.Uint32(lenBuf[:4])
		key := make([]byte, kl)
		if _, err := f.Read(key); err != nil {
			return nil, err
		}
		if _, err := f.Read(lenBuf[:4]); err != nil {
			return nil, err
		}
		vl = binary.LittleEndian.Uint32(lenBuf[:4])
		val := make([]byte, vl)
		if _, err := f.Read(val); err != nil {
			return nil, err
		}
		out = append(out, SnapshotEntry{Key: key, Value: val})
	}
	return out, nil
}
