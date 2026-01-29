// Package docdb implements core document database engine.
//
// LogicalDB is a single database instance that manages:
//   - Append-only data file storage
//   - Write-ahead log (WAL) for durability
//   - Sharded in-memory index for fast lookups
//   - MVCC-lite for snapshot-based reads
//   - Transaction management for atomic operations
//
// Each LogicalDB is isolated and has its own:
//   - Data file (<dbname>.data)
//   - WAL file (<dbname>.wal)
//   - In-memory index (recovered from WAL on open)
//
// Commit ordering invariant:
// 1. Write WAL record (includes fsync if enabled)
// 2. Update index (making transaction visible)
//
// This ensures no phantom visibility after crash:
// - If crash occurs before index update: WAL persists, data is recovered on restart
// - If crash occurs after index update: WAL already persisted, consistent state
// - Transaction is only visible after WAL is durable
package docdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/metrics"
	"github.com/kartikbazzad/docdb/internal/query"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/internal/wal"
)

// LogicalDB represents a single logical database instance (v0.4: partitioned).
//
// It manages the complete lifecycle of documents within this database,
// including storage, indexing, transaction management, and recovery.
//
// v0.4 Architecture:
//   - LogicalDB is a partitioned execution domain
//   - Each partition owns data, WAL, and index
//   - Workers are NOT bound to partitions; they pull tasks and lock partitions
//   - Exactly one writer per partition at a time (enforced by partition.mu)
//   - Unlimited readers (lock-free via immutable index snapshots)
//
// Thread Safety: All public methods are safe for concurrent use.
// Partition-level locking ensures exactly one writer per partition.
type LogicalDB struct {
	mu     sync.RWMutex // Protects LogicalDB-level state (partitions list, etc.)
	dbID   uint64       // Unique database identifier
	dbName string       // Human-readable database name

	// v0.4: Partitioned execution
	partitions []*Partition            // Partitions (nil if PartitionCount=1, using legacy mode)
	dbConfig   *config.LogicalDBConfig // LogicalDB-specific config (partitioning, workers)
	workerPool WorkerPool              // Worker pool for this LogicalDB

	// Legacy mode (PartitionCount=1): single WAL, datafile, index
	dataFile *DataFile   // Legacy: single datafile (used when partitions == nil)
	wal      *wal.Writer // Legacy: single WAL (used when partitions == nil)
	index    *Index      // Legacy: single index (used when partitions == nil)

	mvcc           *MVCC                  // Multi-version concurrency control
	txManager      *TransactionManager    // Transaction lifecycle management
	memory         *memory.Caps           // Memory limit tracking
	pool           *memory.BufferPool     // Efficient buffer allocation
	cfg            *config.Config         // Global database configuration
	logger         *logger.Logger         // Structured logging
	closed         bool                   // True if database is closed
	dataDir        string                 // Directory for data files
	walDir         string                 // Directory for WAL files
	txnsCommitted  uint64                 // Count of committed transactions
	lastCompaction time.Time              // Timestamp of last compaction
	checkpointMgr  *wal.CheckpointManager // Legacy checkpoint manager (for PartitionCount=1)
	errorTracker   *errors.ErrorTracker   // Error tracking for observability
	healingService *HealingService        // Automatic document healing service
	walTrimmer     *wal.Trimmer           // Legacy WAL trimmer (for PartitionCount=1)
	collections    *CollectionRegistry    // Collection management
	querySemaphore chan struct{}          // Phase D.8: Semaphore for concurrent query limiting
}

func NewLogicalDB(dbID uint64, dbName string, cfg *config.Config, memCaps *memory.Caps, pool *memory.BufferPool, log *logger.Logger) *LogicalDB {
	maxConcurrentQueries := 100 // Default when cfg nil or not set
	if cfg != nil && cfg.Query.MaxConcurrentQueries > 0 {
		maxConcurrentQueries = cfg.Query.MaxConcurrentQueries
	}
	querySemaphore := make(chan struct{}, maxConcurrentQueries)

	db := &LogicalDB{
		dbID:           dbID,
		dbName:         dbName,
		mvcc:           NewMVCC(),
		txManager:      NewTransactionManager(NewMVCC()),
		memory:         memCaps,
		pool:           pool,
		cfg:            cfg,
		logger:         log,
		closed:         false,
		errorTracker:   errors.NewErrorTracker(),
		collections:    NewCollectionRegistry(log),
		querySemaphore: querySemaphore, // Phase D.8: Initialize semaphore
	}

	// Use default LogicalDB config (PartitionCount=1 for backward compatibility)
	db.dbConfig = config.DefaultLogicalDBConfig()
	db.dbConfig.PartitionCount = 1 // v0.3 compatibility: single partition

	return db
}

// NewLogicalDBWithConfig creates a LogicalDB with explicit LogicalDBConfig (v0.4).
func NewLogicalDBWithConfig(dbID uint64, dbName string, cfg *config.Config, dbCfg *config.LogicalDBConfig, memCaps *memory.Caps, pool *memory.BufferPool, log *logger.Logger) *LogicalDB {
	maxConcurrentQueries := 100 // Default when cfg nil or not set
	if cfg != nil && cfg.Query.MaxConcurrentQueries > 0 {
		maxConcurrentQueries = cfg.Query.MaxConcurrentQueries
	}
	querySemaphore := make(chan struct{}, maxConcurrentQueries)

	db := &LogicalDB{
		dbID:           dbID,
		dbName:         dbName,
		dbConfig:       dbCfg,
		mvcc:           NewMVCC(),
		txManager:      NewTransactionManager(NewMVCC()),
		memory:         memCaps,
		pool:           pool,
		cfg:            cfg,
		logger:         log,
		closed:         false,
		errorTracker:   errors.NewErrorTracker(),
		collections:    NewCollectionRegistry(log),
		querySemaphore: querySemaphore, // Phase D.8: Initialize semaphore
	}

	return db
}

// validateJSONPayload enforces the engine-level JSON-only invariant.
// It mirrors validation behavior in the IPC layer and client, ensuring
// that invalid JSON never reaches the WAL or data files.
func validateJSONPayload(payload []byte) error {
	if len(payload) == 0 {
		return errors.ErrInvalidJSON
	}

	if !utf8.Valid(payload) {
		return errors.ErrInvalidJSON
	}

	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return errors.ErrInvalidJSON
	}

	return nil
}

// PartitionCount returns the number of partitions (0 or 1 = legacy mode).
func (db *LogicalDB) PartitionCount() int {
	if db.dbConfig == nil {
		return 1
	}
	return db.dbConfig.PartitionCount
}

// SubmitTaskAndWait submits a task to the LogicalDB worker pool and blocks until the result is ready.
// Used by the pool when PartitionCount > 1. Returns the result from the worker.
func (db *LogicalDB) SubmitTaskAndWait(task *Task) *Result {
	if db.workerPool == nil {
		return &Result{Status: types.StatusError, Error: ErrDBNotOpen}
	}
	db.workerPool.Submit(task)
	return <-task.ResultCh
}

// getPartition returns the partition for a given partition ID (v0.4).
// Returns nil if partition ID is invalid or partitions not initialized.
func (db *LogicalDB) getPartition(partitionID int) *Partition {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.partitions == nil || partitionID < 0 || partitionID >= len(db.partitions) {
		return nil
	}
	return db.partitions[partitionID]
}

// executeOnPartition executes a task on a partition (caller must hold partition.mu).
// This is called by worker pool after locking the partition.
func (db *LogicalDB) executeOnPartition(partition *Partition, task *Task) *Result {
	if db.closed {
		return &Result{Status: types.StatusError, Error: ErrDBNotOpen}
	}

	collection := task.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	if !db.collections.Exists(collection) {
		return &Result{Status: types.StatusError, Error: errors.ErrCollectionNotFound}
	}

	switch task.Op {
	case types.OpCreate:
		return db.executeCreateOnPartition(partition, collection, task.DocID, task.Payload)
	case types.OpRead:
		return db.executeReadOnPartition(partition, collection, task.DocID)
	case types.OpUpdate:
		return db.executeUpdateOnPartition(partition, collection, task.DocID, task.Payload)
	case types.OpDelete:
		return db.executeDeleteOnPartition(partition, collection, task.DocID)
	case types.OpPatch:
		return db.executePatchOnPartition(partition, collection, task.DocID, task.PatchOps)
	default:
		return &Result{Status: types.StatusError, Error: errors.ErrUnknownOperation}
	}
}

func (db *LogicalDB) executeCreateOnPartition(partition *Partition, collection string, docID uint64, payload []byte) *Result {
	if err := db.checkWALSizeLimit(); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	if err := validateJSONPayload(payload); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}

	version, exists := partition.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if exists && version.DeletedTxID == nil {
		return &Result{Status: types.StatusConflict, Error: ErrDocAlreadyExists}
	}

	memoryNeeded := uint64(len(payload))
	if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
		return &Result{Status: types.StatusMemoryLimit, Error: ErrMemoryLimit}
	}

	dataFile := partition.GetDataFile()
	offset, err := dataFile.Write(payload)
	if err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: err}
	}

	txID := db.mvcc.NextTxID()
	newVersion := db.mvcc.CreateVersion(docID, txID, offset, uint32(len(payload)))

	wal := partition.GetWAL()
	if err := wal.Write(txID, db.dbID, collection, docID, types.OpCreate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL write: %w", err)}
	}
	if err := wal.Write(txID, db.dbID, "", 0, types.OpCommit, nil); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL commit marker: %w", err)}
	}

	db.txnsCommitted++
	partition.index.Set(collection, newVersion)
	db.collections.IncrementDocCount(collection)
	return &Result{Status: types.StatusOK}
}

func (db *LogicalDB) executeReadOnPartition(partition *Partition, collection string, docID uint64) *Result {
	snapshotTxID := db.mvcc.CurrentSnapshot()
	version, exists := partition.index.Get(collection, docID, snapshotTxID)
	if !exists || version.DeletedTxID != nil {
		return &Result{Status: types.StatusNotFound, Error: ErrDocNotFound}
	}

	data, err := partition.GetDataFile().Read(version.Offset, version.Length)
	if err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	return &Result{Status: types.StatusOK, Data: data}
}

// executeReadOnPartitionLockFree performs a read without holding partition.mu (lock-free / snapshot read).
// Caller must not hold partition write lock. Used by worker pool for OpRead tasks.
func (db *LogicalDB) executeReadOnPartitionLockFree(partition *Partition, collection string, docID uint64) *Result {
	if db.closed {
		return &Result{Status: types.StatusError, Error: ErrDBNotOpen}
	}
	if collection == "" {
		collection = DefaultCollection
	}
	if !db.collections.Exists(collection) {
		return &Result{Status: types.StatusError, Error: errors.ErrCollectionNotFound}
	}
	snapshotTxID := db.mvcc.CurrentSnapshot()
	version, exists := partition.index.Get(collection, docID, snapshotTxID)
	if !exists || version.DeletedTxID != nil {
		return &Result{Status: types.StatusNotFound, Error: ErrDocNotFound}
	}
	data, err := partition.GetDataFile().Read(version.Offset, version.Length)
	if err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	return &Result{Status: types.StatusOK, Data: data}
}

func (db *LogicalDB) executeUpdateOnPartition(partition *Partition, collection string, docID uint64, payload []byte) *Result {
	if err := db.checkWALSizeLimit(); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	if err := validateJSONPayload(payload); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}

	version, exists := partition.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return &Result{Status: types.StatusNotFound, Error: ErrDocNotFound}
	}

	memoryNeeded := uint64(len(payload))
	oldMemory := uint64(version.Length)
	if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
		return &Result{Status: types.StatusMemoryLimit, Error: ErrMemoryLimit}
	}

	dataFile := partition.GetDataFile()
	offset, err := dataFile.Write(payload)
	if err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: err}
	}

	txID := db.mvcc.NextTxID()
	newVersion := db.mvcc.UpdateVersion(version, txID, offset, uint32(len(payload)))

	wal := partition.GetWAL()
	if err := wal.Write(txID, db.dbID, collection, docID, types.OpUpdate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL write: %w", err)}
	}
	if err := wal.Write(txID, db.dbID, "", 0, types.OpCommit, nil); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL commit marker: %w", err)}
	}

	db.txnsCommitted++
	partition.index.Set(collection, newVersion)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}
	return &Result{Status: types.StatusOK}
}

func (db *LogicalDB) executeDeleteOnPartition(partition *Partition, collection string, docID uint64) *Result {
	if err := db.checkWALSizeLimit(); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	version, exists := partition.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return &Result{Status: types.StatusNotFound, Error: ErrDocNotFound}
	}

	oldMemory := uint64(version.Length)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}

	txID := db.mvcc.NextTxID()
	deleteVersion := db.mvcc.DeleteVersion(docID, txID)

	wal := partition.GetWAL()
	if err := wal.Write(txID, db.dbID, collection, docID, types.OpDelete, nil); err != nil {
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL write: %w", err)}
	}
	if err := wal.Write(txID, db.dbID, "", 0, types.OpCommit, nil); err != nil {
		return &Result{Status: types.StatusError, Error: fmt.Errorf("WAL commit marker: %w", err)}
	}

	db.txnsCommitted++
	partition.index.Set(collection, deleteVersion)
	db.collections.DecrementDocCount(collection)
	return &Result{Status: types.StatusOK}
}

func (db *LogicalDB) executePatchOnPartition(partition *Partition, collection string, docID uint64, patchOps []types.PatchOperation) *Result {
	if len(patchOps) == 0 {
		return &Result{Status: types.StatusError, Error: errors.ErrInvalidPatch}
	}

	version, exists := partition.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return &Result{Status: types.StatusNotFound, Error: ErrDocNotFound}
	}

	data, err := partition.GetDataFile().Read(version.Offset, version.Length)
	if err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}

	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return &Result{Status: types.StatusError, Error: errors.ErrInvalidJSON}
	}
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return &Result{Status: types.StatusError, Error: errors.ErrNotJSONObject}
	}

	for _, op := range patchOps {
		path, err := ParsePath(op.Path)
		if err != nil {
			return &Result{Status: types.StatusError, Error: fmt.Errorf("invalid path %q: %w", op.Path, err)}
		}
		switch op.Op {
		case "set":
			if op.Value == nil {
				return &Result{Status: types.StatusError, Error: errors.ErrInvalidPatch}
			}
			if err := SetValue(docMap, path, op.Value); err != nil {
				return &Result{Status: types.StatusError, Error: err}
			}
		case "delete":
			if err := DeleteValue(docMap, path); err != nil {
				return &Result{Status: types.StatusError, Error: err}
			}
		case "insert":
			if op.Value == nil {
				return &Result{Status: types.StatusError, Error: errors.ErrInvalidPatch}
			}
			if len(path) == 0 {
				return &Result{Status: types.StatusError, Error: errors.ErrInvalidPatch}
			}
			indexStr := path[len(path)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return &Result{Status: types.StatusError, Error: errors.ErrInvalidPatch}
			}
			if err := InsertValue(docMap, path[:len(path)-1], index, op.Value); err != nil {
				return &Result{Status: types.StatusError, Error: err}
			}
		default:
			return &Result{Status: types.StatusError, Error: fmt.Errorf("unknown patch op %q", op.Op)}
		}
	}

	updatedPayload, err := json.Marshal(docMap)
	if err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}
	if err := validateJSONPayload(updatedPayload); err != nil {
		return &Result{Status: types.StatusError, Error: err}
	}

	return db.executeUpdateOnPartition(partition, collection, docID, updatedPayload)
}

func (db *LogicalDB) Open(dataDir string, walDir string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.dataDir = dataDir
	db.walDir = walDir

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.MkdirAll(walDir, 0755); err != nil {
		return fmt.Errorf("failed to create WAL directory: %w", err)
	}

	partitionCount := db.dbConfig.PartitionCount
	if partitionCount <= 0 {
		partitionCount = 1
	}

	// v0.3 detection: if single WAL file exists (dbname.wal), use legacy mode even if config says > 1
	legacyWAL := filepath.Join(walDir, fmt.Sprintf("%s.wal", db.dbName))
	p0WAL := filepath.Join(walDir, "p0.wal")
	if _, errLegacy := os.Stat(legacyWAL); errLegacy == nil {
		if _, errP0 := os.Stat(p0WAL); os.IsNotExist(errP0) {
			// v0.3 layout: single WAL file, no partitioned WALs → force PartitionCount=1
			partitionCount = 1
			db.dbConfig.PartitionCount = 1
		}
	}

	// v0.4: Initialize partitions if PartitionCount > 1
	if partitionCount > 1 {
		return db.openPartitioned(dataDir, walDir, partitionCount)
	}

	// Legacy mode (PartitionCount=1): single WAL, datafile, index (v0.3 compatible)
	return db.openLegacy(dataDir, walDir)
}

// openLegacy opens a LogicalDB in legacy mode (PartitionCount=1, v0.3 compatibility).
func (db *LogicalDB) openLegacy(dataDir string, walDir string) error {
	dbFile := filepath.Join(dataDir, fmt.Sprintf("%s.data", db.dbName))
	db.dataFile = NewDataFile(dbFile, db.logger)
	if err := db.dataFile.Open(); err != nil {
		return fmt.Errorf("failed to open data file: %w", err)
	}

	walFile := filepath.Join(walDir, fmt.Sprintf("%s.wal", db.dbName))
	db.wal = wal.NewWriterFromConfig(walFile, db.dbID, db.cfg.WAL.MaxFileSizeMB*1024*1024, &db.cfg.WAL, db.logger)
	if err := db.wal.Open(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}

	// Initialize index for legacy mode
	db.index = NewIndex()

	// Initialize checkpoint manager
	db.checkpointMgr = wal.NewCheckpointManager(
		db.cfg.WAL.Checkpoint.IntervalMB,
		db.cfg.WAL.Checkpoint.AutoCreate,
		db.cfg.WAL.Checkpoint.MaxCheckpoints,
		db.logger,
	)
	db.wal.SetCheckpointManager(db.checkpointMgr)

	// Initialize WAL trimmer if enabled
	if db.cfg.WAL.TrimAfterCheckpoint {
		db.walTrimmer = wal.NewTrimmer(walDir, db.dbName, db.logger)
	}

	// Ensure default collection exists
	db.collections.EnsureDefault()

	if err := db.replayWAL(); err != nil {
		db.logger.Warn("Failed to replay WAL: %v", err)
	}

	// Start healing service if enabled
	if db.cfg.Healing.Enabled {
		db.healingService = NewHealingService(db, &db.cfg.Healing, db.logger)
		db.healingService.Start()
	}

	db.logger.Info("Opened database (legacy mode): %s (id=%d)", db.dbName, db.dbID)
	return nil
}

// openPartitioned opens a LogicalDB in partitioned mode (v0.4, PartitionCount > 1).
// Partition WALs live under walDir/dbName/ (p0.wal, p1.wal, ...) so each DB has its own WAL files.
func (db *LogicalDB) openPartitioned(dataDir string, walDir string, partitionCount int) error {
	// Phase D.8: Validate partition count limit (defensive: cfg may be nil)
	if db.cfg != nil && db.cfg.Query.MaxPartitionsPerDB > 0 && partitionCount > db.cfg.Query.MaxPartitionsPerDB {
		return ErrTooManyPartitions
	}

	// Per-database WAL directory (same idea as legacy dbname.wal)
	partitionWalDir := filepath.Join(walDir, db.dbName)
	if err := os.MkdirAll(partitionWalDir, 0755); err != nil {
		return fmt.Errorf("failed to create partition WAL directory: %w", err)
	}
	checkpointDir := filepath.Join(partitionWalDir, "checkpoints")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Create partitions
	db.partitions = make([]*Partition, partitionCount)
	for i := 0; i < partitionCount; i++ {
		db.partitions[i] = NewPartition(i, db.dbConfig.QueueSize, db.memory, db.logger)

		// Create partition WAL under walDir/dbName/p{i}.wal
		walPath := filepath.Join(partitionWalDir, fmt.Sprintf("p%d.wal", i))
		maxSize := db.dbConfig.MaxSegmentSize
		if maxSize == 0 {
			maxSize = int64(db.cfg.WAL.MaxFileSizeMB * 1024 * 1024)
		}
		partitionWAL := wal.NewPartitionWAL(i, walPath, uint64(maxSize), &db.cfg.WAL, db.logger)
		partitionWAL.GetCheckpointManager().SetCheckpointPath(checkpointDir)
		// Set fsync callback for metrics (Phase C.5)
		partitionIDStr := strconv.Itoa(i)
		partitionWAL.SetFsyncCallback(func(duration time.Duration) {
			metrics.RecordPartitionWALFsync(db.dbName, partitionIDStr, duration)
		})
		if err := partitionWAL.Open(); err != nil {
			return fmt.Errorf("failed to open partition %d WAL: %w", i, err)
		}
		db.partitions[i].SetWAL(partitionWAL)

		// Create partition datafile
		partitionDataFile := NewDataFile(
			filepath.Join(dataDir, fmt.Sprintf("%s_p%d.data", db.dbName, i)),
			db.logger,
		)
		if err := partitionDataFile.Open(); err != nil {
			return fmt.Errorf("failed to open partition %d datafile: %w", i, err)
		}
		db.partitions[i].SetDataFile(partitionDataFile)
	}

	// Create worker pool
	db.workerPool = NewWorkerPool(db, db.dbConfig, db.logger)
	db.workerPool.Start()

	// Ensure default collection exists
	db.collections.EnsureDefault()

	// Parallel partition recovery: replay each partition's WAL (v0.4) and rebuild index.
	var replayWg sync.WaitGroup
	maxTxIDs := make([]uint64, partitionCount)
	maxLSNs := make([]uint64, partitionCount)
	replayErrs := make([]error, partitionCount)
	for i := 0; i < partitionCount; i++ {
		i := i
		replayWg.Add(1)
		go func() {
			defer replayWg.Done()
			walPath := filepath.Join(partitionWalDir, fmt.Sprintf("p%d.wal", i))
			txID, lsn, err := db.replayPartitionWALForPartition(db.partitions[i], walPath)
			maxTxIDs[i] = txID
			maxLSNs[i] = lsn
			replayErrs[i] = err
		}()
	}
	replayWg.Wait()
	for _, err := range replayErrs {
		if err != nil {
			return fmt.Errorf("partition recovery: %w", err)
		}
	}
	// Set MVCC to max txID across all partitions + 1
	globalMaxTxID := uint64(0)
	for _, txID := range maxTxIDs {
		if txID > globalMaxTxID {
			globalMaxTxID = txID
		}
	}
	if globalMaxTxID > 0 {
		db.mvcc.SetCurrentTxID(globalMaxTxID + 1)
	}
	// Set each partition WAL's next LSN so new writes use maxLSN+1
	for i := 0; i < partitionCount; i++ {
		if db.partitions[i].GetWAL() != nil && maxLSNs[i] > 0 {
			db.partitions[i].GetWAL().SetNextLSN(maxLSNs[i])
		}
	}

	db.logger.Info("Opened database (partitioned mode): %s (id=%d, partitions=%d)", db.dbName, db.dbID, partitionCount)
	return nil
}

// replayPartitionWALForPartition replays one partition's WAL (v0.4), applies committed
// records to the partition's datafile and index, and syncs the datafile.
// Returns the max txID, max LSN seen, and any error.
func (db *LogicalDB) replayPartitionWALForPartition(partition *Partition, walPath string) (maxTxID, maxLSN uint64, err error) {
	replayStart := time.Now()
	defer func() {
		replayDuration := time.Since(replayStart)
		metrics.RecordPartitionReplay(db.dbName, strconv.Itoa(partition.ID()), replayDuration)
	}()
	txRecords := make(map[uint64][]*types.WALRecord)
	committed := make(map[uint64]bool)
	lastCheckpointTxID := uint64(0)

	err = wal.ReplayPartitionWAL(walPath, db.logger, func(rec *types.WALRecord) error {
		if rec == nil {
			return nil
		}
		if rec.LSN > maxLSN {
			maxLSN = rec.LSN
		}
		txID := rec.TxID
		if rec.OpType == types.OpCheckpoint {
			if txID > lastCheckpointTxID {
				lastCheckpointTxID = txID
			}
			return nil
		}
		if txID > maxTxID {
			maxTxID = txID
		}
		switch rec.OpType {
		case types.OpCommit:
			committed[txID] = true
		default:
			txRecords[txID] = append(txRecords[txID], rec)
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	dataFile := partition.GetDataFile()
	index := partition.GetIndex()

	// Apply transactions in txID order so Create before Delete is preserved.
	txIDs := make([]uint64, 0, len(txRecords))
	for txID := range txRecords {
		if !committed[txID] {
			continue
		}
		if lastCheckpointTxID > 0 && txID <= lastCheckpointTxID {
			continue
		}
		txIDs = append(txIDs, txID)
	}
	sort.Slice(txIDs, func(i, j int) bool { return txIDs[i] < txIDs[j] })
	for _, txID := range txIDs {
		recs := txRecords[txID]
		for _, rec := range recs {
			checkRecoveryCommittedOnly(txID, committed)
			collection := rec.Collection
			if collection == "" {
				collection = DefaultCollection
			}
			if !db.collections.Exists(collection) {
				db.collections.mu.Lock()
				db.collections.collections[collection] = &types.CollectionMetadata{
					Name:      collection,
					CreatedAt: time.Now(),
					DocCount:  0,
				}
				db.collections.mu.Unlock()
			}
			switch rec.OpType {
			case types.OpCreateCollection, types.OpDeleteCollection:
				continue
			case types.OpCreate:
				offset, err := dataFile.WriteNoSync(rec.Payload)
				if err != nil {
					db.logger.Warn("replay partition %d doc %d: write: %v", partition.ID(), rec.DocID, err)
					continue
				}
				if !db.memory.TryAllocate(db.dbID, uint64(len(rec.Payload))) {
					db.logger.Warn("replay partition %d doc %d: memory limit", partition.ID(), rec.DocID)
					continue
				}
				version := db.mvcc.CreateVersion(rec.DocID, txID, offset, rec.PayloadLen)
				index.Set(collection, version)
				db.collections.IncrementDocCount(collection)
			case types.OpUpdate, types.OpPatch:
				existing, exists := index.Get(collection, rec.DocID, db.mvcc.CurrentSnapshot())
				if !exists {
					offset, err := dataFile.WriteNoSync(rec.Payload)
					if err != nil {
						db.logger.Warn("replay partition %d doc %d: write: %v", partition.ID(), rec.DocID, err)
						continue
					}
					if !db.memory.TryAllocate(db.dbID, uint64(len(rec.Payload))) {
						db.logger.Warn("replay partition %d doc %d: memory limit", partition.ID(), rec.DocID)
						continue
					}
					version := db.mvcc.CreateVersion(rec.DocID, txID, offset, rec.PayloadLen)
					index.Set(collection, version)
					db.collections.IncrementDocCount(collection)
				} else {
					offset, err := dataFile.WriteNoSync(rec.Payload)
					if err != nil {
						db.logger.Warn("replay partition %d doc %d: write: %v", partition.ID(), rec.DocID, err)
						continue
					}
					oldMemory := uint64(existing.Length)
					if !db.memory.TryAllocate(db.dbID, uint64(len(rec.Payload))) {
						db.logger.Warn("replay partition %d doc %d: memory limit", partition.ID(), rec.DocID)
						continue
					}
					if oldMemory > 0 {
						db.memory.Free(db.dbID, oldMemory)
					}
					version := db.mvcc.UpdateVersion(existing, txID, offset, rec.PayloadLen)
					index.Set(collection, version)
				}
			case types.OpDelete:
				existing, exists := index.Get(collection, rec.DocID, db.mvcc.CurrentSnapshot())
				if exists && existing.Length > 0 {
					db.memory.Free(db.dbID, uint64(existing.Length))
				}
				version := db.mvcc.DeleteVersion(rec.DocID, txID)
				index.Set(collection, version)
				db.collections.DecrementDocCount(collection)
			}
		}
	}

	if err := dataFile.Sync(); err != nil {
		db.logger.Warn("partition %d datafile sync after replay: %v", partition.ID(), err)
	}
	return maxTxID, maxLSN, nil
}

// scanPartitionRows scans one partition's index for a collection at the given snapshot
// and returns rows (docID + payload). Lock-free read path.
func (db *LogicalDB) scanPartitionRows(partition *Partition, collection string, snapshotTxID uint64) ([]query.Row, error) {
	if collection == "" {
		collection = DefaultCollection
	}
	idx := partition.GetIndex()
	dataFile := partition.GetDataFile()
	var rows []query.Row
	scanStart := time.Now()
	idx.ScanCollection(collection, snapshotTxID, func(docID uint64, version *types.DocumentVersion) bool {
		data, err := dataFile.Read(version.Offset, version.Length)
		if err != nil {
			db.logger.Warn("query scan partition %d doc %d: read: %v", partition.ID(), docID, err)
			return true
		}
		rows = append(rows, query.Row{DocID: docID, Payload: data})
		return true
	})
	scanDuration := time.Since(scanStart)
	metrics.RecordPartitionIndexScan(db.dbName, strconv.Itoa(partition.ID()), scanDuration)
	return rows, nil
}

// partitionRowStream implements query.RowStream by scanning one partition and yielding rows one at a time.
type partitionRowStream struct {
	ch     chan streamResult
	closed bool
	mu     sync.Mutex
}

type streamResult struct {
	row query.Row
	err error
}

// newPartitionRowStream returns a RowStream that lazily yields rows from the partition.
func (db *LogicalDB) newPartitionRowStream(partition *Partition, collection string, snap uint64) query.RowStream {
	if collection == "" {
		collection = DefaultCollection
	}
	ch := make(chan streamResult, 1)
	stream := &partitionRowStream{ch: ch}
	idx := partition.GetIndex()
	dataFile := partition.GetDataFile()
	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				select {
				case ch <- streamResult{err: fmt.Errorf("scan panic: %v", r)}:
				default:
				}
			}
		}()
		idx.ScanCollection(collection, snap, func(docID uint64, version *types.DocumentVersion) bool {
			data, err := dataFile.Read(version.Offset, version.Length)
			if err != nil {
				ch <- streamResult{err: err}
				return true
			}
			ch <- streamResult{row: query.Row{DocID: docID, Payload: data}}
			return true
		})
		ch <- streamResult{err: io.EOF}
	}()
	return stream
}

func (s *partitionRowStream) Next() (query.Row, error) {
	r, ok := <-s.ch
	if !ok {
		return query.Row{}, io.EOF
	}
	return r.row, r.err
}

func (s *partitionRowStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	for range s.ch {
		// drain so goroutine can exit
	}
	return nil
}

// totalPartitionWALSize returns the sum of all partition WAL sizes (0 if not partitioned).
// Phase D.8: Used to enforce MaxWALSizePerDB.
func (db *LogicalDB) totalPartitionWALSize() uint64 {
	if db.partitions == nil {
		return 0
	}
	var total uint64
	for _, p := range db.partitions {
		if w := p.GetWAL(); w != nil {
			total += w.Size()
		}
	}
	return total
}

// checkWALSizeLimit returns ErrWALSizeLimit if total partition WAL size >= MaxWALSizePerDB.
// No-op if MaxWALSizePerDB is 0 or not partitioned.
func (db *LogicalDB) checkWALSizeLimit() error {
	if db.cfg == nil || db.cfg.Query.MaxWALSizePerDB == 0 || db.partitions == nil {
		return nil
	}
	if db.totalPartitionWALSize() >= db.cfg.Query.MaxWALSizePerDB {
		return ErrWALSizeLimit
	}
	return nil
}

// Default max query memory (100MB) when not set in config. Phase D.8 wires config.
// This is now read from db.cfg.Query.MaxQueryMemoryMB in ExecuteQuery.
const defaultMaxQueryMemoryBytes = 100 * 1024 * 1024

// ExecuteQuery runs a server-side query: snapshot, partition fan-out (or legacy scan), streaming merge, filter/order/limit.
// Context is used for cancellation and timeout (Phase B.3). Memory is capped at defaultMaxQueryMemoryBytes.
func (db *LogicalDB) ExecuteQuery(ctx context.Context, collection string, q query.Query) ([]query.Row, error) {
	// Phase D.8: Acquire query semaphore (concurrent query limiting)
	if db.querySemaphore != nil {
		select {
		case db.querySemaphore <- struct{}{}:
			defer func() { <-db.querySemaphore }()
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, ErrTooManyConcurrentQueries
		}
	}

	// Phase D.8: Apply query timeout if configured
	if db.cfg.Query.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, db.cfg.Query.QueryTimeout)
		defer cancel()
	}

	queryStart := time.Now()
	defer func() {
		executionTime := time.Since(queryStart)
		// Query metrics will be recorded at the end with final counts
		_ = executionTime // Will be used when recording metrics
	}()

	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrDBNotOpen
	}
	if collection == "" {
		collection = DefaultCollection
	}
	if !db.collections.Exists(collection) {
		return nil, errors.ErrCollectionNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}

	snap := db.mvcc.CurrentSnapshot()
	checkSnapshotMonotonic(db.mvcc, snap)
	partitionCount := 0
	if db.partitions != nil {
		partitionCount = len(db.partitions)
	}
	checkQuerySnapshotConsistent(snap, partitionCount)

	// Phase D.8: Use configured query memory limit
	var maxMem uint64 = defaultMaxQueryMemoryBytes
	if db.cfg.Query.MaxQueryMemoryMB > 0 {
		maxMem = uint64(db.cfg.Query.MaxQueryMemoryMB) * 1024 * 1024
	}
	var allRows []query.Row
	var partitionsScanned uint64
	var rowsScanned uint64

	if db.partitions != nil {
		// Partitioned: streaming k-way merge with memory cap
		partitionsScanned = uint64(len(db.partitions))
		streams := make([]query.RowStream, len(db.partitions))
		for i := range db.partitions {
			streams[i] = db.newPartitionRowStream(db.partitions[i], collection, snap)
		}
		merger := query.NewKWayMerger(streams, q.OrderBy, 0) // limit applied when collecting
		defer merger.Close()

		var bytesScanned uint64
		for {
			select {
			case <-ctx.Done():
				executionTime := time.Since(queryStart)
				metrics.RecordQueryMetrics(db.dbName, partitionsScanned, rowsScanned, uint64(len(allRows)), executionTime)
				return nil, ctx.Err()
			default:
			}
			row, ok := merger.Next()
			if !ok {
				break
			}
			rowsScanned++ // Count all rows from merger (before filtering)
			if q.Filter.Field != "" && !db.rowMatchesFilter(row, &q.Filter) {
				continue
			}
			bytesScanned += uint64(len(row.Payload))
			if bytesScanned > maxMem {
				executionTime := time.Since(queryStart)
				metrics.RecordQueryMetrics(db.dbName, partitionsScanned, rowsScanned, uint64(len(allRows)), executionTime)
				return nil, ErrQueryMemoryLimit
			}
			allRows = append(allRows, row)
			if q.Limit > 0 && len(allRows) >= q.Limit {
				break
			}
		}

		// Merger already yielded in sorted order when OrderBy set
		if q.Limit > 0 && len(allRows) > q.Limit {
			allRows = allRows[:q.Limit]
		}
		executionTime := time.Since(queryStart)
		metrics.RecordQueryMetrics(db.dbName, partitionsScanned, rowsScanned, uint64(len(allRows)), executionTime)
		return allRows, nil
	}

	// Legacy: single index + datafile (no streaming)
	partitionsScanned = 1 // Legacy mode = 1 partition
	db.index.ScanCollection(collection, snap, func(docID uint64, version *types.DocumentVersion) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		data, err := db.dataFile.Read(version.Offset, version.Length)
		if err != nil {
			db.logger.Warn("query scan doc %d: read: %v", docID, err)
			return true
		}
		allRows = append(allRows, query.Row{DocID: docID, Payload: data})
		return true
	})
	rowsScanned = uint64(len(allRows)) // In legacy mode, all scanned rows are in allRows before filtering

	// Apply filter (simple JSON field predicate)
	if q.Filter.Field != "" {
		filtered := allRows[:0]
		for _, r := range allRows {
			if db.rowMatchesFilter(r, &q.Filter) {
				filtered = append(filtered, r)
			}
		}
		allRows = filtered
	}

	// Apply order (single field)
	if q.OrderBy != nil && q.OrderBy.Field != "" {
		db.sortRowsByField(allRows, q.OrderBy.Field, q.OrderBy.Asc)
	}

	// Apply limit
	if q.Limit > 0 && len(allRows) > q.Limit {
		allRows = allRows[:q.Limit]
	}

	executionTime := time.Since(queryStart)
	metrics.RecordQueryMetrics(db.dbName, partitionsScanned, rowsScanned, uint64(len(allRows)), executionTime)
	return allRows, nil
}

// rowMatchesFilter returns true if the row's payload JSON matches the expression (eq, neq, etc.).
func (db *LogicalDB) rowMatchesFilter(row query.Row, expr *query.Expression) bool {
	var doc map[string]interface{}
	if err := json.Unmarshal(row.Payload, &doc); err != nil {
		return false
	}
	val, ok := doc[expr.Field]
	if !ok {
		return expr.Op == "neq"
	}
	cmp := compareValues(val, expr.Value)
	switch expr.Op {
	case "eq":
		return cmp == 0
	case "neq":
		return cmp != 0
	case "gt":
		return cmp > 0
	case "gte":
		return cmp >= 0
	case "lt":
		return cmp < 0
	case "lte":
		return cmp <= 0
	default:
		return false
	}
}

func compareValues(a, b interface{}) int {
	// Simple comparison for numbers and strings
	fa, oka := toFloat(a)
	fb, okb := toFloat(b)
	if oka && okb {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}
	sa, oka := toString(a)
	sb, okb := toString(b)
	if oka && okb {
		if sa < sb {
			return -1
		}
		if sa > sb {
			return 1
		}
		return 0
	}
	return 0
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

func toString(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

// sortRowsByField sorts rows in place by a JSON field (string or number).
func (db *LogicalDB) sortRowsByField(rows []query.Row, field string, asc bool) {
	// Extract sort key per row and sort
	type kv struct {
		key interface{}
		row query.Row
	}
	var kvs []kv
	for _, r := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(r.Payload, &doc); err != nil {
			kvs = append(kvs, kv{key: nil, row: r})
			continue
		}
		kvs = append(kvs, kv{key: doc[field], row: r})
	}
	sort.Slice(kvs, func(i, j int) bool {
		ci := compareValues(kvs[i].key, kvs[j].key)
		if asc {
			return ci < 0
		}
		return ci > 0
	})
	for i := range rows {
		rows[i] = kvs[i].row
	}
}

// Patch applies path-based updates to a document atomically.
func (db *LogicalDB) Patch(collection string, docID uint64, ops []types.PatchOperation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	// Normalize collection name
	if collection == "" {
		collection = DefaultCollection
	}

	// Validate collection exists
	if !db.collections.Exists(collection) {
		return errors.ErrCollectionNotFound
	}

	if len(ops) == 0 {
		return errors.ErrInvalidPatch
	}

	// Read current document
	version, exists := db.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return ErrDocNotFound
	}

	data, err := db.dataFile.Read(version.Offset, version.Length)
	if err != nil {
		return fmt.Errorf("failed to read document: %w", err)
	}

	// Parse current document as JSON
	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return errors.ErrInvalidJSON
	}

	// Ensure document is a JSON object
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return errors.ErrNotJSONObject
	}

	// Apply each patch operation
	for _, op := range ops {
		path, err := ParsePath(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path '%s': %w", op.Path, err)
		}

		switch op.Op {
		case "set":
			if op.Value == nil {
				return fmt.Errorf("set operation requires value: %w", errors.ErrInvalidPatch)
			}
			if err := SetValue(docMap, path, op.Value); err != nil {
				return fmt.Errorf("failed to set value at path '%s': %w", op.Path, err)
			}

		case "delete":
			if err := DeleteValue(docMap, path); err != nil {
				return fmt.Errorf("failed to delete value at path '%s': %w", op.Path, err)
			}

		case "insert":
			if op.Value == nil {
				return fmt.Errorf("insert operation requires value: %w", errors.ErrInvalidPatch)
			}
			if len(path) == 0 {
				return fmt.Errorf("insert operation requires array path: %w", errors.ErrInvalidPatch)
			}
			// Last segment should be array index
			indexStr := path[len(path)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return fmt.Errorf("insert operation requires numeric index: %w", errors.ErrInvalidPatch)
			}
			parentPath := path[:len(path)-1]
			if err := InsertValue(docMap, parentPath, index, op.Value); err != nil {
				return fmt.Errorf("failed to insert value at path '%s': %w", op.Path, err)
			}

		default:
			return fmt.Errorf("unknown patch operation '%s': %w", op.Op, errors.ErrInvalidPatch)
		}
	}

	// Marshal updated document
	updatedPayload, err := json.Marshal(docMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated document: %w", err)
	}

	// Validate updated payload
	if err := validateJSONPayload(updatedPayload); err != nil {
		return fmt.Errorf("updated document is invalid JSON: %w", err)
	}

	// Calculate memory change
	memoryNeeded := uint64(len(updatedPayload))
	oldMemory := uint64(version.Length)

	if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
		return ErrMemoryLimit
	}

	// Write updated payload to data file
	offset, err := db.dataFile.Write(updatedPayload)
	if err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return err
	}

	txID := db.mvcc.NextTxID()
	newVersion := db.mvcc.UpdateVersion(version, txID, offset, uint32(len(updatedPayload)))

	// Phase 1: write WAL record for the patch operation.
	if err := db.wal.Write(txID, db.dbID, collection, docID, types.OpPatch, updatedPayload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(collection, newVersion)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}

	// Check if checkpoint should be created after commit
	db.maybeCreateCheckpointAndTrim(txID)

	return nil
}

// CreateCollection creates a new collection.
func (db *LogicalDB) CreateCollection(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	if err := db.collections.CreateCollection(name); err != nil {
		return err
	}

	// Write WAL record for collection creation
	txID := db.mvcc.NextTxID()
	if err := db.wal.Write(txID, db.dbID, name, 0, types.OpCreateCollection, nil); err != nil {
		// Rollback collection creation
		db.collections.mu.Lock()
		delete(db.collections.collections, name)
		db.collections.mu.Unlock()
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	if err := db.wal.WriteCommitMarker(txID); err != nil {
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	return nil
}

// DeleteCollection deletes an empty collection.
func (db *LogicalDB) DeleteCollection(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	if err := db.collections.DeleteCollection(name); err != nil {
		return err
	}

	// Write WAL record for collection deletion
	txID := db.mvcc.NextTxID()
	if err := db.wal.Write(txID, db.dbID, name, 0, types.OpDeleteCollection, nil); err != nil {
		// Rollback - recreate collection
		db.collections.mu.Lock()
		db.collections.collections[name] = &types.CollectionMetadata{
			Name:      name,
			CreatedAt: time.Now(),
			DocCount:  0,
		}
		db.collections.mu.Unlock()
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	if err := db.wal.WriteCommitMarker(txID); err != nil {
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	return nil
}

// ListCollections returns all collection names.
func (db *LogicalDB) ListCollections() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil
	}

	return db.collections.ListCollections()
}

// maybeCreateCheckpointAndTrim creates a checkpoint if needed and trims old WAL segments.
func (db *LogicalDB) maybeCreateCheckpointAndTrim(txID uint64) {
	if db.checkpointMgr == nil || !db.checkpointMgr.ShouldCreateCheckpoint(db.wal.Size()) {
		return
	}

	if err := db.wal.WriteCheckpoint(txID); err != nil {
		db.logger.Warn("Failed to write checkpoint: %v", err)
		// Don't fail the operation if checkpoint fails
		return
	}

	// Trim old WAL segments after checkpoint
	if db.walTrimmer != nil && db.cfg.WAL.TrimAfterCheckpoint {
		if err := db.walTrimmer.TrimSegmentsBeforeCheckpoint(txID, db.cfg.WAL.KeepSegments); err != nil {
			db.logger.Warn("Failed to trim WAL segments: %v", err)
			// Don't fail the operation if trimming fails
		}
	}
}

func (db *LogicalDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	// Stop worker pool (v0.4)
	if db.workerPool != nil {
		db.workerPool.Stop()
		db.workerPool = nil
	}

	// Stop healing service
	if db.healingService != nil {
		db.healingService.Stop()
		db.healingService = nil
	}

	// Close partitions (v0.4)
	if db.partitions != nil {
		for _, partition := range db.partitions {
			if w := partition.GetWAL(); w != nil {
				w.Close()
			}
			if df := partition.GetDataFile(); df != nil {
				df.Close()
			}
		}
		db.partitions = nil
	}

	// Close legacy WAL and datafile (PartitionCount=1)
	if db.wal != nil {
		db.wal.Close()
		db.wal = nil
	}

	if db.dataFile != nil {
		db.dataFile.Close()
		db.dataFile = nil
	}

	db.closed = true
	db.logger.Info("Closed database: %s (id=%d)", db.dbName, db.dbID)
	return nil
}

func (db *LogicalDB) Name() string {
	return db.dbName
}

func (db *LogicalDB) Create(collection string, docID uint64, payload []byte) error {
	// Validate JSON-only payload outside lock to reduce contention
	if err := validateJSONPayload(payload); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	// Normalize collection name
	if collection == "" {
		collection = DefaultCollection
	}

	// Validate collection exists (uses collections.mu internally)
	if !db.collections.Exists(collection) {
		return errors.ErrCollectionNotFound
	}

	version, exists := db.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if exists && version.DeletedTxID == nil {
		return ErrDocAlreadyExists
	}

	memoryNeeded := uint64(len(payload))
	if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
		return ErrMemoryLimit
	}

	offset, err := db.dataFile.Write(payload)
	if err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return err
	}

	txID := db.mvcc.NextTxID()
	newVersion := db.mvcc.CreateVersion(docID, txID, offset, uint32(len(payload)))

	// Phase 1: write WAL record for the operation.
	if err := db.wal.Write(txID, db.dbID, collection, docID, types.OpCreate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the new version visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(collection, newVersion)
	db.collections.IncrementDocCount(collection)

	// Check if checkpoint should be created after commit
	db.maybeCreateCheckpointAndTrim(txID)

	return nil
}

func (db *LogicalDB) Read(collection string, docID uint64) ([]byte, error) {
	db.mu.RLock()

	if db.closed {
		db.mu.RUnlock()
		return nil, ErrDBNotOpen
	}

	// Normalize collection name
	if collection == "" {
		collection = DefaultCollection
	}

	version, exists := db.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		db.mu.RUnlock()
		return nil, ErrDocNotFound
	}

	data, err := db.dataFile.Read(version.Offset, version.Length)
	db.mu.RUnlock()

	// Trigger healing on corruption detection
	if err != nil && db.healingService != nil {
		db.healingService.HealOnCorruption(collection, docID)
	}

	return data, err
}

func (db *LogicalDB) Update(collection string, docID uint64, payload []byte) error {
	// Normalize collection name
	if collection == "" {
		collection = DefaultCollection
	}

	// Validate JSON-only payload outside lock to reduce contention
	if err := validateJSONPayload(payload); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	// Validate collection exists (uses collections.mu internally)
	if !db.collections.Exists(collection) {
		return errors.ErrCollectionNotFound
	}

	version, exists := db.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return ErrDocNotFound
	}

	memoryNeeded := uint64(len(payload))
	oldMemory := uint64(version.Length)

	if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
		return ErrMemoryLimit
	}

	offset, err := db.dataFile.Write(payload)
	if err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return err
	}

	txID := db.mvcc.NextTxID()
	newVersion := db.mvcc.UpdateVersion(version, txID, offset, uint32(len(payload)))

	// Phase 1: write WAL record for the update.
	if err := db.wal.Write(txID, db.dbID, collection, docID, types.OpUpdate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the updated version visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(collection, newVersion)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}

	// Check if checkpoint should be created after commit
	db.maybeCreateCheckpointAndTrim(txID)

	return nil
}

func (db *LogicalDB) Delete(collection string, docID uint64) error {
	// Normalize collection name
	if collection == "" {
		collection = DefaultCollection
	}

	// Validate collection exists (uses collections.mu internally)
	if !db.collections.Exists(collection) {
		return errors.ErrCollectionNotFound
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	version, exists := db.index.Get(collection, docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return ErrDocNotFound
	}

	oldMemory := uint64(version.Length)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}

	txID := db.mvcc.NextTxID()
	deleteVersion := db.mvcc.DeleteVersion(docID, txID)

	// Phase 1: write WAL record for the delete.
	if err := db.wal.Write(txID, db.dbID, collection, docID, types.OpDelete, nil); err != nil {
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the delete visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		classifier := errors.NewClassifier()
		category := classifier.Classify(err)
		db.errorTracker.RecordError(err, category)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(collection, deleteVersion)
	db.collections.DecrementDocCount(collection)

	// Check if checkpoint should be created after commit
	db.maybeCreateCheckpointAndTrim(txID)

	return nil
}

func (db *LogicalDB) IndexSize() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.index.Size()
}

func (db *LogicalDB) MemoryUsage() uint64 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.memory.DBUsage(db.dbID)
}

func (db *LogicalDB) Begin() *Tx {
	return db.txManager.Begin()
}

func (db *LogicalDB) Commit(tx *Tx) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	records, err := db.txManager.Commit(tx)
	if err != nil {
		return err
	}

	// Phase 1: write all WAL records for this transaction.
	for _, record := range records {
		collection := record.Collection
		if collection == "" {
			collection = DefaultCollection
		}
		if err := db.wal.Write(record.TxID, record.DBID, collection, record.DocID, record.OpType, record.Payload); err != nil {
			return fmt.Errorf("failed to write WAL: %w", err)
		}
	}

	// Phase 2: write the transaction commit marker. Only after this
	// succeeds do we make the transaction's changes visible in the index.
	if err := db.wal.WriteCommitMarker(tx.ID); err != nil {
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++

	// Apply changes to MVCC/index using the transaction's ID so that
	// visibility matches the WAL records.
	for _, record := range records {
		collection := record.Collection
		if collection == "" {
			collection = DefaultCollection
		}

		switch record.OpType {
		case types.OpCreate:
			version := db.mvcc.CreateVersion(record.DocID, tx.ID, 0, record.PayloadLen)
			db.index.Set(collection, version)
			db.collections.IncrementDocCount(collection)
		case types.OpUpdate, types.OpPatch:
			existing, exists := db.index.Get(collection, record.DocID, db.mvcc.CurrentSnapshot())
			if exists {
				version := db.mvcc.UpdateVersion(existing, tx.ID, 0, record.PayloadLen)
				db.index.Set(collection, version)
			}
		case types.OpDelete:
			version := db.mvcc.DeleteVersion(record.DocID, tx.ID)
			db.index.Set(collection, version)
			db.collections.DecrementDocCount(collection)
		}
	}

	// Check if checkpoint should be created after transaction commit
	db.maybeCreateCheckpointAndTrim(tx.ID)

	return nil
}

func (db *LogicalDB) Rollback(tx *Tx) error {
	return db.txManager.Rollback(tx)
}

func (db *LogicalDB) Stats() *types.Stats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	live := db.index.TotalLiveCount()
	tombstoned := db.index.TotalTombstonedCount()

	return &types.Stats{
		TotalDBs:       1,
		ActiveDBs:      1,
		TotalTxns:      db.mvcc.CurrentSnapshot(),
		TxnsCommitted:  db.txnsCommitted,
		WALSize:        db.wal.Size(),
		MemoryUsed:     db.memory.DBUsage(db.dbID),
		MemoryCapacity: db.memory.DBLimit(db.dbID),
		DocsLive:       uint64(live),
		DocsTombstoned: uint64(tombstoned),
		LastCompaction: db.lastCompaction,
	}
}

// HealingService returns the healing service for this database.
func (db *LogicalDB) HealingService() *HealingService {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.healingService
}

// GetWALStats returns WAL/group-commit metrics when group commit is active.
// Returns nil when group commit is not in use. Map keys: total_batches, total_records,
// avg_batch_size, avg_batch_latency_ns, max_batch_size, max_batch_latency_ns, last_flush_time_unix_ns.
func (db *LogicalDB) GetWALStats() map[string]interface{} {
	db.mu.RLock()
	wal := db.wal
	db.mu.RUnlock()
	if wal == nil {
		return nil
	}
	st, ok := wal.GetGroupCommitStats()
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"total_batches":           st.TotalBatches,
		"total_records":           st.TotalRecords,
		"avg_batch_size":          st.AvgBatchSize,
		"avg_batch_latency_ns":    st.AvgBatchLatency.Nanoseconds(),
		"max_batch_size":          st.MaxBatchSize,
		"max_batch_latency_ns":    st.MaxBatchLatency.Nanoseconds(),
		"last_flush_time_unix_ns": st.LastFlushTime.UnixNano(),
	}
}

func (db *LogicalDB) replayWAL() error {
	walBasePath := filepath.Join(db.walDir, fmt.Sprintf("%s.wal", db.dbName))
	if _, err := os.Stat(walBasePath); os.IsNotExist(err) {
		return nil
	}

	recovery := wal.NewRecovery(walBasePath, db.logger)

	maxTxID := uint64(0)
	records := 0
	lastCheckpointTxID := uint64(0)

	txRecords := make(map[uint64][]*types.WALRecord)
	committed := make(map[uint64]bool)

	// Single pass: find checkpoint and buffer records (avoids second full WAL read).
	err := recovery.Replay(func(rec *types.WALRecord) error {
		if rec == nil {
			return nil
		}

		txID := rec.TxID

		if rec.OpType == types.OpCheckpoint {
			if txID > lastCheckpointTxID {
				lastCheckpointTxID = txID
			}
			return nil
		}

		if txID > maxTxID {
			maxTxID = txID
		}

		switch rec.OpType {
		case types.OpCommit:
			committed[txID] = true
		default:
			txRecords[txID] = append(txRecords[txID], rec)
			records++
		}

		return nil
	})

	// Apply buffered records even when Replay returned error (e.g. corrupt record).
	// We truncate at corruption and have already buffered all valid records before the error.
	if err != nil {
		db.logger.Warn("Failed to replay WAL: %v", err)
	}

	if lastCheckpointTxID > 0 {
		db.logger.Info("Found checkpoint at tx_id=%d, recovery will start from there", lastCheckpointTxID)
	}

	// Update MVCC to use the next transaction ID after the highest one found
	if maxTxID > 0 {
		db.mvcc.SetCurrentTxID(maxTxID + 1)
	}

	// Apply only records from committed transactions, in txID order,
	// so Create-before-Delete ordering is preserved (map iteration is non-deterministic).
	txIDs := make([]uint64, 0, len(txRecords))
	for txID := range txRecords {
		if !committed[txID] {
			continue
		}
		if lastCheckpointTxID > 0 && txID <= lastCheckpointTxID {
			continue
		}
		txIDs = append(txIDs, txID)
	}
	sort.Slice(txIDs, func(i, j int) bool { return txIDs[i] < txIDs[j] })
	for _, txID := range txIDs {
		recs := txRecords[txID]
		for _, rec := range recs {
			collection := rec.Collection
			if collection == "" {
				collection = DefaultCollection
			}

			// Ensure collection exists (create if needed during recovery)
			if !db.collections.Exists(collection) {
				db.collections.mu.Lock()
				db.collections.collections[collection] = &types.CollectionMetadata{
					Name:      collection,
					CreatedAt: time.Now(),
					DocCount:  0,
				}
				db.collections.mu.Unlock()
			}

			switch rec.OpType {
			case types.OpCreateCollection:
				// Collection creation - already handled above
				continue

			case types.OpDeleteCollection:
				// Collection deletion - skip during recovery (collections recreated from WAL)
				continue

			case types.OpCreate:
				// WriteNoSync during replay; one Sync at end of replay (avoids N fsyncs).
				offset, err := db.dataFile.WriteNoSync(rec.Payload)
				if err != nil {
					db.logger.Warn("Failed to write payload to data file during replay for doc %d: %v", rec.DocID, err)
					continue
				}

				// Allocate memory for the payload
				memoryNeeded := uint64(len(rec.Payload))
				if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
					db.logger.Warn("Memory limit reached during WAL replay for doc %d", rec.DocID)
					continue
				}

				// Create version with correct offset
				version := db.mvcc.CreateVersion(rec.DocID, txID, offset, rec.PayloadLen)
				db.index.Set(collection, version)
				db.collections.IncrementDocCount(collection)

			case types.OpUpdate, types.OpPatch:
				existing, exists := db.index.Get(collection, rec.DocID, db.mvcc.CurrentSnapshot())
				if !exists {
					// If document doesn't exist, treat as create
					offset, err := db.dataFile.WriteNoSync(rec.Payload)
					if err != nil {
						db.logger.Warn("Failed to write payload to data file during replay for doc %d: %v", rec.DocID, err)
						continue
					}

					memoryNeeded := uint64(len(rec.Payload))
					if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
						db.logger.Warn("Memory limit reached during WAL replay for doc %d", rec.DocID)
						continue
					}

					version := db.mvcc.CreateVersion(rec.DocID, txID, offset, rec.PayloadLen)
					db.index.Set(collection, version)
					db.collections.IncrementDocCount(collection)
				} else {
					// Write new payload to data file (no sync during replay)
					offset, err := db.dataFile.WriteNoSync(rec.Payload)
					if err != nil {
						db.logger.Warn("Failed to write payload to data file during replay for doc %d: %v", rec.DocID, err)
						continue
					}

					// Calculate memory change
					oldMemory := uint64(existing.Length)
					memoryNeeded := uint64(len(rec.Payload))

					// Try to allocate new memory
					if !db.memory.TryAllocate(db.dbID, memoryNeeded) {
						db.logger.Warn("Memory limit reached during WAL replay for doc %d", rec.DocID)
						continue
					}

					// Free old memory
					if oldMemory > 0 {
						db.memory.Free(db.dbID, oldMemory)
					}

					// Update version with correct offset
					version := db.mvcc.UpdateVersion(existing, txID, offset, rec.PayloadLen)
					db.index.Set(collection, version)
				}

			case types.OpDelete:
				existing, exists := db.index.Get(collection, rec.DocID, db.mvcc.CurrentSnapshot())
				if exists {
					// Free memory for deleted document
					if existing.Length > 0 {
						db.memory.Free(db.dbID, uint64(existing.Length))
					}
				}
				version := db.mvcc.DeleteVersion(rec.DocID, txID)
				db.index.Set(collection, version)
				db.collections.DecrementDocCount(collection)
			}
		}
	}

	// Single fsync after all replay writes (was N fsyncs during apply).
	if syncErr := db.dataFile.Sync(); syncErr != nil {
		db.logger.Warn("Data file sync after replay failed: %v", syncErr)
	}

	db.logger.Info("Replayed %d WAL records (committed only)", records)
	// Return replay error so caller can log; state is still consistent (we applied good records).
	return err
}
