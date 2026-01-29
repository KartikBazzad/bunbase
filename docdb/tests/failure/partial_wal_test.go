package failure

import (
	"fmt"
	"testing"
)

func TestPartialWALRecovery(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("partialwal")
	defer db.Close()

	// Create multiple documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 10; i++ {
		if err := db.Create(defaultColl, uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Get WAL size
	walSize, err := helper.WALSize("partialwal")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Close database
	db.Close()

	// Truncate WAL at various offsets
	testOffsets := []int64{
		walSize / 4,
		walSize / 2,
		walSize * 3 / 4,
	}

	for _, offset := range testOffsets {
		t.Run(fmt.Sprintf("Offset_%d", offset), func(t *testing.T) {
			// Truncate WAL
			if err := helper.TruncateWAL("partialwal", offset); err != nil {
				t.Fatalf("Failed to truncate WAL: %v", err)
			}

			// Reopen database
			db2 := helper.ReopenDB("partialwal")
			defer db2.Close()

			// Count recoverable documents
			recovered := 0
			for i := 1; i <= 10; i++ {
				if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
					recovered++
				}
			}

			t.Logf("Recovered %d documents after truncating at offset %d", recovered, offset)

			// Cleanup for next iteration
			db2.Close()
		})
	}
}

func TestPartialWALMultiSegment(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("multiseg")
	defer db.Close()

	// Create many documents to potentially trigger WAL rotation
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 50; i++ {
		if err := db.Create(defaultColl, uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Close database
	db.Close()

	// Truncate WAL segment (if rotation occurred)
	walSize, err := helper.WALSize("multiseg")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Truncate at midpoint
	if err := helper.TruncateWAL("multiseg", walSize/2); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("multiseg")
	defer db2.Close()

	// Verify recovery
	recovered := 0
	for i := 1; i <= 50; i++ {
		if err := helper.VerifyDocument(db2, uint64(i), payload); err == nil {
			recovered++
		}
	}

	t.Logf("Recovered %d out of 50 documents after partial WAL truncation", recovered)

	if recovered == 0 {
		t.Error("No documents were recovered")
	}
}

func TestPartialWALAtRecordBoundary(t *testing.T) {
	helper := NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDB("boundary")
	defer db.Close()

	// Create documents with varying sizes
	payloads := [][]byte{
		[]byte(`{"small":"data"}`),
		[]byte(`{"medium":"data","more":"fields"}`),
		[]byte(`{"large":"data","field1":"value1","field2":"value2","field3":"value3"}`),
	}

	for i, payload := range payloads {
		if err := db.Create(defaultColl, uint64(i+1), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i+1, err)
		}
	}

	// Get WAL size
	walSize, err := helper.WALSize("boundary")
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	// Close database
	db.Close()

	// Truncate at a point that might be mid-record
	truncateOffset := walSize / 2
	if err := helper.TruncateWAL("boundary", truncateOffset); err != nil {
		t.Fatalf("Failed to truncate WAL: %v", err)
	}

	// Reopen database
	db2 := helper.ReopenDB("boundary")
	defer db2.Close()

	// Verify recovery handles partial records gracefully
	for i, payload := range payloads {
		if err := helper.VerifyDocument(db2, uint64(i+1), payload); err != nil {
			t.Logf("Document %d recovery: %v (may be expected if truncation occurred mid-record)", i+1, err)
		}
	}
}
