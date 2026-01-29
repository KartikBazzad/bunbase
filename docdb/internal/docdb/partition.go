// Package docdb implements partitioned execution for v0.4.
//
// A Partition owns data, WAL, and index for a subset of documents.
// Partitions do NOT own goroutines; workers pull tasks and lock partitions.
package docdb

import (
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/wal"
)

// Partition represents a single partition within a LogicalDB (v0.4).
//
// A partition:
//   - owns data (datafile)
//   - owns WAL (partition-specific WAL file)
//   - owns index (primary index for documents in this partition)
//   - does NOT own threads/goroutines
//   - does NOT spawn goroutines
//
// All writes require mu (mutex). Reads use immutable index snapshots (lock-free).
type Partition struct {
	id       int               // Partition ID (0..PartitionCount-1)
	mu       sync.Mutex        // Write serialization (exactly one writer at a time)
	queue    chan *Task        // Bounded task queue
	wal      *wal.PartitionWAL // Partition-specific WAL (v0.4)
	dataFile *DataFile         // Partition-specific datafile (or shared with offset tracking)
	index    *Index            // Primary index for this partition
	memory   *memory.Caps      // Memory limit tracking
	logger   *logger.Logger    // Logger
}

// NewPartition creates a new partition.
func NewPartition(id int, queueSize int, memCaps *memory.Caps, log *logger.Logger) *Partition {
	return &Partition{
		id:     id,
		queue:  make(chan *Task, queueSize),
		index:  NewIndex(),
		memory: memCaps,
		logger: log,
	}
}

// ID returns the partition ID.
func (p *Partition) ID() int {
	return p.id
}

// GetIndex returns the partition's index (for reads; use snapshot for lock-free access).
func (p *Partition) GetIndex() *Index {
	return p.index
}

// SetWAL sets the partition's WAL writer.
func (p *Partition) SetWAL(wal *wal.PartitionWAL) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wal = wal
}

// SetDataFile sets the partition's datafile.
func (p *Partition) SetDataFile(dataFile *DataFile) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dataFile = dataFile
}

// GetWAL returns the partition's WAL (caller must hold mu for writes).
func (p *Partition) GetWAL() *wal.PartitionWAL {
	return p.wal
}

// GetDataFile returns the partition's datafile (caller must hold mu for writes).
func (p *Partition) GetDataFile() *DataFile {
	return p.dataFile
}
