// Package docdb implements the coordinator log for multi-partition 2PC.
//
// The coordinator log is the single source of truth for commit/abort decisions.
// Each record is (txID uint64, decision byte, checksum uint32). CRC32 over
// txID+decision detects torn writes. Durability: fsync after append before
// writing OpCommit/OpAbort to partition WALs.
package docdb

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
)

const (
	coordinatorRecordSize          = 8 + 1 + 4 // txID(8) + decision(1) + crc32(4)
	coordinatorDecisionAbort  byte = 0
	coordinatorDecisionCommit byte = 1
)

var coordinatorByteOrder = binary.LittleEndian

// CoordinatorLog is an append-only log of (txID, commit/abort) decisions for 2PC.
// One per LogicalDB; must survive crashes so recovery can resolve in-doubt transactions.
type CoordinatorLog struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	logger *logger.Logger
}

// NewCoordinatorLog creates a coordinator log. Call Open(path) before use.
func NewCoordinatorLog(path string, log *logger.Logger) *CoordinatorLog {
	return &CoordinatorLog{
		path:   path,
		logger: log,
	}
}

// Open creates or opens the coordinator log file. Idempotent; safe to call if already open.
func (c *CoordinatorLog) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(c.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	c.file = f
	return nil
}

// Close closes the coordinator log file.
func (c *CoordinatorLog) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return nil
	}
	err := c.file.Close()
	c.file = nil
	return err
}

// AppendDecision appends (txID, decision) with CRC32 and fsyncs.
// commit true = commit, false = abort. Caller must fsync before writing OpCommit/OpAbort to partitions.
func (c *CoordinatorLog) AppendDecision(txID uint64, commit bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return ErrDBNotOpen
	}
	decision := coordinatorDecisionAbort
	if commit {
		decision = coordinatorDecisionCommit
	}
	buf := make([]byte, coordinatorRecordSize)
	coordinatorByteOrder.PutUint64(buf[0:8], txID)
	buf[8] = decision
	crc := crc32.ChecksumIEEE(buf[:9])
	coordinatorByteOrder.PutUint32(buf[9:13], crc)
	if _, err := c.file.Write(buf); err != nil {
		return err
	}
	return c.file.Sync()
}

// Replay reads the coordinator log and returns txID -> true if commit, false if abort.
// Invalid or torn records are skipped (checksum mismatch). Call at startup before partition replay.
func (c *CoordinatorLog) Replay() (map[uint64]bool, error) {
	c.mu.Lock()
	f := c.file
	path := c.path
	c.mu.Unlock()
	// Replay from file on disk; may be called before Open or with file closed (e.g. at startup we open, replay, then use)
	var file *os.File
	var err error
	if f != nil {
		if _, err = f.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}
		file = f
	} else {
		file, err = os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return make(map[uint64]bool), nil
			}
			return nil, err
		}
		defer file.Close()
		if _, err = file.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}
	}
	decisions := make(map[uint64]bool)
	buf := make([]byte, coordinatorRecordSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if n == 0 {
				break
			}
			// Partial read at EOF is possible; only process full records
			break
		}
		if n < coordinatorRecordSize {
			break
		}
		// Verify checksum
		crc := crc32.ChecksumIEEE(buf[:9])
		stored := coordinatorByteOrder.Uint32(buf[9:13])
		if crc != stored {
			// Torn or corrupt record; skip
			continue
		}
		txID := coordinatorByteOrder.Uint64(buf[0:8])
		decision := buf[8]
		commit := decision == coordinatorDecisionCommit
		decisions[txID] = commit
	}
	// Restore append position if we used the open file
	if f != nil {
		if _, err := f.Seek(0, os.SEEK_END); err != nil {
			return decisions, err
		}
	}
	return decisions, nil
}
