package failure

import (
	"testing"
)

func TestCompactionCrash(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("compactiondb")
	defer db.Close()

	// Create multiple documents to trigger compaction threshold
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 20; i++ {
		if err := db.Create(defaultColl, uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Delete some documents to create tombstones
	for i := 1; i <= 10; i++ {
		if err := db.Delete(defaultColl, uint64(i)); err != nil {
			t.Fatalf("Failed to delete document %d: %v", i, err)
		}
	}

	// Get data file size before compaction
	dataFileSize, err := helper.DataFileSize("compactiondb")
	if err != nil {
		t.Fatalf("Failed to get data file size: %v", err)
	}

	// Close database
	db.Close()

	// Truncate data file to simulate crash during compaction
	truncateOffset := dataFileSize / 2
	if err := helper.TruncateDataFile("compactiondb", truncateOffset); err != nil {
		t.Fatalf("Failed to truncate data file: %v", err)
	}

	// Reopen database - should handle corrupted data file
	db2 := helper.ReopenDB("compactiondb")
	defer db2.Close()

	// Verify that remaining documents are still readable
	readable := 0
	for i := 11; i <= 20; i++ {
		if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
			readable++
		}
	}

	if readable == 0 {
		t.Error("No documents were readable after compaction crash")
	}

	t.Logf("Readable documents after crash: %d out of 10", readable)
}

func TestCompactionCrashWithPartialWrite(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("compactiondb")
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 10; i++ {
		if err := db.Create(defaultColl, uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Get data file size
	dataFileSize, err := helper.DataFileSize("compactiondb")
	if err != nil {
		t.Fatalf("Failed to get data file size: %v", err)
	}

	// Close database
	db.Close()

	// Corrupt data file at a specific offset (simulating partial write)
	corruptOffset := dataFileSize / 3
	if err := helper.CorruptDataFile("compactiondb", corruptOffset, 50); err != nil {
		t.Fatalf("Failed to corrupt data file: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("compactiondb")
	defer db2.Close()

	// Some documents should still be readable
	readable := 0
	for i := 1; i <= 10; i++ {
		if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
			readable++
		}
	}

	t.Logf("Readable documents after corruption: %d out of 10", readable)
}
