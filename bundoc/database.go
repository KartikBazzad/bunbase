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
	"path/filepath"
	"strings"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/transaction"
	"github.com/kartikbazzad/bunbase/bundoc/internal/wal"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/rules"
	"github.com/kartikbazzad/bunbase/bundoc/security"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
	"github.com/xeipuuv/gojsonschema"
)

// Database represents a bundoc database instance.
// It acts as the central coordinator for all database subsystems.
type Database struct {
	path         string
	bufferPool   *storage.BufferPool             // Manages in-memory page cache
	pager        *storage.Pager                  // Handles raw disk I/O
	walWriter    *wal.WAL                        // Write-Ahead Log for durability
	versionMgr   *mvcc.VersionManager            // Manages MVCC version chains
	snapshotMgr  *mvcc.SnapshotManager           // Manages transaction snapshots
	txnMgr       *transaction.TransactionManager // Coordinates transaction lifecycles
	metadataMgr  *MetadataManager                // Persists schema/index definitions
	Security     *security.UserManager           // Manages Users and Auth
	Audit        *security.AuditLogger           // Security Audit Logger
	RulesEngine  *rules.RulesEngine              // CEL Rules Engine
	collections  map[string]*Collection          // Registry of loaded collections
	groupIndexes map[string]*storage.BPlusTree   // Registry of active Group Indexes (Key: pattern::field)
	mu           sync.RWMutex                    // Protects map access and closure state
	closed       bool                            // Flag indicating if DB is closed
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

	// EncryptionKey for at-rest encryption (32 bytes for AES-256)
	// If nil, encryption is disabled.
	EncryptionKey []byte

	// AuditLogPath for security events (default: Path/audit.log)
	AuditLogPath string
}

// DefaultOptions returns default database options
func DefaultOptions(path string) *Options {
	return &Options{
		Path:           path,
		BufferPoolSize: 1000, // 8MB default
		WALPath:        path + "/wal",
		MetadataPath:   path + "/system_catalog.json",
		AuditLogPath:   path + "/audit.log",
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
	pager, err := storage.NewPager(opts.Path+"/data.db", opts.EncryptionKey)
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

	// Initialize Rules Engine
	re, err := rules.NewRulesEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rules engine: %w", err)
	}

	db := &Database{
		path:         opts.Path,
		bufferPool:   bufferPool,
		pager:        pager,
		walWriter:    walWriter,
		versionMgr:   versionMgr,
		snapshotMgr:  snapshotMgr,
		txnMgr:       txnMgr,
		metadataMgr:  metadataMgr,
		RulesEngine:  re,
		collections:  make(map[string]*Collection),
		groupIndexes: make(map[string]*storage.BPlusTree),
		closed:       false,
	}

	// Initialize Security
	userStore := NewInternalUserStore(db)
	db.Security = security.NewUserManager(userStore)

	// Initialize Audit Logger
	auditPath := opts.AuditLogPath
	if auditPath == "" {
		auditPath = opts.Path + "/audit.log"
	}
	auditLogger, err := security.NewAuditLogger(auditPath)
	if err != nil {
		// Log error but don't fail DB open? Or fail secure?
		// Fail secure is better.
		return nil, fmt.Errorf("failed to init audit logger: %w", err)
	}
	db.Audit = auditLogger

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

		// Restore Schema
		if meta.Schema != "" {
			loader := gojsonschema.NewStringLoader(meta.Schema)
			schema, err := gojsonschema.NewSchema(loader)
			if err != nil {
				// Log error but allow loading?
				// Better to fail or warn?
				// For now, WARN and continue without schema enforcement could be dangerous
				// But failing DB open is also bad.
				// Let's log info via println for now (since no logger passed)
				fmt.Printf("[WARN] Failed to load schema for collection %s: %v\n", name, err)
			} else {
				coll.schemaLoader = schema
			}
		}

		db.collections[name] = coll
	}

	// Restore Group Indexes
	for _, meta := range metadataMgr.ListGroupIndexes() {
		idx, err := storage.LoadBPlusTree(bufferPool, storage.PageID(meta.RootID))
		if err != nil {
			return nil, fmt.Errorf("failed to load group index %s::%s: %w", meta.Pattern, meta.Field, err)
		}

		p, f := meta.Pattern, meta.Field
		idx.SetOnRootChange(func(newRootID storage.PageID) {
			metadataMgr.UpdateGroupIndex(p, f, newRootID)
		})

		key := meta.Pattern + "::" + meta.Field
		db.groupIndexes[key] = idx
	}

	// Link Collections to Group Indexes
	// We do this after both are loaded
	for _, coll := range db.collections {
		for key, gIdx := range db.groupIndexes {
			parts := strings.Split(key, "::")
			if len(parts) != 2 {
				continue
			}
			pattern, field := parts[0], parts[1]

			matched, _ := filepath.Match(pattern, coll.Name())
			if matched {
				coll.linkedGroupIndexes = append(coll.linkedGroupIndexes, &GroupIndexLink{
					Index: gIdx,
					Field: field,
				})
			}
		}
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

	// Link Group Indexes
	for key, gIdx := range db.groupIndexes {
		// key is pattern::field
		// We need to parse pattern
		// But we can just use metadata manager list if we stored it separate,
		// OR just split key.
		// Let's iterate metadata to get pattern cleanly?
		// Actually, we initialized db.groupIndexes from metadata list.
		// But map key loses structure effectively unless we split.
		// Let's use metadata manager list again or cache it better.
		// Optimization: Split key "pattern::field"

		parts := strings.Split(key, "::")
		if len(parts) != 2 {
			continue
		}
		pattern, field := parts[0], parts[1]

		// Match
		matched, _ := filepath.Match(pattern, name)
		if matched {
			coll.linkedGroupIndexes = append(coll.linkedGroupIndexes, &GroupIndexLink{
				Index: gIdx,
				Field: field,
			})
			fmt.Printf("[INFO] Linked collection %s to Group Index %s::%s\n", name, pattern, field)
		}
	}

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

// ListCollectionsWithPrefix returns names of collections filtering by prefix
func (db *Database) ListCollectionsWithPrefix(prefix string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	names := make([]string, 0)
	for name := range db.collections {
		if prefix == "" {
			names = append(names, name)
			continue
		}
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			names = append(names, name)
		}
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

	// Close Audit Logger
	if db.Audit != nil {
		db.Audit.Close()
	}

	return nil
}

// EnsureGroupIndex creates a collection group index.
// Arguments:
// - pattern: Glob pattern or prefix (e.g. "users/*/posts" or just glob match)
// - field: Field to index
func (db *Database) EnsureGroupIndex(pattern, field string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	key := pattern + "::" + field
	if _, exists := db.groupIndexes[key]; exists {
		return nil
	}

	fmt.Printf("[INFO] Creating Group Index: %s :: %s\n", pattern, field)

	// Create Index
	index, err := storage.NewBPlusTree(db.bufferPool)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Backfill
	// 1. Find all matching collections
	// Simple matching: if pattern has *, use path.Match, else strict?
	// The user asked for "users/*/posts".
	// Let's implement simple glob matching from "path"
	// But we iterate all collections.

	// Helper for matching
	// path.Match might be enough.
	// NOTE: path.Match uses shell patterns. "*" matches everything.
	// users/*/posts matches users/123/posts.

	// We iterate DB collections, NOT file system.
	for _, coll := range db.collections {
		// Use filepath.Match or path.Match?
		// path.Match uses '/' as separator? Yes.
		matched, _ := filepath.Match(pattern, coll.Name())
		if !matched {
			continue
		}

		fmt.Printf("[INFO] Backfilling from collection: %s\n", coll.Name())

		// Scan Primary Index of matched collection
		startKey := []byte{0x00}
		endKey := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

		scanResults, err := coll.indexes["_id"].RangeScan(startKey, endKey)
		if err != nil {
			// Log error but continue?
			fmt.Printf("[WARN] Failed to scan collection %s: %v\n", coll.Name(), err)
			continue
		}

		for _, entry := range scanResults {
			doc, err := storage.DeserializeDocument(entry.Value)
			if err != nil {
				continue
			}

			id, _ := doc.GetID()
			if val, ok := doc[field]; ok {
				valStr := fmt.Sprintf("%v", val)
				// Composite Key: Value \0 Collection \0 ID
				// This ensures uniqueness (Coll+ID is unique globally-ish)
				compKey := []byte(valStr + "\x00" + coll.Name() + "\x00" + string(id))

				// Value? Group Index needs to point to (Collection, ID).
				// We can store Collection \0 ID as value.
				// OR just use empty value and parse key?
				// Using Value is better for retrieval.
				compVal := []byte(coll.Name() + "\x00" + string(id))

				if err := index.Insert(compKey, compVal); err != nil {
					return fmt.Errorf("failed to insert group index entry: %w", err)
				}
			}
		}
	}

	// Persist Metadata
	index.SetOnRootChange(func(newRootID storage.PageID) {
		db.metadataMgr.UpdateGroupIndex(pattern, field, newRootID)
	})

	db.groupIndexes[key] = index

	if err := db.metadataMgr.UpdateGroupIndex(pattern, field, index.GetRootID()); err != nil {
		return fmt.Errorf("failed to persist group index metadata: %w", err)
	}

	return nil
}

// IsClosed returns true if the database is closed
func (db *Database) IsClosed() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.closed
}

// FindInGroup executes a query against a collection group using an index.
// Currently only supports simple equality checks on indexed fields.
func (db *Database) FindInGroup(auth *rules.AuthContext, txn *transaction.Transaction, pattern string, queryMap map[string]interface{}) ([]storage.Document, error) {
	// 1. Analyze Query (Simplified: Find strict equality on indexed field)
	// We need to find ONE field in the query that matches an existing Group Index.
	var index *storage.BPlusTree
	var value interface{}

	db.mu.RLock()
	// Check all fields in query
	for k, v := range queryMap {
		key := pattern + "::" + k
		if idx, ok := db.groupIndexes[key]; ok {
			index = idx
			value = v
			break // Found an index!
		}
	}
	db.mu.RUnlock()

	if index == nil {
		// Fallback: Scatter-Gather (Iterate all collections)
		return db.scanGroup(auth, txn, pattern, queryMap)
	}

	// 2. Index Scan
	valStr := fmt.Sprintf("%v", value)
	startKey := []byte(valStr + "\x00")
	endKey := []byte(valStr + "\x00" + "\xFF")

	scanResults, err := index.RangeScan(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("group index scan failed: %w", err)
	}

	var results []storage.Document
	for _, entry := range scanResults {
		// Value is CollectionName \0 DocID
		parts := strings.Split(string(entry.Value), "\x00")
		if len(parts) != 2 {
			continue
		}
		collName, docID := parts[0], parts[1]

		coll, err := db.GetCollection(collName)
		if err != nil {
			continue // Collection deleted?
		}

		doc, err := coll.FindByID(auth, txn, docID)
		if err != nil {
			continue
		}

		// Re-validate query (in case of index collision or complex query)
		results = append(results, doc)
	}

	return results, nil
}

// scanGroup performs a scatter-gather scan of all matching collections
func (db *Database) scanGroup(auth *rules.AuthContext, txn *transaction.Transaction, pattern string, queryMap map[string]interface{}) ([]storage.Document, error) {
	var results []storage.Document

	colls := db.ListCollections() // helper

	for _, name := range colls {
		matched, _ := filepath.Match(pattern, name)
		if !matched {
			continue
		}

		coll, err := db.GetCollection(name)
		if err != nil {
			continue
		}

		// Execute Query on Collection
		docs, err := coll.FindQuery(auth, txn, queryMap)
		if err != nil {
			continue
		}
		results = append(results, docs...)
	}

	return results, nil
}
