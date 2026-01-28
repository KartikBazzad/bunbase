package failure

import (
	"testing"
)

func TestCorruptWALRecord(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("corruptwal")
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 5; i++ {
		if err := db.Create(uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Get WAL size
	walSize, err := helper.WALSize("corruptwal")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Close database
	db.Close()

	// Corrupt WAL at midpoint
	corruptOffset := walSize / 2
	if err := helper.CorruptWAL("corruptwal", corruptOffset, 100); err != nil {
		t.Fatalf("Failed to corrupt WAL: %v", err)
	}

	// Reopen database - should handle corruption gracefully
	db2 := helper.ReopenDB("corruptwal")
	defer db2.Close()

	// Some documents should still be readable
	readable := 0
	for i := 1; i <= 5; i++ {
		if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
			readable++
		}
	}

	t.Logf("Readable documents after WAL corruption: %d out of 5", readable)

	if readable == 0 {
		t.Error("No documents were readable after corruption")
	}
}

func TestCorruptDataFileRecord(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("corruptdata")
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 5; i++ {
		if err := db.Create(uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Get data file size
	dataFileSize, err := helper.DataFileSize("corruptdata")
	if err != nil {
		t.Fatalf("Failed to get data file size: %v", err)
	}

	// Close database
	db.Close()

	// Corrupt data file at midpoint
	corruptOffset := dataFileSize / 2
	if err := helper.CorruptDataFile("corruptdata", corruptOffset, 50); err != nil {
		t.Fatalf("Failed to corrupt data file: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("corruptdata")
	defer db2.Close()

	// Try to read documents - corruption should be detected
	readable := 0
	corrupted := 0
	for i := 1; i <= 5; i++ {
		if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
			readable++
		} else {
			corrupted++
		}
	}

	t.Logf("Readable: %d, Corrupted: %d", readable, corrupted)

	// At least some documents should be readable (those before corruption point)
	if readable == 0 && dataFileSize > 0 {
		t.Error("No documents were readable after data file corruption")
	}
}

func TestCorruptRecordIsolation(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("isolate")
	defer db.Close()

	// Create documents
	payloads := [][]byte{
		[]byte(`{"doc":"1"}`),
		[]byte(`{"doc":"2"}`),
		[]byte(`{"doc":"3"}`),
	}

	for i, payload := range payloads {
		if err := db.Create(uint64(i+1), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i+1, err)
		}
	}

	// Close database
	db.Close()

	// Corrupt only the middle document's data
	dataFileSize, err := helper.DataFileSize("isolate")
	if err != nil {
		t.Fatalf("Failed to get data file size: %v", err)
	}

	// Corrupt at 1/3 point (likely where second document is)
	corruptOffset := dataFileSize / 3
	if err := helper.CorruptDataFile("isolate", corruptOffset, 20); err != nil {
		t.Fatalf("Failed to corrupt data file: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("isolate")
	defer db2.Close()

	// First document should be readable
	if err := helper.VerifyDocument(db2, 1, payloads[0]); err != nil {
		t.Errorf("Document 1 should be readable: %v", err)
	}

	// Last document might be readable depending on corruption location
	if err := helper.VerifyDocument(db2, 3, payloads[2]); err != nil {
		t.Logf("Document 3 recovery: %v", err)
	}
}
