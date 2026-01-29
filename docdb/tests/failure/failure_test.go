package failure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func setupTestDB(t *testing.T) (*docdb.LogicalDB, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-fail-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "faildb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, tmpDir, cleanup
}

func TestCorruptWAL(t *testing.T) {
	db, tmpDir, cleanup := setupTestDB(t)
	defer cleanup()

	// Use valid JSON payload so engine-level JSON enforcement still allows
	// us to exercise WAL corruption behavior.
	const coll = "_default"
	payload := []byte(`{"data":"test payload"}`)
	docID := uint64(1)

	err := db.Create(coll, docID, payload)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	walPath := filepath.Join(tmpDir, "wal", "faildb", "p0.wal")
	walData, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("Failed to read WAL: %v", err)
	}

	if len(walData) > 10 {
		corruptData := make([]byte, len(walData)-5)
		copy(corruptData, walData[:5])
		for i := 5; i < len(corruptData); i++ {
			corruptData[i] = 0xFF
		}

		if err := os.WriteFile(walPath, corruptData, 0644); err != nil {
			t.Fatalf("Failed to corrupt WAL: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "faildb", cfg, memCaps, pool, log)
	if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to reopen database after corrupted WAL: %v", err)
	}
	defer db2.Close()

	_, err = db2.Read(coll, docID)
	if err == nil {
		t.Fatal("Expected error when reading after corrupted WAL")
	}

	t.Logf("Successfully handled corrupted WAL")
}

func TestTruncatedWAL(t *testing.T) {
	db, tmpDir, cleanup := setupTestDB(t)
	defer cleanup()

	const coll = "_default"
	// Valid JSON payload for truncated WAL scenario.
	payload := []byte(`{"data":"test payload"}`)

	for i := 0; i < 10; i++ {
		err := db.Create(coll, uint64(i+1), payload)
		if err != nil {
			t.Fatalf("Failed to create document %d: %v", i+1, err)
		}
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	walPath := filepath.Join(tmpDir, "wal", "faildb", "p0.wal")
	walData, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("Failed to read WAL: %v", err)
	}

	truncatedSize := len(walData) / 2
	if err := os.WriteFile(walPath, walData[:truncatedSize], 0644); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "faildb", cfg, memCaps, pool, log)
	if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to reopen database after truncated WAL: %v", err)
	}
	defer db2.Close()

	for i := 1; i <= 5; i++ {
		_, err := db2.Read(coll, uint64(i))
		if err != nil {
			t.Logf("Document %d may not be available after truncation: %v", i, err)
		}
	}

	t.Logf("Successfully handled truncated WAL")
}

func TestMissingWAL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docdb-missingwal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "missingwaldb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Valid JSON payload for missing WAL scenario.
	const coll = "_default"
	payload := []byte(`{"data":"test payload"}`)

	err = db.Create(coll, 1, payload)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	walPath := filepath.Join(tmpDir, "wal", "missingwaldb", "p0.wal")
	if err := os.Remove(walPath); err != nil {
		t.Fatalf("Failed to remove WAL: %v", err)
	}

	db2 := docdb.NewLogicalDB(1, "missingwaldb", cfg, memCaps, pool, log)
	if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to reopen database after missing WAL: %v", err)
	}
	defer db2.Close()

	retrieved, err := db2.Read(coll, 1)
	if err != nil {
		t.Logf("Document may not be available after missing WAL: %v", err)
	} else {
		if string(retrieved) != string(payload) {
			t.Fatalf("Payload mismatch: got %s, want %s", retrieved, payload)
		}
	}

	t.Logf("Successfully handled missing WAL")
}
