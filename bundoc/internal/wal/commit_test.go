package wal

import (
	"sync"
	"testing"
	"time"
)

func TestGroupCommitter(t *testing.T) {
	// Create temp directory
	tmpdir := t.TempDir()

	// Create WAL
	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Create group committer
	gc := NewGroupCommitter(wal)
	defer gc.Stop()

	// Append and commit records
	record := &Record{
		TxnID:     100,
		Type:      RecordTypeInsert,
		Key:       []byte("key"),
		Value:     []byte("value"),
		Timestamp: time.Now().UnixNano(),
	}

	lsn, err := wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Commit via group committer
	if err := gc.Commit(lsn); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify record is persisted
	records, err := wal.ReadAllRecords()
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	if len(records) == 0 {
		t.Error("Expected at least one record")
	}
}

func TestGroupCommitterConcurrent(t *testing.T) {
	tmpdir := t.TempDir()

	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	gc := NewGroupCommitter(wal)
	defer gc.Stop()

	// Concurrent commits
	numCommitters := 50
	commitsPerGoroutine := 20
	var wg sync.WaitGroup

	for i := 0; i < numCommitters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < commitsPerGoroutine; j++ {
				record := &Record{
					TxnID:     uint64(id*1000 + j),
					Type:      RecordTypeInsert,
					Key:       []byte("key"),
					Value:     []byte("value"),
					Timestamp: time.Now().UnixNano(),
				}
				lsn, err := wal.Append(record)
				if err != nil {
					t.Errorf("Failed to append: %v", err)
					return
				}
				if err := gc.Commit(lsn); err != nil {
					t.Errorf("Failed to commit: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all records were written
	records, err := wal.ReadAllRecords()
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	expected := numCommitters * commitsPerGoroutine
	if len(records) != expected {
		t.Errorf("Expected %d records, got %d", expected, len(records))
	}
}

func TestSharedFlusher(t *testing.T) {
	tmpdir := t.TempDir()

	wal1, err := NewWAL(tmpdir + "/wal1")
	if err != nil {
		t.Fatalf("Failed to create WAL1: %v", err)
	}
	defer wal1.Close()

	wal2, err := NewWAL(tmpdir + "/wal2")
	if err != nil {
		t.Fatalf("Failed to create WAL2: %v", err)
	}
	defer wal2.Close()

	flusher := GetSharedFlusher()

	// Append to both WALs
	record1 := &Record{
		TxnID:     1,
		Type:      RecordTypeInsert,
		Key:       []byte("key1"),
		Value:     []byte("value1"),
		Timestamp: time.Now().UnixNano(),
	}
	wal1.Append(record1)

	record2 := &Record{
		TxnID:     2,
		Type:      RecordTypeInsert,
		Key:       []byte("key2"),
		Value:     []byte("value2"),
		Timestamp: time.Now().UnixNano(),
	}
	wal2.Append(record2)

	// Flush both via shared flusher
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := flusher.Flush(wal1); err != nil {
			t.Errorf("Failed to flush WAL1: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := flusher.Flush(wal2); err != nil {
			t.Errorf("Failed to flush WAL2: %v", err)
		}
	}()

	wg.Wait()

	// Verify both WALs have records
	records1, _ := wal1.ReadAllRecords()
	records2, _ := wal2.ReadAllRecords()

	if len(records1) == 0 || len(records2) == 0 {
		t.Error("Expected records in both WALs")
	}
}

func TestRecovery(t *testing.T) {
	tmpdir := t.TempDir()

	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Write committed transaction
	r1 := &Record{TxnID: 1, Type: RecordTypeInsert, Key: []byte("k1"), Value: []byte("v1"), Timestamp: time.Now().UnixNano()}
	r2 := &Record{TxnID: 1, Type: RecordTypeCommit, Timestamp: time.Now().UnixNano()}
	wal.Append(r1)
	wal.Append(r2)

	// Write aborted transaction
	r3 := &Record{TxnID: 2, Type: RecordTypeInsert, Key: []byte("k2"), Value: []byte("v2"), Timestamp: time.Now().UnixNano()}
	r4 := &Record{TxnID: 2, Type: RecordTypeAbort, Timestamp: time.Now().UnixNano()}
	wal.Append(r3)
	wal.Append(r4)

	// Write incomplete transaction (no commit/abort)
	r5 := &Record{TxnID: 3, Type: RecordTypeInsert, Key: []byte("k3"), Value: []byte("v3"), Timestamp: time.Now().UnixNano()}
	wal.Append(r5)

	wal.Sync()
	wal.Close()

	// Recover
	wal2, _ := NewWAL(tmpdir)
	defer wal2.Close()

	recovery := NewRecovery(wal2)
	validRecords, err := recovery.Recover()
	if err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}

	// Should only have records from transaction 1 (committed)
	if len(validRecords) != 1 {
		t.Errorf("Expected 1 valid record, got %d", len(validRecords))
	}

	if len(validRecords) > 0 && validRecords[0].TxnID != 1 {
		t.Errorf("Expected TxnID 1, got %d", validRecords[0].TxnID)
	}
}

func TestRecoveryIntegrity(t *testing.T) {
	tmpdir := t.TempDir()

	wal, err := NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Write records with monotonic LSNs
	for i := 0; i < 10; i++ {
		record := &Record{
			TxnID:     uint64(i),
			Type:      RecordTypeInsert,
			Key:       []byte("key"),
			Value:     []byte("value"),
			Timestamp: time.Now().UnixNano(),
		}
		wal.Append(record)
	}

	wal.Sync()

	recovery := NewRecovery(wal)
	if err := recovery.VerifyIntegrity(); err != nil {
		t.Errorf("WAL integrity check failed: %v", err)
	}

	wal.Close()
}

func BenchmarkGroupCommit(b *testing.B) {
	tmpdir := b.TempDir()

	wal, _ := NewWAL(tmpdir)
	defer wal.Close()

	gc := NewGroupCommitter(wal)
	defer gc.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &Record{
			TxnID:     uint64(i),
			Type:      RecordTypeInsert,
			Key:       []byte("key"),
			Value:     []byte("value"),
			Timestamp: time.Now().UnixNano(),
		}
		lsn, _ := wal.Append(record)
		gc.Commit(lsn)
	}
}
