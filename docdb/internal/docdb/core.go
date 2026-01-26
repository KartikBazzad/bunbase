package docdb

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/internal/wal"
)

type LogicalDB struct {
	mu        sync.RWMutex
	dbID      uint64
	dbName    string
	dataFile  *DataFile
	wal       *wal.Writer
	index     *Index
	mvcc      *MVCC
	txManager *TransactionManager
	memory    *memory.Caps
	pool      *memory.BufferPool
	cfg       *config.Config
	logger    *logger.Logger
	closed    bool
	dataDir   string
	walDir    string
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
	db.wal = wal.NewWriter(walFile, db.cfg.WAL.MaxFileSizeMB*1024*1024, db.cfg.WAL.FsyncOnCommit, db.logger)
	if err := db.wal.Open(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
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

	if err := db.wal.Write(txID, db.dbID, docID, types.OpCreate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	db.index.Set(newVersion)
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

	if err := db.wal.Write(txID, db.dbID, docID, types.OpUpdate, payload); err != nil {
		db.memory.Free(db.dbID, memoryNeeded)
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	db.index.Set(newVersion)
	if oldMemory > 0 {
		db.memory.Free(db.dbID, oldMemory)
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

	if err := db.wal.Write(txID, db.dbID, docID, types.OpDelete, nil); err != nil {
		return fmt.Errorf("failed to write WAL: %w", err)
	}

	db.index.Set(deleteVersion)
	if version.Length > 0 {
		db.memory.Free(db.dbID, uint64(version.Length))
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

	for _, record := range records {
		if err := db.wal.Write(record.TxID, record.DBID, record.DocID, record.OpType, record.Payload); err != nil {
			return fmt.Errorf("failed to write WAL: %w", err)
		}
	}

	txID := db.mvcc.NextTxID()
	for _, record := range records {
		switch record.OpType {
		case types.OpCreate:
			version := db.mvcc.CreateVersion(record.DocID, txID, 0, record.PayloadLen)
			db.index.Set(version)
		case types.OpUpdate:
			existing, exists := db.index.Get(record.DocID, db.mvcc.CurrentSnapshot())
			if exists {
				version := db.mvcc.UpdateVersion(existing, txID, 0, record.PayloadLen)
				db.index.Set(version)
			}
		case types.OpDelete:
			version := db.mvcc.DeleteVersion(record.DocID, txID)
			db.index.Set(version)
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

	return &types.Stats{
		TotalDBs:       1,
		ActiveDBs:      1,
		TotalTxns:      db.mvcc.CurrentSnapshot(),
		WALSize:        db.wal.Size(),
		MemoryUsed:     db.memory.DBUsage(db.dbID),
		MemoryCapacity: db.memory.DBLimit(db.dbID),
	}
}

func (db *LogicalDB) replayWAL() error {
	walFile := filepath.Join(db.walDir, fmt.Sprintf("%s.wal", db.dbName))
	if _, err := os.Stat(walFile); os.IsNotExist(err) {
		return nil
	}

	reader := wal.NewReader(walFile, db.logger)
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()

	var records []*types.WALRecord
	for {
		record, err := reader.Next()
		if err != nil {
			if err == wal.ErrCorruptRecord || err == wal.ErrFileRead {
				return fmt.Errorf("corrupt WAL: %w", err)
			}
			return err
		}
		if record == nil {
			break
		}
		records = append(records, record)
	}

	for _, rec := range records {
		txID := rec.TxID

		switch rec.OpType {
		case types.OpCreate:
			version := db.mvcc.CreateVersion(rec.DocID, txID, 0, rec.PayloadLen)
			db.index.Set(version)
		case types.OpUpdate:
			existing, exists := db.index.Get(rec.DocID, db.mvcc.CurrentSnapshot())
			if exists {
				version := db.mvcc.UpdateVersion(existing, txID, 0, rec.PayloadLen)
				db.index.Set(version)
			}
		case types.OpDelete:
			version := db.mvcc.DeleteVersion(rec.DocID, txID)
			db.index.Set(version)
		}
	}

	db.logger.Info("Replayed %d WAL records", len(records))
	return nil
}
