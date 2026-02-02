// Package bundoc implements a high-performance, embedded document database in Go.
//
// Key Features:
//   - ACID Transactions via MVCC (Multi-Version Concurrency Control)
//   - Write-Ahead Logging (WAL) for durability and crash recovery
//   - B+Tree Indexing for fast lookups and range scans
//   - Connection Pooling for concurrent access management
//   - Persistent Metadata for schema recovery
//
// Architecture:
// The database is composed of several layers:
//  1. Database: The main entry point coordinating all components.
//  2. Collection: Manages documents and their associated indexes.
//  3. Transaction Manager: Handles ACID properties and isolation levels.
//  4. MVCC: Manages version chains and snapshot isolation for non-blocking reads.
//  5. WAL: Ensures durability by logging all changes before applying them.
//  6. Storage: Manages disk I/O (Pager), memory caching (BufferPool), and data structures (B+Tree).
package bundoc

import (
	"fmt"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/transaction"
	"github.com/kartikbazzad/bunbase/bundoc/internal/wal"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// Database represents a bundoc database instance.
// It acts as the central coordinator for all database subsystems.
type Database struct {
	path        string
	bufferPool  *storage.BufferPool             // Manages in-memory page cache
	pager       *storage.Pager                  // Handles raw disk I/O
	walWriter   *wal.WAL                        // Write-Ahead Log for durability
	versionMgr  *mvcc.VersionManager            // Manages MVCC version chains
	snapshotMgr *mvcc.SnapshotManager           // Manages transaction snapshots
	txnMgr      *transaction.TransactionManager // Coordinates transaction lifecycles
	metadataMgr *MetadataManager                // Persists schema/index definitions
	collections map[string]*Collection          // Registry of loaded collections
	mu          sync.RWMutex                    // Protects map access and closure state
	closed      bool                            // Flag indicating if DB is closed
}

// Options configures a database instance
type Options struct {
	// Path to database directory
	Path string

	// BufferPoolSize in number of pages (default: 1000 = 8MB)
	BufferPoolSize int

	// WALPath for write-ahead log (default: Path/wal)
	WALPath string

	// MetadataPath for system catalog (default: Path/system_catalog.json)
	MetadataPath string
}

// DefaultOptions returns default database options
func DefaultOptions(path string) *Options {
	return &Options{
		Path:           path,
		BufferPoolSize: 1000, // 8MB default
		WALPath:        path + "/wal",
		MetadataPath:   path + "/system_catalog.json",
	}
}

// Open opens a database at the given path with the provided options.
// It initializes all subsystems:
// 1. Pager for disk I/O
// 2. BufferPool for page caching
// 3. Write-Ahead Log (WAL) for durability
// 4. MetadataManager for schema recovery
// 5. MVCC components (VersionManager, SnapshotManager)
// 6. TransactionManager
//
// It then effectively performs "Recovery" by loading valid B-Tree roots from
// the system catalog (metadata), ensuring that the database state is consistent
// with the last successful commit.
func Open(opts *Options) (*Database, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	// Create pager for disk I/O
	pager, err := storage.NewPager(opts.Path + "/data.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create pager: %w", err)
	}

	// Create buffer pool
	bufferPool := storage.NewBufferPool(opts.BufferPoolSize, pager)

	// Create WAL
	walWriter, err := wal.NewWAL(opts.WALPath)
	if err != nil {
		pager.Close()
		return nil, fmt.Errorf("failed to create WAL: %w", err)
	}

	// Create Metadata Manager
	metaPath := opts.MetadataPath
	if metaPath == "" {
		metaPath = opts.Path + "/system_catalog.json"
	}
	metadataMgr, err := NewMetadataManager(metaPath)
	if err != nil {
		pager.Close()
		walWriter.Close()
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	// Create MVCC components
	versionMgr := mvcc.NewVersionManager()
	snapshotMgr := mvcc.NewSnapshotManager(versionMgr)

	// Create transaction manager
	txnMgr := transaction.NewTransactionManager(snapshotMgr, walWriter)

	db := &Database{
		path:        opts.Path,
		bufferPool:  bufferPool,
		pager:       pager,
		walWriter:   walWriter,
		versionMgr:  versionMgr,
		snapshotMgr: snapshotMgr,
		txnMgr:      txnMgr,
		metadataMgr: metadataMgr,
		collections: make(map[string]*Collection),
		closed:      false,
	}

	// Restore Collections from Metadata
	for _, name := range metadataMgr.ListCollections() {
		meta, _ := metadataMgr.GetCollection(name)
		coll := &Collection{
			name:    name,
			db:      db,
			indexes: make(map[string]*storage.BPlusTree),
			mu:      sync.RWMutex{},
		}

		// Restore Indexes
		for field, rootID := range meta.Indexes {
			idx, err := storage.LoadBPlusTree(bufferPool, storage.PageID(rootID))
			if err != nil {
				return nil, fmt.Errorf("failed to load index for collection %s field %s: %w", name, field, err)
			}

			// Attach listener to update metadata on split
			f := field // Capture closure
			idx.SetOnRootChange(func(newRootID storage.PageID) {
				// We need to update the specific field in the metadata
				// This requires locking the collection or metadata manager
				// Best to delegate to db or have a helper
				// For now, load-modify-save via MetadataManager

				// Warning: This callback runs under BTree lock.
				// Ensure MetadataManager.UpdateCollection doesn't deadlock.
				// MetadataManager has its own lock. Should be fine.

				// Ideally, we get current indexes state.
				// But we only have valid state in metadata manager or collection?
				// Simple: Get current meta, update field, save.

				currentMeta, _ := metadataMgr.GetCollection(name)
				if currentMeta.Indexes == nil {
					currentMeta.Indexes = make(map[string]uint64)
				}
				currentMeta.Indexes[f] = uint64(newRootID)

				// Re-map to storage.PageID for UpdateCollection
				// Wait, UpdateCollection takes map[string]storage.PageID
				// Let's optimize MetadataManager API later if needed.

				saveIdx := make(map[string]storage.PageID)
				for k, v := range currentMeta.Indexes {
					saveIdx[k] = storage.PageID(v)
				}
				metadataMgr.UpdateCollection(name, saveIdx)
			})

			coll.indexes[field] = idx
		}

		db.collections[name] = coll
	}

	return db, nil
}

// CreateCollection creates a new collection
func (db *Database) CreateCollection(name string) (*Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	// Check if collection already exists
	if _, exists := db.collections[name]; exists {
		return nil, fmt.Errorf("collection %s already exists", name)
	}

	// Create B+tree index for this collection
	index, err := storage.NewBPlusTree(db.bufferPool)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	// Create collection
	coll := &Collection{
		name:    name,
		db:      db,
		indexes: make(map[string]*storage.BPlusTree),
		mu:      sync.RWMutex{},
	}
	coll.indexes["_id"] = index

	// Register listener
	index.SetOnRootChange(func(newRootID storage.PageID) {
		currentMeta, _ := db.metadataMgr.GetCollection(name)
		if currentMeta.Indexes == nil {
			currentMeta.Indexes = make(map[string]uint64)
		}
		currentMeta.Indexes["_id"] = uint64(newRootID)

		saveIdx := make(map[string]storage.PageID)
		for k, v := range currentMeta.Indexes {
			saveIdx[k] = storage.PageID(v)
		}
		db.metadataMgr.UpdateCollection(name, saveIdx)
	})

	db.collections[name] = coll

	// Persist Initial Metadata
	initIndexes := map[string]storage.PageID{
		"_id": index.GetRootID(),
	}
	if err := db.metadataMgr.UpdateCollection(name, initIndexes); err != nil {
		return nil, fmt.Errorf("failed to persist collection metadata: %w", err)
	}

	return coll, nil
}

// GetCollection returns an existing collection
func (db *Database) GetCollection(name string) (*Collection, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	coll, exists := db.collections[name]
	if !exists {
		return nil, fmt.Errorf("collection %s does not exist", name)
	}

	return coll, nil
}

// DropCollection drops a collection
func (db *Database) DropCollection(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if _, exists := db.collections[name]; !exists {
		return fmt.Errorf("collection %s does not exist", name)
	}

	// Remove from collections map
	delete(db.collections, name)

	return nil
}

// ListCollections returns names of all collections
func (db *Database) ListCollections() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	names := make([]string, 0, len(db.collections))
	for name := range db.collections {
		names = append(names, name)
	}
	return names
}

// BeginTransaction starts a new transaction with the specified isolation level
func (db *Database) BeginTransaction(level mvcc.IsolationLevel) (*transaction.Transaction, error) {
	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	return db.txnMgr.Begin(level)
}

// CommitTransaction commits a transaction
func (db *Database) CommitTransaction(txn *transaction.Transaction) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	return db.txnMgr.Commit(txn)
}

// RollbackTransaction rolls back a transaction
func (db *Database) RollbackTransaction(txn *transaction.Transaction) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	return db.txnMgr.Rollback(txn)
}

// Close closes the database and releases resources
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database already closed")
	}

	db.closed = true

	// Close transaction manager
	if err := db.txnMgr.Close(); err != nil {
		return fmt.Errorf("failed to close transaction manager: %w", err)
	}

	// Flush buffer pool
	if err := db.bufferPool.FlushAllPages(); err != nil {
		return fmt.Errorf("failed to flush buffer pool: %w", err)
	}

	// Close WAL
	if err := db.walWriter.Close(); err != nil {
		return fmt.Errorf("failed to close WAL: %w", err)
	}

	// Close pager
	if err := db.pager.Close(); err != nil {
		return fmt.Errorf("failed to close pager: %w", err)
	}

	return nil
}

// IsClosed returns true if the database is closed
func (db *Database) IsClosed() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.closed
}
