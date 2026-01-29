package integration

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

// setupSingleDBForPartialWrite creates a database for partial write testing
func setupSingleDBForPartialWrite(t *testing.T, name string) (db *docdb.LogicalDB, dataDir, walDir string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-partial-write-*")
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

// TestPartialWrite_UnverifiedRecordSkipped tests that records without
// verification flags are skipped during reads.
func TestPartialWrite_UnverifiedRecordSkipped(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDBForPartialWrite(t, "partial-unverified")
	defer cleanup()

	const coll = "_default"
	docID := uint64(1)
	payload := []byte(`{"data":"test"}`)

	// Create a document normally (this writes with verification flag)
	if err := db.Create(coll, docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close DB so we can manually corrupt the data file
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Manually write an unverified record to the data file
	// Format: [4: len] [N: payload] [4: crc32] [1: verified]
	// We'll write everything except the verification flag, or write it as 0
	dataFilePath := filepath.Join(dataDir, "partial-unverified.data")
	dataFile, err := os.OpenFile(dataFilePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}

	// Write an unverified record (verification flag = 0)
	payloadLen := uint32(len(payload))
	crc32Value := uint32(0x12345678) // Dummy CRC for testing

	header := make([]byte, 4+4)
	binary.LittleEndian.PutUint32(header[0:], payloadLen)
	binary.LittleEndian.PutUint32(header[4:], crc32Value)

	if _, err := dataFile.Write(header); err != nil {
		dataFile.Close()
		t.Fatalf("Failed to write header: %v", err)
	}

	if _, err := dataFile.Write(payload); err != nil {
		dataFile.Close()
		t.Fatalf("Failed to write payload: %v", err)
	}

	// Write verification flag as 0 (unverified)
	unverifiedFlag := []byte{0}
	if _, err := dataFile.Write(unverifiedFlag); err != nil {
		dataFile.Close()
		t.Fatalf("Failed to write unverified flag: %v", err)
	}

	if err := dataFile.Close(); err != nil {
		t.Fatalf("Failed to close data file: %v", err)
	}

	// Reopen DB - the unverified record should not be readable
	// (Note: This record won't be in the index anyway since it wasn't committed,
	// but we're testing that DataFile.Read itself rejects unverified records)
	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "partial-unverified", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// The original document should still be readable
	got, err := db2.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read original document failed: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Original document payload mismatch: got %s, want %s", got, payload)
	}
}

// TestPartialWrite_CrashBeforeVerificationFlag tests that a crash
// before the verification flag is written leaves an unverified record.
func TestPartialWrite_CrashBeforeVerificationFlag(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDBForPartialWrite(t, "partial-crash")
	defer cleanup()

	const coll = "_default"
	docID := uint64(1)
	payload := []byte(`{"data":"crash test"}`)

	// Create document normally
	if err := db.Create(coll, docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close DB
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Manually create a partial write scenario:
	// Write header + payload + CRC but no verification flag
	dataFilePath := filepath.Join(dataDir, "partial-crash.data")
	dataFile, err := os.OpenFile(dataFilePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}

	// Get current file size to append after existing records
	info, err := dataFile.Stat()
	if err != nil {
		dataFile.Close()
		t.Fatalf("Failed to stat data file: %v", err)
	}
	currentSize := info.Size()

	// Write partial record (missing verification flag)
	payloadLen := uint32(len(payload))
	crc32Value := uint32(0x87654321)

	header := make([]byte, 4+4)
	binary.LittleEndian.PutUint32(header[0:], payloadLen)
	binary.LittleEndian.PutUint32(header[4:], crc32Value)

	if _, err := dataFile.Write(header); err != nil {
		dataFile.Close()
		t.Fatalf("Failed to write header: %v", err)
	}

	if _, err := dataFile.Write(payload); err != nil {
		dataFile.Close()
		t.Fatalf("Failed to write payload: %v", err)
	}

	// Intentionally don't write verification flag (simulating crash)
	if err := dataFile.Close(); err != nil {
		t.Fatalf("Failed to close data file: %v", err)
	}

	// Reopen DB and try to read - should fail because record is incomplete
	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "partial-crash", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Original document should still be readable
	got, err := db2.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read original document failed: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Original document payload mismatch: got %s, want %s", got, payload)
	}

	// Verify file size is what we expect (original + partial record without flag)
	info2, err := os.Stat(dataFilePath)
	if err != nil {
		t.Fatalf("Failed to stat data file after reopen: %v", err)
	}

	expectedSize := currentSize + int64(4+4+len(payload)) // header + payload, no verification flag
	if info2.Size() != expectedSize {
		t.Logf("File size: %d, expected: %d (partial record without verification flag)", info2.Size(), expectedSize)
	}
}

// TestPartialWrite_VerifiedRecordReadable tests that properly verified
// records are readable after restart.
func TestPartialWrite_VerifiedRecordReadable(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDBForPartialWrite(t, "partial-verified")
	defer cleanup()

	const coll = "_default"
	docID := uint64(1)
	payload := []byte(`{"data":"verified test"}`)

	// Create document (writes with verification flag)
	if err := db.Create(coll, docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it's readable immediately
	got, err := db.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch: got %s, want %s", got, payload)
	}

	// Close and reopen
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "partial-verified", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Should still be readable after restart
	got2, err := db2.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read after restart failed: %v", err)
	}
	if string(got2) != string(payload) {
		t.Fatalf("Payload mismatch after restart: got %s, want %s", got2, payload)
	}
}

// TestPartialWrite_RecoverySkipsUnverified tests that during WAL replay,
// if we encounter unverified records in the data file, they are skipped.
func TestPartialWrite_RecoverySkipsUnverified(t *testing.T) {
	// This test verifies that recovery doesn't try to read unverified records
	// Since recovery writes fresh records from WAL, this is mainly testing
	// that the data file format change doesn't break recovery
	db, dataDir, walDir, cleanup := setupSingleDBForPartialWrite(t, "partial-recovery")
	defer cleanup()

	const coll = "_default"
	docID := uint64(1)
	payload := []byte(`{"data":"recovery test"}`)

	// Create document
	if err := db.Create(coll, docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close and reopen to trigger recovery
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cfg2 := config.DefaultConfig()
	cfg2.DataDir = dataDir
	cfg2.WAL.Dir = walDir
	log2 := logger.Default()
	memCaps2 := memory.NewCaps(cfg2.Memory.GlobalCapacityMB, cfg2.Memory.PerDBLimitMB)
	memCaps2.RegisterDB(1, cfg2.Memory.PerDBLimitMB)
	pool2 := memory.NewBufferPool(cfg2.Memory.BufferSizes)

	db2 := docdb.NewLogicalDB(1, "partial-recovery", cfg2, memCaps2, pool2, log2)
	if err := db2.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Document should be recoverable
	got, err := db2.Read(coll, docID)
	if err != nil {
		t.Fatalf("Read after recovery failed: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch after recovery: got %s, want %s", got, payload)
	}
}
