package benchmarks

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func BenchmarkCreateDocument(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "docdb-bench-create-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.New(io.Discard, logger.LevelError, "")
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "benchdb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	const coll = "_default"
	payload := []byte("benchmark payload")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		docID := uint64(1)
		for pb.Next() {
			err := db.Create(coll, docID, payload)
			if err != nil {
				b.Fatalf("Failed to create document: %v", err)
			}
			docID++
		}
	})
}

func BenchmarkReadDocument(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "docdb-bench-read-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.New(io.Discard, logger.LevelError, "")
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "benchdb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	const coll = "_default"
	numDocs := 1000
	for i := 1; i <= numDocs; i++ {
		payload := []byte("benchmark payload")
		err := db.Create(coll, uint64(i), payload)
		if err != nil {
			b.Fatalf("Failed to create document: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		docID := uint64(1)
		for pb.Next() {
			_, err := db.Read(coll, docID)
			if err != nil {
				b.Fatalf("Failed to read document: %v", err)
			}
			docID++
			if docID > uint64(numDocs) {
				docID = 1
			}
		}
	})
}

func BenchmarkUpdateDocument(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "docdb-bench-update-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.New(io.Discard, logger.LevelError, "")
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, "benchdb", cfg, memCaps, pool, log)
	if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	const coll = "_default"
	payload := []byte("benchmark payload")
	docID := uint64(1)
	err = db.Create(coll, docID, payload)
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newPayload := []byte("updated benchmark payload")
		err := db.Update(coll, docID, newPayload)
		if err != nil {
			b.Fatalf("Failed to update document: %v", err)
		}
	}
}
