package mvcc

import (
	"bytes"
	"testing"
	"time"
)

func TestVersionManager(t *testing.T) {
	vm := NewVersionManager()

	// Test timestamp generation
	ts1 := vm.NewTimestamp()
	ts2 := vm.NewTimestamp()

	if ts2 <= ts1 {
		t.Errorf("Timestamps should be monotonically increasing: ts1=%d, ts2=%d", ts1, ts2)
	}

	// Test current timestamp
	current := vm.GetCurrentTimestamp()
	if current < ts2 {
		t.Errorf("Current timestamp should be >= last generated timestamp")
	}
}

func TestCreateVersion(t *testing.T) {
	vm := NewVersionManager()

	data := []byte("test data")
	txnID := uint64(100)

	version := vm.CreateVersion(data, txnID)

	if version == nil {
		t.Fatal("Expected version to be created")
	}
	if version.TxnID != txnID {
		t.Errorf("Expected TxnID %d, got %d", txnID, version.TxnID)
	}
	if !bytes.Equal(version.Data, data) {
		t.Errorf("Expected data %v, got %v", data, version.Data)
	}
	if version.Next != nil {
		t.Error("New version should have nil Next")
	}
}

func TestVersionChain(t *testing.T) {
	vm := NewVersionManager()

	// Create version chain
	v1 := vm.CreateVersion([]byte("v1"), 1)
	v2 := vm.CreateVersion([]byte("v2"), 2)
	v3 := vm.CreateVersion([]byte("v3"), 3)

	// Build chain: v3 -> v2 -> v1
	head := vm.AddVersion(nil, v1)
	head = vm.AddVersion(head, v2)
	head = vm.AddVersion(head, v3)

	// Verify chain
	if head != v3 {
		t.Error("Head should be v3")
	}
	if head.Next != v2 {
		t.Error("v3.Next should be v2")
	}
	if head.Next.Next != v1 {
		t.Error("v2.Next should be v1")
	}

	// Count versions
	count := CountVersions(head)
	if count != 3 {
		t.Errorf("Expected 3 versions, got %d", count)
	}
}

func TestFindVersion(t *testing.T) {
	vm := NewVersionManager()

	// Create versions with specific timestamps
	v1 := &Version{Timestamp: 100, Data: []byte("v1"), TxnID: 1}
	v2 := &Version{Timestamp: 200, Data: []byte("v2"), TxnID: 2}
	v3 := &Version{Timestamp: 300, Data: []byte("v3"), TxnID: 3}

	// Build chain
	head := vm.AddVersion(nil, v1)
	head = vm.AddVersion(head, v2)
	head = vm.AddVersion(head, v3)

	// Create a snapshot that sees everything (MaxTxnID high, no active/aborted)
	snapshot := &Snapshot{
		Timestamp:      250,
		MaxTxnID:       1000,
		ActiveTxns:     make([]uint64, 0),
		AbortedTxns:    make([]uint64, 0),
		IsolationLevel: ReadCommitted,
	}

	// Find version for snapshot at 250 (should get v2)
	found := FindVersion(head, snapshot)
	if found != v2 {
		t.Errorf("Expected to find v2, got %v", found)
	}

	// Update snapshot timestamp to 150
	snapshot.Timestamp = 150
	// Find version at timestamp 150 (should get v1)
	found = FindVersion(head, snapshot)
	if found != v1 {
		t.Errorf("Expected to find v1, got %v", found)
	}

	// Update snapshot timestamp to 50
	snapshot.Timestamp = 50
	// Find version at timestamp 50 (should get nil)
	found = FindVersion(head, snapshot)
	if found != nil {
		t.Error("Expected nil for timestamp before all versions")
	}
}

func TestSnapshotIsolation(t *testing.T) {
	vm := NewVersionManager()
	sm := NewSnapshotManager(vm)

	// Begin snapshot with Read Committed isolation
	snapshot := sm.BeginSnapshot(100, ReadCommitted)

	if snapshot == nil {
		t.Fatal("Failed to create snapshot")
	}
	if snapshot.IsolationLevel != ReadCommitted {
		t.Errorf("Expected ReadCommitted isolation, got %v", snapshot.IsolationLevel)
	}

	// Commit a transaction
	sm.CommitTransaction(100)

	// Verify transaction is NOT active (implicitly committed)
	// Accessing private field activeTxns for test verification
	// We can check if it's in active map (should be false)
	// But committedTxns map is gone, so we can't check that directly.
	// Instead, check IsVisible on a version from this txn.
	v := &Version{Timestamp: 10, TxnID: 100}

	// New snapshot should see it
	snap2 := sm.BeginSnapshot(101, ReadCommitted)
	if !snap2.IsVisible(v) {
		t.Error("Transaction 100 should be visible (committed)")
	}

	// Release snapshot
	sm.ReleaseSnapshot(snapshot)
}

func TestVisibilityRules(t *testing.T) {
	vm := NewVersionManager()
	sm := NewSnapshotManager(vm)

	// Create a version from transaction 1
	version := &Version{
		Timestamp: 100,
		Data:      []byte("data"),
		TxnID:     1,
	}

	// Transaction 1 is implicitly committed (never started, never active)

	// Create snapshot at timestamp 200
	snapshot := sm.BeginSnapshot(2, ReadCommitted)
	snapshot.Timestamp = 200
	snapshot.MaxTxnID = 200 // Ensure it covers txn 1

	// Version should be visible (committed and before snapshot)
	if !snapshot.IsVisible(version) {
		t.Error("Committed version before snapshot should be visible")
	}

	// Create version from uncommitted transaction
	// To simulated uncommitted, add to ActiveTxns
	snapshot.ActiveTxns = append(snapshot.ActiveTxns, 3)

	uncommittedVersion := &Version{
		Timestamp: 150,
		Data:      []byte("uncommitted"),
		TxnID:     3,
	}

	// Should not be visible (active)
	if snapshot.IsVisible(uncommittedVersion) {
		t.Error("Active version should not be visible to ReadCommitted")
	}

	sm.ReleaseSnapshot(snapshot)
}

func TestGarbageCollection(t *testing.T) {
	vm := NewVersionManager()

	// Create versions
	v1 := &Version{Timestamp: 100, Data: []byte("v1"), TxnID: 1}
	v2 := &Version{Timestamp: 200, Data: []byte("v2"), TxnID: 2}
	v3 := &Version{Timestamp: 300, Data: []byte("v3"), TxnID: 3}

	// Build chain
	head := vm.AddVersion(nil, v1)
	head = vm.AddVersion(head, v2)
	head = vm.AddVersion(head, v3)

	// Verify initial count
	if CountVersions(head) != 3 {
		t.Errorf("Expected 3 versions initially")
	}

	// Garbage collect with oldest snapshot at 250
	// Should remove v1 (timestamp 100 < 250) and v2 (timestamp 200 < 250)
	head = GarbageCollect(head, 250)

	// Count remaining versions (should be 1: v3 with timestamp 300)
	remaining := CountVersions(head)
	if remaining != 1 {
		t.Errorf("Expected 1 version after GC, got %d", remaining)
	}

	// Verify the remaining version is v3
	if head != v3 {
		t.Error("Expected head to be v3 after GC")
	}
}

func TestGarbageCollector(t *testing.T) {
	vm := NewVersionManager()
	sm := NewSnapshotManager(vm)

	gc := NewGarbageCollector(sm, time.Millisecond*100)

	// Start GC
	gc.Start()
	defer gc.Stop()

	stats := gc.GetStats()
	if !stats.Running {
		t.Error("GC should be running")
	}

	// Create and clean a version chain
	v1 := &Version{Timestamp: 100, Data: []byte("v1"), TxnID: 1}
	v2 := &Version{Timestamp: 200, Data: []byte("v2"), TxnID: 2}
	head := vm.AddVersion(v1, v2)

	cleaned := gc.ManualGC(head)
	if cleaned == nil {
		t.Error("GC should return cleaned chain")
	}

	// Stop GC
	gc.Stop()
	time.Sleep(time.Millisecond * 50)

	stats = gc.GetStats()
	if stats.Running {
		t.Error("GC should be stopped")
	}
}

func TestConcurrentTimestamps(t *testing.T) {
	vm := NewVersionManager()

	// Generate timestamps concurrently
	const numGoroutines = 100
	const timestampsPerGoroutine = 100

	timestamps := make(chan Timestamp, numGoroutines*timestampsPerGoroutine)
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < timestampsPerGoroutine; j++ {
				ts := vm.NewTimestamp()
				timestamps <- ts
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(timestamps)

	// Verify all timestamps are unique
	seen := make(map[Timestamp]bool)
	for ts := range timestamps {
		if seen[ts] {
			t.Errorf("Duplicate timestamp: %d", ts)
		}
		seen[ts] = true
	}

	expectedCount := numGoroutines * timestampsPerGoroutine
	if len(seen) != expectedCount {
		t.Errorf("Expected %d unique timestamps, got %d", expectedCount, len(seen))
	}
}
