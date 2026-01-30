package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/internal/wal"
)

// setupSingleDB creates a single LogicalDB instance backed by temporary
// data/WAL directories. It mirrors the patterns used in other integration
// tests but focuses on a single database for write-ordering scenarios.
func setupSingleDB(t *testing.T, name string) (db *docdb.LogicalDB, dataDir, walDir string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-write-ordering-*")
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

// reopenDB reopens a LogicalDB instance against existing data/WAL directories
// to exercise WAL replay after simulated crashes.
func reopenDB(t *testing.T, name, dataDir, walDir string) *docdb.LogicalDB {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, name, cfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}

	return db
}

// TestWriteOrdering_NormalCommit verifies that a normal commit path (single-op
// create) survives restart and remains visible.
func TestWriteOrdering_NormalCommit(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-normal")
	defer cleanup()

	docID := uint64(1)
	payload := []byte(`{"data":"normal commit"}`)

	if err := db.Create("_default", docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Simulate clean shutdown and restart.
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db2 := reopenDB(t, "writeorder-normal", dataDir, walDir)
	defer db2.Close()

	got, err := db2.Read("_default", docID)
	if err != nil {
		t.Fatalf("Read after restart failed: %v", err)
	}

	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch after restart: got %s, want %s", got, payload)
	}
}

// TestWriteOrdering_UncommittedSkipped constructs a WAL that has data-bearing
// records but no corresponding OpCommit marker and verifies that recovery
// skips these uncommitted changes.
func TestWriteOrdering_UncommittedSkipped(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-uncommitted")
	defer cleanup()

	// Close the DB so that we can safely rewrite the WAL file.
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walPath := filepath.Join(walDir, "writeorder-uncommitted.wal")

	// Overwrite WAL with a single OpCreate record for txID=1 but no OpCommit.
	// On replay, this transaction should be treated as uncommitted and ignored.
	log := logger.Default()
	writer := wal.NewWriter(walPath, 1, 0, true, log)
	if err := writer.Open(); err != nil {
		t.Fatalf("Failed to open WAL writer: %v", err)
	}
	defer writer.Close()

	docID := uint64(42)
	payload := []byte(`{"data":"uncommitted"}`)
	if err := writer.Write(1, 1, "_default", docID, types.OpCreate, payload); err != nil {
		t.Fatalf("Failed to write uncommitted WAL record: %v", err)
	}

	// Reopen DB to trigger WAL replay.
	db2 := reopenDB(t, "writeorder-uncommitted", dataDir, walDir)
	defer db2.Close()

	_, err := db2.Read("_default", docID)
	if err == nil {
		t.Fatalf("Expected document to be absent after replay of uncommitted tx, but read succeeded")
	}
}

// TestWriteOrdering_UpdateCommit verifies that Update operations with commit
// markers survive restart correctly.
func TestWriteOrdering_UpdateCommit(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-update")
	defer cleanup()

	docID := uint64(1)
	initialPayload := []byte(`{"data":"initial"}`)
	updatedPayload := []byte(`{"data":"updated"}`)

	// Create initial document
	if err := db.Create("_default", docID, initialPayload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update document
	if err := db.Update("_default", docID, updatedPayload); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Simulate restart
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db2 := reopenDB(t, "writeorder-update", dataDir, walDir)
	defer db2.Close()

	got, err := db2.Read("_default", docID)
	if err != nil {
		t.Fatalf("Read after restart failed: %v", err)
	}

	if string(got) != string(updatedPayload) {
		t.Fatalf("Payload mismatch after restart: got %s, want %s", got, updatedPayload)
	}
}

// TestWriteOrdering_DeleteCommit verifies that Delete operations with commit
// markers survive restart correctly.
func TestWriteOrdering_DeleteCommit(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-delete")
	defer cleanup()

	docID := uint64(1)
	payload := []byte(`{"data":"to be deleted"}`)

	// Create document
	if err := db.Create("_default", docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete document
	if err := db.Delete("_default", docID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Simulate restart
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db2 := reopenDB(t, "writeorder-delete", dataDir, walDir)
	defer db2.Close()

	_, err := db2.Read("_default", docID)
	if err == nil {
		t.Fatalf("Expected document to be deleted after restart, but read succeeded")
	}
	if err != docdb.ErrDocNotFound {
		t.Fatalf("Expected ErrDocNotFound, got: %v", err)
	}
}

// TestWriteOrdering_CrashAfterCommitMarker verifies that if a crash occurs
// after the commit marker is written but before index update, recovery
// correctly applies the committed transaction.
func TestWriteOrdering_CrashAfterCommitMarker(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-crash-after")
	defer cleanup()

	docID := uint64(1)
	payload := []byte(`{"data":"committed"}`)

	// Create document (this writes WAL record + commit marker)
	if err := db.Create("_default", docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close DB to simulate crash after commit marker
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen - recovery should find commit marker and index the document
	db2 := reopenDB(t, "writeorder-crash-after", dataDir, walDir)
	defer db2.Close()

	got, err := db2.Read("_default", docID)
	if err != nil {
		t.Fatalf("Read after crash recovery failed: %v", err)
	}

	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch after crash recovery: got %s, want %s", got, payload)
	}
}

// TestWriteOrdering_MultipleTransactions tests recovery with multiple
// transactions, some committed and some uncommitted.
func TestWriteOrdering_MultipleTransactions(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-multi")
	defer cleanup()

	// Close DB so we can manually construct WAL with mixed transactions.
	// v0.4: partition WAL lives under walDir/dbName/p0.wal; use v0.4 format (EncodeRecordV4).
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	partitionWalDir := filepath.Join(walDir, "writeorder-multi")
	if err := os.MkdirAll(partitionWalDir, 0755); err != nil {
		t.Fatalf("Failed to create partition WAL dir: %v", err)
	}
	walPath := filepath.Join(partitionWalDir, "p0.wal")
	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to create partition WAL file: %v", err)
	}
	defer f.Close()

	writeV4 := func(lsn, txID, dbID uint64, collection string, docID uint64, opType types.OperationType, payload []byte) {
		rec, err := wal.EncodeRecordV4(lsn, txID, dbID, collection, docID, opType, payload)
		if err != nil {
			t.Fatalf("EncodeRecordV4: %v", err)
		}
		if _, err := f.Write(rec); err != nil {
			t.Fatalf("Write WAL record: %v", err)
		}
	}

	doc1 := uint64(1)
	payload1 := []byte(`{"data":"committed1"}`)
	doc2 := uint64(2)
	payload2 := []byte(`{"data":"uncommitted2"}`)
	doc3 := uint64(3)
	payload3 := []byte(`{"data":"committed3"}`)

	// Transaction 1: committed (has commit marker)
	writeV4(1, 1, 1, "_default", doc1, types.OpCreate, payload1)
	writeV4(2, 1, 1, "", 0, types.OpCommit, nil)
	// Transaction 2: uncommitted (no commit marker)
	writeV4(3, 2, 1, "_default", doc2, types.OpCreate, payload2)
	// Transaction 3: committed (has commit marker)
	writeV4(4, 3, 1, "_default", doc3, types.OpCreate, payload3)
	writeV4(5, 3, 1, "", 0, types.OpCommit, nil)

	if err := f.Sync(); err != nil {
		t.Fatalf("Sync WAL: %v", err)
	}
	f.Close()

	// Reopen DB to trigger recovery
	db2 := reopenDB(t, "writeorder-multi", dataDir, walDir)
	defer db2.Close()

	// Document 1 should exist (committed)
	got1, err := db2.Read("_default", doc1)
	if err != nil {
		t.Fatalf("Read doc1 failed: %v", err)
	}
	if string(got1) != string(payload1) {
		t.Fatalf("Doc1 payload mismatch: got %s, want %s", got1, payload1)
	}

	// Document 2 should NOT exist (uncommitted)
	_, err = db2.Read("_default", doc2)
	if err == nil {
		t.Fatalf("Expected doc2 to be absent (uncommitted), but read succeeded")
	}

	// Document 3 should exist (committed)
	got3, err := db2.Read("_default", doc3)
	if err != nil {
		t.Fatalf("Read doc3 failed: %v", err)
	}
	if string(got3) != string(payload3) {
		t.Fatalf("Doc3 payload mismatch: got %s, want %s", got3, payload3)
	}
}

// TestWriteOrdering_PartialTransaction tests recovery when WAL has multiple
// records for a transaction but no commit marker.
func TestWriteOrdering_PartialTransaction(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-partial")
	defer cleanup()

	// Close DB so we can manually construct WAL
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walPath := filepath.Join(walDir, "writeorder-partial.wal")
	log := logger.Default()
	writer := wal.NewWriter(walPath, 1, 0, true, log)
	if err := writer.Open(); err != nil {
		t.Fatalf("Failed to open WAL writer: %v", err)
	}
	defer writer.Close()

	// Write multiple operations for transaction 1, but no commit marker
	txID := uint64(1)
	doc1 := uint64(1)
	doc2 := uint64(2)
	doc3 := uint64(3)

	payload1 := []byte(`{"data":"partial1"}`)
	payload2 := []byte(`{"data":"partial2"}`)
	payload3 := []byte(`{"data":"partial3"}`)

	if err := writer.Write(txID, 1, "_default", doc1, types.OpCreate, payload1); err != nil {
		t.Fatalf("Failed to write op1: %v", err)
	}
	if err := writer.Write(txID, 1, "_default", doc2, types.OpCreate, payload2); err != nil {
		t.Fatalf("Failed to write op2: %v", err)
	}
	if err := writer.Write(txID, 1, "_default", doc3, types.OpCreate, payload3); err != nil {
		t.Fatalf("Failed to write op3: %v", err)
	}
	// Intentionally no commit marker

	// Reopen DB - none of these documents should be indexed
	db2 := reopenDB(t, "writeorder-partial", dataDir, walDir)
	defer db2.Close()

	_, err := db2.Read("_default", doc1)
	if err == nil {
		t.Fatalf("Expected doc1 to be absent (partial tx), but read succeeded")
	}

	_, err = db2.Read("_default", doc2)
	if err == nil {
		t.Fatalf("Expected doc2 to be absent (partial tx), but read succeeded")
	}

	_, err = db2.Read("_default", doc3)
	if err == nil {
		t.Fatalf("Expected doc3 to be absent (partial tx), but read succeeded")
	}
}

// TestWriteOrdering_SequentialOperations tests multiple sequential operations
// to ensure each gets its own commit marker and survives restart.
func TestWriteOrdering_SequentialOperations(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-sequential")
	defer cleanup()

	// Create multiple documents sequentially
	numDocs := 10
	documents := make(map[uint64][]byte)

	for i := 1; i <= numDocs; i++ {
		docID := uint64(i)
		payload := []byte(fmt.Sprintf(`{"data":"doc%d"}`, i))
		documents[docID] = payload

		if err := db.Create("_default", docID, payload); err != nil {
			t.Fatalf("Create doc %d failed: %v", docID, err)
		}
	}

	// Simulate restart
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db2 := reopenDB(t, "writeorder-sequential", dataDir, walDir)
	defer db2.Close()

	// Verify all documents survived restart
	for docID, expectedPayload := range documents {
		got, err := db2.Read("_default", docID)
		if err != nil {
			t.Fatalf("Read doc %d after restart failed: %v", docID, err)
		}
		if string(got) != string(expectedPayload) {
			t.Fatalf("Doc %d payload mismatch: got %s, want %s", docID, got, expectedPayload)
		}
	}

	// Verify index size matches
	if db2.IndexSize() != numDocs {
		t.Fatalf("Index size mismatch: got %d, want %d", db2.IndexSize(), numDocs)
	}
}

// TestWriteOrdering_MixedOperations tests a mix of Create, Update, and Delete
// operations to ensure all operation types work correctly with commit markers.
func TestWriteOrdering_MixedOperations(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-mixed")
	defer cleanup()

	doc1 := uint64(1)
	doc2 := uint64(2)
	doc3 := uint64(3)

	// Create all three documents
	payload1a := []byte(`{"data":"doc1"}`)
	payload2a := []byte(`{"data":"doc2"}`)
	payload3a := []byte(`{"data":"doc3"}`)

	if err := db.Create("_default", doc1, payload1a); err != nil {
		t.Fatalf("Create doc1 failed: %v", err)
	}
	if err := db.Create("_default", doc2, payload2a); err != nil {
		t.Fatalf("Create doc2 failed: %v", err)
	}
	if err := db.Create("_default", doc3, payload3a); err != nil {
		t.Fatalf("Create doc3 failed: %v", err)
	}

	// Update doc1
	payload1b := []byte(`{"data":"doc1-updated"}`)
	if err := db.Update("_default", doc1, payload1b); err != nil {
		t.Fatalf("Update doc1 failed: %v", err)
	}

	// Delete doc2
	if err := db.Delete("_default", doc2); err != nil {
		t.Fatalf("Delete doc2 failed: %v", err)
	}

	// Simulate restart
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db2 := reopenDB(t, "writeorder-mixed", dataDir, walDir)
	defer db2.Close()

	// Verify doc1 was updated
	got1, err := db2.Read("_default", doc1)
	if err != nil {
		t.Fatalf("Read doc1 failed: %v", err)
	}
	if string(got1) != string(payload1b) {
		t.Fatalf("Doc1 payload mismatch: got %s, want %s", got1, payload1b)
	}

	// Verify doc2 was deleted
	_, err = db2.Read("_default", doc2)
	if err == nil {
		t.Fatalf("Expected doc2 to be deleted, but read succeeded")
	}

	// Verify doc3 still exists unchanged
	got3, err := db2.Read("_default", doc3)
	if err != nil {
		t.Fatalf("Read doc3 failed: %v", err)
	}
	if string(got3) != string(payload3a) {
		t.Fatalf("Doc3 payload mismatch: got %s, want %s", got3, payload3a)
	}
}

// TestWriteOrdering_UncommittedUpdate tests that an uncommitted update
// operation is not applied during recovery.
func TestWriteOrdering_UncommittedUpdate(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-uncommitted-update")
	defer cleanup()

	docID := uint64(1)
	initialPayload := []byte(`{"data":"initial"}`)

	// Create initial document
	if err := db.Create("_default", docID, initialPayload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close DB to manually construct WAL
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walPath := filepath.Join(walDir, "writeorder-uncommitted-update.wal")
	log := logger.Default()
	writer := wal.NewWriter(walPath, 1, 0, true, log)
	if err := writer.Open(); err != nil {
		t.Fatalf("Failed to open WAL writer: %v", err)
	}
	defer writer.Close()

	// Write an uncommitted update (no commit marker)
	updatedPayload := []byte(`{"data":"uncommitted-update"}`)
	if err := writer.Write(2, 1, "_default", docID, types.OpUpdate, updatedPayload); err != nil {
		t.Fatalf("Failed to write uncommitted update: %v", err)
	}

	// Reopen DB - update should be ignored, original value should remain
	db2 := reopenDB(t, "writeorder-uncommitted-update", dataDir, walDir)
	defer db2.Close()

	got, err := db2.Read("_default", docID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(got) != string(initialPayload) {
		t.Fatalf("Expected original payload after uncommitted update: got %s, want %s", got, initialPayload)
	}
}

// TestWriteOrdering_UncommittedDelete tests that an uncommitted delete
// operation is not applied during recovery.
func TestWriteOrdering_UncommittedDelete(t *testing.T) {
	db, dataDir, walDir, cleanup := setupSingleDB(t, "writeorder-uncommitted-delete")
	defer cleanup()

	docID := uint64(1)
	payload := []byte(`{"data":"to survive"}`)

	// Create document
	if err := db.Create("_default", docID, payload); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close DB to manually construct WAL
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walPath := filepath.Join(walDir, "writeorder-uncommitted-delete.wal")
	log := logger.Default()
	writer := wal.NewWriter(walPath, 1, 0, true, log)
	if err := writer.Open(); err != nil {
		t.Fatalf("Failed to open WAL writer: %v", err)
	}
	defer writer.Close()

	// Write an uncommitted delete (no commit marker)
	if err := writer.Write(2, 1, "_default", docID, types.OpDelete, nil); err != nil {
		t.Fatalf("Failed to write uncommitted delete: %v", err)
	}

	// Reopen DB - delete should be ignored, document should still exist
	db2 := reopenDB(t, "writeorder-uncommitted-delete", dataDir, walDir)
	defer db2.Close()

	got, err := db2.Read("_default", docID)
	if err != nil {
		t.Fatalf("Expected document to survive uncommitted delete, but read failed: %v", err)
	}

	if string(got) != string(payload) {
		t.Fatalf("Payload mismatch: got %s, want %s", got, payload)
	}
}
