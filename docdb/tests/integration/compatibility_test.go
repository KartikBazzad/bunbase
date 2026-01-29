package integration

import (
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func TestV01DatabaseOpenInV02(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	// Create a v0.1 database (without collections)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents without collection (should use _default)
	payload1 := []byte(`{"name":"doc1"}`)
	if err := db.Create("_default", 1, payload1); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	payload2 := []byte(`{"name":"doc2"}`)
	if err := db.Create("_default", 2, payload2); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	db.Close()

	// Reopen database - should work with v0.2
	db2 := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify documents are accessible via _default collection
	data, err := db2.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document 1: %v", err)
	}
	if string(data) != string(payload1) {
		t.Errorf("Document 1 mismatch: got %s, want %s", string(data), string(payload1))
	}

	data, err = db2.Read("_default", 2)
	if err != nil {
		t.Fatalf("Failed to read document 2: %v", err)
	}
	if string(data) != string(payload2) {
		t.Errorf("Document 2 mismatch: got %s, want %s", string(data), string(payload2))
	}

	// Verify _default collection exists
	collections := db2.ListCollections()
	found := false
	for _, coll := range collections {
		if coll == "_default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("_default collection not found")
	}
}

func TestV01DocumentsInDefaultCollection(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents using empty collection (should default to _default)
	payload1 := []byte(`{"name":"doc1"}`)
	if err := db.Create("", 1, payload1); err != nil {
		t.Fatalf("Failed to create document with empty collection: %v", err)
	}

	// Read using _default collection
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}
	if string(data) != string(payload1) {
		t.Errorf("Document mismatch: got %s, want %s", string(data), string(payload1))
	}

	// Read using empty collection (should also work)
	data, err = db.Read("", 1)
	if err != nil {
		t.Fatalf("Failed to read document with empty collection: %v", err)
	}
	if string(data) != string(payload1) {
		t.Errorf("Document mismatch: got %s, want %s", string(data), string(payload1))
	}
}

func TestMixedV01V02WALReplay(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create documents in _default collection (v0.1 style)
	payload1 := []byte(`{"name":"doc1"}`)
	if err := db.Create("_default", 1, payload1); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Create a new collection
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Create document in new collection (v0.2 style)
	payload2 := []byte(`{"name":"user1"}`)
	if err := db.Create("users", 1, payload2); err != nil {
		t.Fatalf("Failed to create document in users collection: %v", err)
	}

	db.Close()

	// Reopen and verify both collections work
	db2 := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify _default collection document
	data, err := db2.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read from _default: %v", err)
	}
	if string(data) != string(payload1) {
		t.Errorf("_default document mismatch")
	}

	// Verify users collection document
	data, err = db2.Read("users", 1)
	if err != nil {
		t.Fatalf("Failed to read from users: %v", err)
	}
	if string(data) != string(payload2) {
		t.Errorf("users document mismatch")
	}

	// Verify collections list
	collections := db2.ListCollections()
	if len(collections) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(collections))
	}
}

func TestV01OperationsStillWork(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// All operations with empty collection should default to _default
	payload := []byte(`{"name":"test"}`)

	// Create with empty collection
	if err := db.Create("", 1, payload); err != nil {
		t.Fatalf("Create with empty collection failed: %v", err)
	}

	// Read with empty collection
	data, err := db.Read("", 1)
	if err != nil {
		t.Fatalf("Read with empty collection failed: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("Read data mismatch")
	}

	// Update with empty collection
	updatedPayload := []byte(`{"name":"updated"}`)
	if err := db.Update("", 1, updatedPayload); err != nil {
		t.Fatalf("Update with empty collection failed: %v", err)
	}

	// Verify update
	data, err = db.Read("", 1)
	if err != nil {
		t.Fatalf("Read after update failed: %v", err)
	}
	if string(data) != string(updatedPayload) {
		t.Errorf("Updated data mismatch")
	}

	// Delete with empty collection
	if err := db.Delete("", 1); err != nil {
		t.Fatalf("Delete with empty collection failed: %v", err)
	}

	// Verify deletion
	_, err = db.Read("", 1)
	if err == nil {
		t.Error("Document should be deleted")
	}
}
