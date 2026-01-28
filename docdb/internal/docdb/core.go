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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/internal/wal"
)

// LogicalDB represents a single logical database instance.
//
// It manages the complete lifecycle of documents within this database,
// including storage, indexing, transaction management, and recovery.
//
// Thread Safety: All public methods are safe for concurrent use.
// The mu (RWMutex) protects all internal state.
type LogicalDB struct {
	mu             sync.RWMutex           // Protects all internal state
	dbID           uint64                 // Unique database identifier
	dbName         string                 // Human-readable database name
	dataFile       *DataFile              // Append-only file for document payloads
	wal            *wal.Writer            // Write-ahead log for durability
	index          *Index                 // Sharded in-memory index
	mvcc           *MVCC                  // Multi-version concurrency control
	txManager      *TransactionManager    // Transaction lifecycle management
	memory         *memory.Caps           // Memory limit tracking
	pool           *memory.BufferPool     // Efficient buffer allocation
	cfg            *config.Config         // Database configuration
	logger         *logger.Logger         // Structured logging
	closed         bool                   // True if database is closed
	dataDir        string                 // Directory for data files
	walDir         string                 // Directory for WAL files
	txnsCommitted  uint64                 // Count of committed transactions
	lastCompaction time.Time              // Timestamp of last compaction
	checkpointMgr  *wal.CheckpointManager // Checkpoint management for bounded recovery
}

func NewLogicalDB(dbID uint64, dbName string, cfg *config.Config, memCaps *memory.Caps, pool *memory.BufferPool, log *logger.Logger) *LogicalDB {
	return &LogicalDB{
		dbID:      dbID,
		dbName:    dbName,
		index:     NewIndex(),
		mvcc:      NewMVCC(),
		txManager: NewTransactionManager(NewMVCC()),
		memory:    memCaps,
		pool:      pool,
		cfg:       cfg,
		logger:    log,
		closed:    false,
	}
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

	dbFile := filepath.Join(dataDir, fmt.Sprintf("%s.data", db.dbName))
	db.dataFile = NewDataFile(dbFile, db.logger)
	if err := db.dataFile.Open(); err != nil {
		return fmt.Errorf("failed to open data file: %w", err)
	}

	walFile := filepath.Join(walDir, fmt.Sprintf("%s.wal", db.dbName))
	db.wal = wal.NewWriter(walFile, db.dbID, db.cfg.WAL.MaxFileSizeMB*1024*1024, db.cfg.WAL.FsyncOnCommit, db.logger)
	if err := db.wal.Open(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}

	// Initialize checkpoint manager
	db.checkpointMgr = wal.NewCheckpointManager(
		db.cfg.WAL.Checkpoint.IntervalMB,
		db.cfg.WAL.Checkpoint.AutoCreate,
		db.cfg.WAL.Checkpoint.MaxCheckpoints,
		db.logger,
	)
	db.wal.SetCheckpointManager(db.checkpointMgr)

	if err := db.replayWAL(); err != nil {
		db.logger.Warn("Failed to replay WAL: %v", err)
	}

	db.logger.Info("Opened database: %s (id=%d)", db.dbName, db.dbID)
	return nil
}

func (db *LogicalDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	if db.wal != nil {
		db.wal.Close()
	}

	if db.dataFile != nil {
		db.dataFile.Close()
	}

	db.closed = true
	db.logger.Info("Closed database: %s (id=%d)", db.dbName, db.dbID)
	return nil
}

func (db *LogicalDB) Name() string {
	return db.dbName
}

func (db *LogicalDB) Create(docID uint64, payload []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	// Enforce JSON-only payloads at the engine level before any WAL or
	// data file writes, so invalid inputs cannot corrupt durable state.
	if err := validateJSONPayload(payload); err != nil {
		return err
	}

	version, exists := db.index.Get(docID, db.mvcc.CurrentSnapshot())
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
	if err := db.wal.Write(txID, db.dbID, docID, types.OpCreate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the new version visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(newVersion)

	// Check if checkpoint should be created after commit
	if db.checkpointMgr != nil && db.checkpointMgr.ShouldCreateCheckpoint(db.wal.Size()) {
		if err := db.wal.WriteCheckpoint(txID); err != nil {
			db.logger.Warn("Failed to write checkpoint: %v", err)
			// Don't fail the operation if checkpoint fails
		}
	}

	return nil
}

func (db *LogicalDB) Read(docID uint64) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrDBNotOpen
	}

	version, exists := db.index.Get(docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return nil, ErrDocNotFound
	}

	return db.dataFile.Read(version.Offset, version.Length)
}

func (db *LogicalDB) Update(docID uint64, payload []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	// Enforce JSON-only payloads at the engine level before any WAL or
	// data file writes, so invalid inputs cannot corrupt durable state.
	if err := validateJSONPayload(payload); err != nil {
		return err
	}

	version, exists := db.index.Get(docID, db.mvcc.CurrentSnapshot())
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
	if err := db.wal.Write(txID, db.dbID, docID, types.OpUpdate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the updated version visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(newVersion)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
	}

	// Check if checkpoint should be created after commit
	if db.checkpointMgr != nil && db.checkpointMgr.ShouldCreateCheckpoint(db.wal.Size()) {
		if err := db.wal.WriteCheckpoint(txID); err != nil {
			db.logger.Warn("Failed to write checkpoint: %v", err)
			// Don't fail the operation if checkpoint fails
		}
	}

	return nil
}

func (db *LogicalDB) Delete(docID uint64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrDBNotOpen
	}

	version, exists := db.index.Get(docID, db.mvcc.CurrentSnapshot())
	if !exists || version.DeletedTxID != nil {
		return ErrDocNotFound
	}

	txID := db.mvcc.NextTxID()
	deleteVersion := db.mvcc.DeleteVersion(docID, txID)

	// Phase 1: write WAL record for the delete.
	if err := db.wal.Write(txID, db.dbID, docID, types.OpDelete, nil); err != nil {
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	// Phase 2: write commit marker. Only after this succeeds do we
	// make the delete visible in the index.
	if err := db.wal.WriteCommitMarker(txID); err != nil {
		return fmt.Errorf("failed to write WAL commit marker: %w", err)
	}

	db.txnsCommitted++
	db.index.Set(deleteVersion)
	if version.Length > 0 {
		db.memory.Free(db.dbID, uint64(version.Length))
	}

	// Check if checkpoint should be created after commit
	if db.checkpointMgr != nil && db.checkpointMgr.ShouldCreateCheckpoint(db.wal.Size()) {
		if err := db.wal.WriteCheckpoint(txID); err != nil {
			db.logger.Warn("Failed to write checkpoint: %v", err)
			// Don't fail the operation if checkpoint fails
		}
	}

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
		if err := db.wal.Write(record.TxID, record.DBID, record.DocID, record.OpType, record.Payload); err != nil {
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
		switch record.OpType {
		case types.OpCreate:
			version := db.mvcc.CreateVersion(record.DocID, tx.ID, 0, record.PayloadLen)
			db.index.Set(version)
		case types.OpUpdate:
			existing, exists := db.index.Get(record.DocID, db.mvcc.CurrentSnapshot())
			if exists {
				version := db.mvcc.UpdateVersion(existing, tx.ID, 0, record.PayloadLen)
				db.index.Set(version)
			}
		case types.OpDelete:
			version := db.mvcc.DeleteVersion(record.DocID, tx.ID)
			db.index.Set(version)
		}
	}

	// Check if checkpoint should be created after transaction commit
	if db.checkpointMgr != nil && db.checkpointMgr.ShouldCreateCheckpoint(db.wal.Size()) {
		if err := db.wal.WriteCheckpoint(tx.ID); err != nil {
			db.logger.Warn("Failed to write checkpoint: %v", err)
			// Don't fail the operation if checkpoint fails
		}
	}

	return nil
}

func (db *LogicalDB) Rollback(tx *Tx) error {
	return db.txManager.Rollback(tx)
}

func (db *LogicalDB) Stats() *types.Stats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	live := db.index.LiveCount()
	tombstoned := db.index.TombstonedCount()

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

func (db *LogicalDB) replayWAL() error {
	walBasePath := filepath.Join(db.walDir, fmt.Sprintf("%s.wal", db.dbName))
	if _, err := os.Stat(walBasePath); os.IsNotExist(err) {
		return nil
	}

	recovery := wal.NewRecovery(walBasePath, db.logger)

	maxTxID := uint64(0)
	records := 0

	// Buffer WAL records per transaction so that we can apply only
	// those belonging to transactions that have a corresponding
	// OpCommit marker.
	txRecords := make(map[uint64][]*types.WALRecord)
	committed := make(map[uint64]bool)

	// First, find the last checkpoint by scanning the WAL
	lastCheckpointTxID := uint64(0)
	checkpointRecovery := wal.NewRecovery(walBasePath, db.logger)
	checkpointRecovery.Replay(func(rec *types.WALRecord) error {
		if rec != nil && rec.OpType == types.OpCheckpoint {
			if rec.TxID > lastCheckpointTxID {
				lastCheckpointTxID = rec.TxID
			}
		}
		return nil
	})

	if lastCheckpointTxID > 0 {
		db.logger.Info("Found checkpoint at tx_id=%d, recovery will start from there", lastCheckpointTxID)
	}

	// Now replay from checkpoint (or beginning), buffering committed transactions
	err := recovery.Replay(func(rec *types.WALRecord) error {
		if rec == nil {
			return nil
		}

		txID := rec.TxID

		// Skip records before checkpoint (they're already applied at checkpoint time)
		if lastCheckpointTxID > 0 && txID < lastCheckpointTxID {
			return nil
		}

		if txID > maxTxID {
			maxTxID = txID
		}

		switch rec.OpType {
		case types.OpCheckpoint:
			// Checkpoints are metadata, already processed in first pass
		case types.OpCommit:
			// Mark transaction as committed; its buffered records will
			// be applied after replay completes.
			committed[txID] = true
		default:
			// Buffer data-bearing records by transaction.
			txRecords[txID] = append(txRecords[txID], rec)
			records++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("corrupt WAL: %w", err)
	}

	// Update MVCC to use the next transaction ID after the highest one found
	if maxTxID > 0 {
		db.mvcc.SetCurrentTxID(maxTxID + 1)
	}

	// Apply only records from committed transactions, rebuilding the
	// data file and index state from durable WAL.
	for txID, recs := range txRecords {
		if !committed[txID] {
			// Skip uncommitted transactions: their effects should not
			// be visible after recovery.
			continue
		}

		for _, rec := range recs {
			switch rec.OpType {
			case types.OpCreate:
				// Write payload to data file to get the actual offset
				offset, err := db.dataFile.Write(rec.Payload)
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
				db.index.Set(version)

			case types.OpUpdate:
				existing, exists := db.index.Get(rec.DocID, db.mvcc.CurrentSnapshot())
				if !exists {
					// If document doesn't exist, treat as create
					offset, err := db.dataFile.Write(rec.Payload)
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
					db.index.Set(version)
				} else {
					// Write new payload to data file
					offset, err := db.dataFile.Write(rec.Payload)
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
					db.index.Set(version)
				}

			case types.OpDelete:
				existing, exists := db.index.Get(rec.DocID, db.mvcc.CurrentSnapshot())
				if exists {
					// Free memory for deleted document
					if existing.Length > 0 {
						db.memory.Free(db.dbID, uint64(existing.Length))
					}
				}
				version := db.mvcc.DeleteVersion(rec.DocID, txID)
				db.index.Set(version)
			}
		}
	}

	db.logger.Info("Replayed %d WAL records (committed only)", records)
	return nil
}
