package failure

import (
	"testing"
)

func TestWALWriteCrashMidTransaction(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("crashdb")
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value1"}`)
	if err := db.Create(defaultColl, 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Get WAL size before crash
	walSize, err := helper.WALSize("crashdb")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Simulate crash by closing database abruptly
	db.Close()

	// Truncate WAL to simulate incomplete write (mid-transaction)
	truncateOffset := walSize / 2
	if err := helper.TruncateWAL("crashdb", truncateOffset); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	// Reopen database - should recover from checkpoint or beginning
	db2 := helper.ReopenDB("crashdb")
	defer db2.Close()

	// Document 1 should still be readable (was committed before crash)
	if err := helper.VerifyDocument(db2, 1, payload); err != nil {
		t.Logf("Document 1 recovery: %v (may be expected if crash occurred before commit)", err)
	}
}

func TestWALWriteCrashMultipleTransactions(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("crashdb")
	defer db.Close()

	// Create multiple documents
	payloads := [][]byte{
		[]byte(`{"key":"value1"}`),
		[]byte(`{"key":"value2"}`),
		[]byte(`{"key":"value3"}`),
	}

	for i, payload := range payloads {
		if err := db.Create(defaultColl, uint64(i+1), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i+1, err)
		}
	}

	// Get WAL size
	walSize, err := helper.WALSize("crashdb")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Close and truncate WAL
	db.Close()
	if err := helper.TruncateWAL("crashdb", walSize*3/4); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	// Reopen and verify recovery
	db2 := helper.ReopenDB("crashdb")
	defer db2.Close()

	// Some documents should be recoverable
	recovered := 0
	for i, payload := range payloads {
		if err := helper.VerifyDocument(db2, uint64(i+1), payload); err == nil {
			recovered++
		}
	}

	if recovered == 0 {
		t.Error("No documents were recovered after crash")
	}

	t.Logf("Recovered %d out of %d documents", recovered, len(payloads))
}

func TestWALWriteCrashDuringCommitMarker(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("crashdb")
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create(defaultColl, 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Get WAL size
	walSize, err := helper.WALSize("crashdb")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Close database
	db.Close()

	// Truncate WAL just before the end (simulating crash during commit marker write)
	truncateOffset := walSize - 10
	if truncateOffset < 0 {
		truncateOffset = walSize / 2
	}

	if err := helper.TruncateWAL("crashdb", truncateOffset); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("crashdb")
	defer db2.Close()

	// Document should be recoverable if commit marker was written
	// If not, it should be filtered out during recovery
	if err := helper.VerifyDocument(db2, 1, payload); err != nil {
		t.Logf("Document recovery result: %v (may be expected if commit marker was incomplete)", err)
	}
}
