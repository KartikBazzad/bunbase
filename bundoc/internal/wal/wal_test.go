package wal

import (
	"testing"
	"time"
)

func TestSegmentWriteRead(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create segment
	segment, err := NewSegment(tmpdir, 0, LSN(1))
	if err != nil {
		t.Fatalf("Failed to create segment: %v", err)
	}
	defer segment.Close()

	// Write some records
	records := []*Record{
		{
			LSN:       LSN(1),
			TxnID:     100,
			Type:      RecordTypeInsert,
			Key:       []byte("key1"),
			Value:     []byte("value1"),
			Timestamp: time.Now().UnixNano(),
		},
		{
			LSN:       LSN(2),
			TxnID:     100,
			Type:      RecordTypeCommit,
			Key:       []byte{},
			Value:     []byte{},
			PrevLSN:   LSN(1),
			Timestamp: time.Now().UnixNano(),
		},
	}

	for _, record := range records {
		if err := segment.Write(record); err != nil {
			t.Fatalf("Failed to write record: %v", err)
		}
	}

	// Sync to disk
	if err := segment.Sync(); err != nil {
		t.Fatalf("Failed to sync segment: %v", err)
	}

	// Read records back
	readRecords, err := segment.ReadRecords()
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	// Verify count
	if len(readRecords) != len(records) {
		t.Errorf("Expected %d records, got %d", len(records), len(readRecords))
	}

	// Verify first record
	if len(readRecords) > 0 {
		if readRecords[0].LSN != records[0].LSN {
			t.Errorf("LSN mismatch: expected %d, got %d", records[0].LSN, readRecords[0].LSN)
		}
		if readRecords[0].TxnID != records[0].TxnID {
			t.Errorf("TxnID mismatch: expected %d, got %d", records[0].TxnID, readRecords[0].TxnID)
		}
	}
}

func TestWALAppend(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create WAL
	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append records
	record1 := &Record{
		TxnID:     200,
		Type:      RecordTypeInsert,
		Key:       []byte("test_key"),
		Value:     []byte("test_value"),
		Timestamp: time.Now().UnixNano(),
	}

	lsn1, err := wal.Append(record1)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	if lsn1 == 0 {
		t.Error("Expected non-zero LSN")
	}

	// Append another record
	record2 := &Record{
		TxnID:     200,
		Type:      RecordTypeCommit,
		Key:       []byte{},
		Value:     []byte{},
		PrevLSN:   lsn1,
		Timestamp: time.Now().UnixNano(),
	}

	lsn2, err := wal.Append(record2)
	if err != nil {
		t.Fatalf("Failed to append second record: %v", err)
	}

	if lsn2 <= lsn1 {
		t.Errorf("Expected LSN2 > LSN1, got LSN1=%d, LSN2=%d", lsn1, lsn2)
	}

	// Sync
	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync WAL: %v", err)
	}

	// Verify current LSN
	currentLSN := wal.GetCurrentLSN()
	if currentLSN < lsn2 {
		t.Errorf("Expected current LSN >= %d, got %d", lsn2, currentLSN)
	}
}

func TestWALRecovery(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create WAL and write records
	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Write several records
	expectedRecords := 10
	for i := 0; i < expectedRecords; i++ {
		record := &Record{
			TxnID:     uint64(i),
			Type:      RecordTypeInsert,
			Key:       []byte("key"),
			Value:     []byte("value"),
			Timestamp: time.Now().UnixNano(),
		}
		if _, err := wal.Append(record); err != nil {
			t.Fatalf("Failed to append record %d: %v", i, err)
		}
	}

	wal.Sync()
	wal.Close()

	// Reopen WAL and read all records
	wal2, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	records, err := wal2.ReadAllRecords()
	if err != nil {
		t.Fatalf("Failed to read all records: %v", err)
	}

	if len(records) != expectedRecords {
		t.Errorf("Expected %d records, got %d", expectedRecords, len(records))
	}
}

func TestSegmentRotation(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create segment with small max size
	segment, err := NewSegment(tmpdir, 0, LSN(1))
	if err != nil {
		t.Fatalf("Failed to create segment: %v", err)
	}

	// Override max size to force rotation
	segment.maxSize = 1024 // 1KB

	// Write records until full
	recordCount := 0
	for !segment.IsFull() && recordCount < 100 {
		record := &Record{
			LSN:       LSN(recordCount + 1),
			TxnID:     uint64(recordCount),
			Type:      RecordTypeInsert,
			Key:       []byte("key"),
			Value:     make([]byte, 100), // 100 bytes
			Timestamp: time.Now().UnixNano(),
		}
		if err := segment.Write(record); err != nil {
			t.Fatalf("Failed to write record: %v", err)
		}
		recordCount++
	}

	if !segment.IsFull() && recordCount >= 100 {
		t.Error("Expected segment to be full before reaching 100 records")
	}

	segment.Close()
}

func TestWALConcurrentWrites(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create WAL
	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Concurrent writes
	numWriters := 10
	recordsPerWriter := 10
	done := make(chan bool, numWriters)

	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			for j := 0; j < recordsPerWriter; j++ {
				record := &Record{
					TxnID:     uint64(writerID*1000 + j),
					Type:      RecordTypeInsert,
					Key:       []byte("key"),
					Value:     []byte("value"),
					Timestamp: time.Now().UnixNano(),
				}
				if _, err := wal.Append(record); err != nil {
					t.Errorf("Writer %d failed to append: %v", writerID, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all writers
	for i := 0; i < numWriters; i++ {
		<-done
	}

	// Verify all records were written
	wal.Sync()
	records, err := wal.ReadAllRecords()
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	expectedTotal := numWriters * recordsPerWriter
	if len(records) != expectedTotal {
		t.Errorf("Expected %d records, got %d", expectedTotal, len(records))
	}
}
