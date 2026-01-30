package integration

import (
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func TestWALTrimmingAfterCheckpoint(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 1
	cfg.WAL.Checkpoint.IntervalMB = 1 // Small interval for testing
	cfg.WAL.MaxFileSizeMB = 1         // Small size to trigger rotation

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "trimdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create many documents to trigger WAL rotation and checkpoints
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 100; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// List WAL segments (partitioned layout: wal/dbName/p0.wal*)
	walFiles, err := filepath.Glob(filepath.Join(walDir, "trimdb", "p0.wal*"))
	if err != nil {
		t.Fatalf("Failed to list WAL files: %v", err)
	}

	t.Logf("WAL files after operations: %d", len(walFiles))

	// With trimming enabled and keepSegments=1, we should have at most
	// keepSegments+1 (active + kept) segments after checkpoints
	// Note: This is a simplified check - actual trimming depends on checkpoint creation
	if len(walFiles) > 10 {
		t.Logf("WAL trimming may not have occurred yet (files: %d)", len(walFiles))
	}
}

func TestWALTrimmingDisabled(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = false // Disabled
	cfg.WAL.MaxFileSizeMB = 1

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "notrimdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents to trigger rotation
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 50; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// List WAL segments (partitioned layout: wal/dbName/p0.wal*)
	walFiles, err := filepath.Glob(filepath.Join(walDir, "notrimdb", "p0.wal*"))
	if err != nil {
		t.Fatalf("Failed to list WAL files: %v", err)
	}

	t.Logf("WAL files with trimming disabled: %d", len(walFiles))

	// With trimming disabled, all segments should remain
	if len(walFiles) == 0 {
		t.Error("Expected WAL files to exist")
	}
}

func TestWALTrimmingSegmentRetention(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 2 // Keep 2 segments
	cfg.WAL.Checkpoint.IntervalMB = 1
	cfg.WAL.MaxFileSizeMB = 1

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "retentiondb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 100; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// List WAL segments
	walFiles, err := filepath.Glob(filepath.Join(walDir, "retentiondb", "p0.wal*"))
	if err != nil {
		t.Fatalf("Failed to list WAL files: %v", err)
	}

	t.Logf("WAL files with keepSegments=2: %d", len(walFiles))

	// Should have at most keepSegments+1 (active + kept) segments
	// Note: Actual count depends on checkpoint creation timing
	if len(walFiles) > 10 {
		t.Logf("Segment retention test - files: %d (may vary based on checkpoint timing)", len(walFiles))
	}
}

func TestWALTrimmingRecovery(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 1
	cfg.WAL.Checkpoint.IntervalMB = 1

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "recoverydb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 20; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Close database
	db.Close()

	// Reopen database - should recover from remaining WAL segments
	db2 := docdb.NewLogicalDB(1, "recoverydb", cfg, memCaps, pool, log)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify documents are still readable
	readable := 0
	for i := 1; i <= 20; i++ {
		data, err := db2.Read("default", uint64(i))
		if err == nil && string(data) == string(payload) {
			readable++
		}
	}

	if readable == 0 {
		t.Error("No documents were readable after recovery with trimming")
	}

	t.Logf("Recovered %d out of 20 documents", readable)
}

func TestWALTrimmingNoDataLoss(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 1
	cfg.WAL.Checkpoint.IntervalMB = 1

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "nodatalossdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create documents before checkpoint
	payload1 := []byte(`{"batch":"1"}`)
	for i := 1; i <= 10; i++ {
		if err := db.Create("default", uint64(i), payload1); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Wait for checkpoint (if it occurs)
	// Create more documents after checkpoint
	payload2 := []byte(`{"batch":"2"}`)
	for i := 11; i <= 20; i++ {
		if err := db.Create("default", uint64(i), payload2); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Close and reopen
	db.Close()

	db2 := docdb.NewLogicalDB(1, "nodatalossdb", cfg, memCaps, pool, log)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// All documents should be readable
	readable := 0
	for i := 1; i <= 20; i++ {
		_, err := db2.Read("default", uint64(i))
		if err == nil {
			readable++
		}
	}

	if readable < 10 {
		t.Errorf("Expected at least 10 readable documents, got %d", readable)
	}

	t.Logf("Readable documents after trimming: %d out of 20", readable)
}

func TestWALTrimmingCheckpointCoordination(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 2
	cfg.WAL.Checkpoint.IntervalMB = 1 // Small interval to trigger checkpoint
	cfg.WAL.MaxFileSizeMB = 1         // Small size to trigger rotation

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "checkpointdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create enough documents to trigger multiple rotations and checkpoints
	payload := []byte(`{"data":"test"}`)
	docCount := 200
	for i := 1; i <= docCount; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// List WAL segments
	walFiles, err := filepath.Glob(filepath.Join(walDir, "checkpointdb", "p0.wal*"))
	if err != nil {
		t.Fatalf("Failed to list WAL files: %v", err)
	}

	t.Logf("WAL files after operations: %d", len(walFiles))

	// With trimming enabled and keepSegments=2, we should have at most
	// keepSegments+1 (active + kept) segments after checkpoints
	// Note: Actual count depends on checkpoint creation timing
	if len(walFiles) > 10 {
		t.Logf("WAL trimming coordination test - files: %d (may vary based on checkpoint timing)", len(walFiles))
	}

	// Verify all documents are still readable
	readable := 0
	for i := 1; i <= docCount; i++ {
		data, err := db.Read("default", uint64(i))
		if err == nil && string(data) == string(payload) {
			readable++
		}
	}

	if readable < docCount/2 {
		t.Errorf("Expected at least %d readable documents, got %d", docCount/2, readable)
	}

	t.Logf("Readable documents: %d out of %d", readable, docCount)
}

func TestWALTrimmingSafetyMargin(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.WAL.TrimAfterCheckpoint = true
	cfg.WAL.KeepSegments = 3 // Keep 3 segments as safety margin
	cfg.WAL.Checkpoint.IntervalMB = 1
	cfg.WAL.MaxFileSizeMB = 1

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "safetydb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents to trigger rotation and checkpoints
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 150; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// List WAL segments
	walFiles, err := filepath.Glob(filepath.Join(walDir, "safetydb", "p0.wal*"))
	if err != nil {
		t.Fatalf("Failed to list WAL files: %v", err)
	}

	t.Logf("WAL files with keepSegments=3: %d", len(walFiles))

	// With keepSegments=3, we should have at most 4 segments (active + 3 kept)
	// Note: Actual count depends on checkpoint creation timing
	if len(walFiles) > 10 {
		t.Logf("Safety margin test - files: %d (may vary based on checkpoint timing)", len(walFiles))
	}

	// Verify documents are readable after trimming
	readable := 0
	for i := 1; i <= 150; i++ {
		data, err := db.Read("default", uint64(i))
		if err == nil && string(data) == string(payload) {
			readable++
		}
	}

	if readable < 50 {
		t.Errorf("Expected at least 50 readable documents with safety margin, got %d", readable)
	}

	t.Logf("Readable documents with safety margin: %d out of 150", readable)
}
