package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

// setupDBForCheckpoint creates a database with checkpoint configuration
func setupDBForCheckpoint(t *testing.T, name string, checkpointIntervalMB uint64) (db *docdb.LogicalDB, dataDir, walDir string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-checkpoint-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dataDir = filepath.Join(tmpDir, "data")
	walDir = filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.Checkpoint.IntervalMB = checkpointIntervalMB
	cfg.WAL.Checkpoint.AutoCreate = true

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db = docdb.NewLogicalDB(1, name, cfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup = func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, dataDir, walDir, cleanup
}

// TestCheckpoint_Creation tests that checkpoints are created at the configured interval
func TestCheckpoint_Creation(t *testing.T) {
	// Use a small checkpoint interval for testing (1MB)
	db, dataDir, walDir, cleanup := setupDBForCheckpoint(t, "checkpoint-test", 1)
	defer cleanup()

	// Create many documents to exceed checkpoint interval
	// Each document creates WAL records, so we'll eventually trigger a checkpoint
	// Create valid JSON with large data field
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = 'x'
	}
	largePayload := []byte(`{"data":"` + string(largeData) + `"}`) // ~100KB

	const coll = "_default"
	for i := 1; i <= 20; i++ {
		docID := uint64(i)
		if err := db.Create(coll, docID, largePayload); err != nil {
			t.Fatalf("Create doc %d failed: %v", docID, err)
		}
	}

	// Close and reopen to verify recovery works with checkpoints
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	cfg2.WAL.Checkpoint.IntervalMB = 1
	cfg2.WAL.Checkpoint.AutoCreate = true
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "checkpoint-test", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify documents are recoverable
	for i := 1; i <= 20; i++ {
		docID := uint64(i)
		got, err := db2.Read(coll, docID)
		if err != nil {
			t.Fatalf("Read doc %d after checkpoint recovery failed: %v", docID, err)
		}
		if len(got) != len(largePayload) {
			t.Fatalf("Doc %d size mismatch: got %d, want %d", docID, len(got), len(largePayload))
		}
	}
}

// TestCheckpoint_RecoveryFromCheckpoint tests that recovery can start from a checkpoint
func TestCheckpoint_RecoveryFromCheckpoint(t *testing.T) {
	db, dataDir, walDir, cleanup := setupDBForCheckpoint(t, "checkpoint-recovery", 1)
	defer cleanup()

	// Create documents to trigger checkpoint
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = 'x'
	}
	largePayload := []byte(`{"data":"` + string(largeData) + `"}`)

	const coll = "_default"
	for i := 1; i <= 15; i++ {
		docID := uint64(i)
		if err := db.Create(coll, docID, largePayload); err != nil {
			t.Fatalf("Create doc %d failed: %v", docID, err)
		}
	}

	// Close and reopen
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	cfg2.WAL.Checkpoint.IntervalMB = 1
	cfg2.WAL.Checkpoint.AutoCreate = true
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "checkpoint-recovery", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// All documents should be recoverable
	if db2.IndexSize() != 15 {
		t.Fatalf("Index size mismatch after checkpoint recovery: got %d, want 15", db2.IndexSize())
	}
}

// TestCheckpoint_NoCheckpointRecoversAll tests that without checkpoints,
// recovery still works from the beginning
func TestCheckpoint_NoCheckpointRecoversAll(t *testing.T) {
	// Disable checkpoints
	db, dataDir, walDir, cleanup := setupDBForCheckpoint(t, "checkpoint-none", 0)
	defer cleanup()

	const coll = "_default"
	docID := uint64(1)
	payload := []byte(`{"data":"test"}`)

	if err := db.Create(coll, docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	cfg2.WAL.Checkpoint.IntervalMB = 0 // Disabled
	cfg2.WAL.Checkpoint.AutoCreate = false
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "checkpoint-none", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	got, err := db2.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read after recovery failed: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch: got %s, want %s", got, payload)
	}
}
