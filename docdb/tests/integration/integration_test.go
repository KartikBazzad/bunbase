package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/pool"
)

func setupTestPool(t *testing.T) (*pool.Pool, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()

	p := pool.NewPool(cfg, log)
	if err := p.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	cleanup := func() {
		p.Stop()
		os.RemoveAll(tmpDir)
	}

	return p, tmpDir, cleanup
}

func TestDatabaseOperations(t *testing.T) {
	p, _, cleanup := setupTestPool(t)
	defer cleanup()

	t.Run("CreateDatabase", func(t *testing.T) {
		dbID, err := p.CreateDB("testdb")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		if dbID == 0 {
			t.Fatal("Expected non-zero DB ID")
		}

		_, err = p.CreateDB("testdb")
		if err == nil {
			t.Fatal("Expected error when creating duplicate database")
		}
	})

	t.Run("CreateDocument", func(t *testing.T) {
		dbID, err := p.CreateDB("createdb")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		db, err := p.OpenDB(dbID)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		payload := []byte("test payload")
		docID := uint64(1)

		err = db.Create(docID, payload)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		retrieved, err := db.Read(docID)
		if err != nil {
			t.Fatalf("Failed to read document: %v", err)
		}

		if string(retrieved) != string(payload) {
			t.Fatalf("Payload mismatch: got %s, want %s", retrieved, payload)
		}

		err = db.Create(docID, payload)
		if err == nil {
			t.Fatal("Expected error when creating duplicate document")
		}
	})

	t.Run("UpdateDocument", func(t *testing.T) {
		dbID, err := p.CreateDB("updatedb")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		db, err := p.OpenDB(dbID)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		payload := []byte("initial payload")
		docID := uint64(1)

		err = db.Create(docID, payload)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		newPayload := []byte("updated payload")
		err = db.Update(docID, newPayload)
		if err != nil {
			t.Fatalf("Failed to update document: %v", err)
		}

		retrieved, err := db.Read(docID)
		if err != nil {
			t.Fatalf("Failed to read document: %v", err)
		}

		if string(retrieved) != string(newPayload) {
			t.Fatalf("Payload mismatch: got %s, want %s", retrieved, newPayload)
		}
	})

	t.Run("DeleteDocument", func(t *testing.T) {
		dbID, err := p.CreateDB("deletedb")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		db, err := p.OpenDB(dbID)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		payload := []byte("test payload")
		docID := uint64(1)

		err = db.Create(docID, payload)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		err = db.Delete(docID)
		if err != nil {
			t.Fatalf("Failed to delete document: %v", err)
		}

		_, err = db.Read(docID)
		if err != docdb.ErrDocNotFound {
			t.Fatalf("Expected ErrDocNotFound, got: %v", err)
		}
	})

	t.Run("MultipleDocuments", func(t *testing.T) {
		dbID, err := p.CreateDB("multidb")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		db, err := p.OpenDB(dbID)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		numDocs := 100
		for i := 1; i <= numDocs; i++ {
			payload := []byte("payload for doc")
			err := db.Create(uint64(i), payload)
			if err != nil {
				t.Fatalf("Failed to create document %d: %v", i, err)
			}
		}

		for i := 1; i <= numDocs; i++ {
			_, err := db.Read(uint64(i))
			if err != nil {
				t.Fatalf("Failed to read document %d: %v", i, err)
			}
		}

		if db.IndexSize() != numDocs {
			t.Fatalf("Index size mismatch: got %d, want %d", db.IndexSize(), numDocs)
		}
	})
}

func TestMVCC(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docdb-mvcc-test-*")
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

	db := docdb.NewLogicalDB(1, "mvccdb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	payload1 := []byte("version 1")
	docID := uint64(1)

	err = db.Create(docID, payload1)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	payload2 := []byte("version 2")
	err = db.Update(docID, payload2)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	retrieved, err := db.Read(docID)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	if string(retrieved) != string(payload2) {
		t.Fatalf("Payload mismatch: got %s, want %s", retrieved, payload2)
	}
}

func TestMemoryLimits(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docdb-memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
	cfg.Memory.GlobalCapacityMB = 1
	cfg.Memory.PerDBLimitMB = 1

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "memorydb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	largePayload := make([]byte, 2*1024*1024)
	docID := uint64(1)

	err = db.Create(docID, largePayload)
	if err != docdb.ErrMemoryLimit {
		t.Fatalf("Expected ErrMemoryLimit, got: %v", err)
	}
}

func TestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docdb-persist-test-*")
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

	db := docdb.NewLogicalDB(1, "persistdb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	payload := []byte("persistent payload")
	docID := uint64(1)

	err = db.Create(docID, payload)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	db2 := docdb.NewLogicalDB(1, "persistdb", cfg, memCaps, pool, log)
	if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	retrieved, err := db2.Read(docID)
	if err != nil {
		t.Fatalf("Failed to read document after restart: %v", err)
	}

	if string(retrieved) != string(payload) {
		t.Fatalf("Payload mismatch after restart: got %s, want %s", retrieved, payload)
	}
}
