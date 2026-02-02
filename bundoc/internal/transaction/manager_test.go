package transaction

import (
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/internal/wal"
)

func TestTransactionBeginCommit(t *testing.T) {
	// Setup
	tmpdir := t.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, err := wal.NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	// Begin transaction
	txn, err := tm.Begin(mvcc.ReadCommitted)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if txn.ID == 0 {
		t.Error("Transaction ID should be non-zero")
	}
	if txn.Status != StatusActive {
		t.Error("New transaction should be active")
	}

	// Write some data
	err = tm.Write(txn, "key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	err = tm.Write(txn, "key2", []byte("value2"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Verify write set
	if len(txn.WriteSet) != 2 {
		t.Errorf("Expected 2 writes, got %d", len(txn.WriteSet))
	}

	// Commit
	err = tm.Commit(txn)
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	if txn.Status != StatusCommitted {
		t.Error("Transaction should be committed")
	}

	// Verify transaction is no longer active
	count := tm.GetActiveTransactionCount()
	if count != 0 {
		t.Errorf("Expected 0 active transactions, got %d", count)
	}
}

func TestTransactionRollback(t *testing.T) {
	// Setup
	tmpdir := t.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, err := wal.NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	// Begin transaction
	txn, err := tm.Begin(mvcc.ReadCommitted)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Write data
	err = tm.Write(txn, "key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Rollback
	err = tm.Rollback(txn)
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	if txn.Status != StatusAborted {
		t.Error("Transaction should be aborted")
	}
}

func TestConcurrentTransactions(t *testing.T) {
	// Setup
	tmpdir := t.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, err := wal.NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	// Start multiple concurrent transactions
	numTxns := 10
	done := make(chan bool, numTxns)
	errors := make(chan error, numTxns)

	for i := 0; i < numTxns; i++ {
		go func(id int) {
			txn, err := tm.Begin(mvcc.ReadCommitted)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			// Write data
			key := string(rune('a' + id))
			value := []byte("value")
			err = tm.Write(txn, key, value)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			// Simulate some work
			time.Sleep(time.Millisecond * 10)

			// Commit
			err = tm.Commit(txn)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all transactions
	successCount := 0
	for i := 0; i < numTxns; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case err := <-errors:
			t.Errorf("Transaction error: %v", err)
		case <-time.After(time.Second * 5):
			t.Fatal("Timeout waiting for transactions")
		}
	}

	if successCount != numTxns {
		t.Errorf("Expected %d successful transactions, got %d", numTxns, successCount)
	}

	// All transactions should be completed
	count := tm.GetActiveTransactionCount()
	if count != 0 {
		t.Errorf("Expected 0 active transactions, got %d", count)
	}
}

func TestIsolationLevels(t *testing.T) {
	tmpdir := t.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, err := wal.NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	levels := []mvcc.IsolationLevel{
		mvcc.ReadUncommitted,
		mvcc.ReadCommitted,
		mvcc.RepeatableRead,
		mvcc.Serializable,
	}

	for _, level := range levels {
		txn, err := tm.Begin(level)
		if err != nil {
			t.Errorf("Failed to begin transaction with level %d: %v", level, err)
			continue
		}

		if txn.IsolationLevel != level {
			t.Errorf("Expected isolation level %d, got %d", level, txn.IsolationLevel)
		}

		tm.Rollback(txn)
	}
}

func TestReadOwnWrites(t *testing.T) {
	tmpdir := t.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, err := wal.NewWAL(tmpdir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	txn, err := tm.Begin(mvcc.ReadCommitted)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Write a value
	key := "test_key"
	value := []byte("test_value")
	err = tm.Write(txn, key, value)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Read should return the written value
	readValue, err := tm.Read(txn, key)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if string(readValue) != string(value) {
		t.Errorf("Expected to read %s, got %s", value, readValue)
	}

	tm.Rollback(txn)
}

func BenchmarkTransactionCommit(b *testing.B) {
	tmpdir := b.TempDir()
	vm := mvcc.NewVersionManager()
	sm := mvcc.NewSnapshotManager(vm)
	walWriter, _ := wal.NewWAL(tmpdir)
	defer walWriter.Close()

	tm := NewTransactionManager(sm, walWriter)
	defer tm.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := tm.Begin(mvcc.ReadCommitted)
		tm.Write(txn, "key", []byte("value"))
		tm.Commit(txn)
	}
}
