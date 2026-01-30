package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
)

// setupMultiPartitionDB creates a LogicalDB with partitionCount >= 2 for multi-partition tx tests.
func setupMultiPartitionDB(t *testing.T, name string, partitionCount int) (db *docdb.LogicalDB, dataDir, walDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "docdb-multipartition-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dataDir = filepath.Join(tmpDir, "data")
	walDir = filepath.Join(tmpDir, "wal")
	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = partitionCount
	db = docdb.NewLogicalDBWithConfig(1, name, cfg, dbCfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	cleanup = func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, dataDir, walDir, cleanup
}

func reopenMultiPartitionDB(t *testing.T, name, dataDir, walDir string, partitionCount int) *docdb.LogicalDB {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = partitionCount
	db := docdb.NewLogicalDBWithConfig(1, name, cfg, dbCfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	return db
}

// TestMultiPartitionCommit_Success commits a transaction with ops on two partitions and verifies visibility.
func TestMultiPartitionCommit_Success(t *testing.T) {
	db, dataDir, walDir, cleanup := setupMultiPartitionDB(t, "mp-commit", 2)
	defer cleanup()
	// With 2 partitions: docID 1 -> partition 1, docID 2 -> partition 0
	tx := db.Begin()
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 1, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"b":2}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got1, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Read doc 1: %v", err)
	}
	if string(got1) != `{"a":1}` {
		t.Fatalf("doc 1: got %s", got1)
	}
	got2, err := db.Read("_default", 2)
	if err != nil {
		t.Fatalf("Read doc 2: %v", err)
	}
	if string(got2) != `{"b":2}` {
		t.Fatalf("doc 2: got %s", got2)
	}
	// Restart and verify both visible
	db.Close()
	db2 := reopenMultiPartitionDB(t, "mp-commit", dataDir, walDir, 2)
	defer db2.Close()
	got1, _ = db2.Read("_default", 1)
	got2, _ = db2.Read("_default", 2)
	if string(got1) != `{"a":1}` || string(got2) != `{"b":2}` {
		t.Fatalf("after restart: doc1=%s doc2=%s", got1, got2)
	}
}

// TestSinglePartitionCommit_Success uses single-partition fast path (no coordinator).
func TestSinglePartitionCommit_Success(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "sp-commit", 2)
	defer cleanup()
	// Both docs on same partition: docID 0 and 2 -> partition 0
	tx := db.Begin()
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 0, []byte(`{"x":0}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"x":2}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got0, _ := db.Read("_default", 0)
	got2, _ := db.Read("_default", 2)
	if string(got0) != `{"x":0}` || string(got2) != `{"x":2}` {
		t.Fatalf("single-partition commit: got %s, %s", got0, got2)
	}
}

// TestMultiPartition_EmptyTx commits an empty transaction (no-op).
func TestMultiPartition_EmptyTx(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "mp-empty", 2)
	defer cleanup()
	tx := db.Begin()
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit empty tx: %v", err)
	}
}
