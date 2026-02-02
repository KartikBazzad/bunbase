package manager

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewInstanceManager(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	if mgr.dataPath != tmpdir {
		t.Errorf("Expected data path %s, got %s", tmpdir, mgr.dataPath)
	}

	if mgr.maxHot != 100 {
		t.Errorf("Expected maxHot 100, got %d", mgr.maxHot)
	}
}

func TestAcquireAndRelease(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Acquire instance
	db, release, err := mgr.Acquire("test-project")
	if err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}
	defer release()

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	// Verify instance is hot
	stats := mgr.GetStats()
	if stats.TotalInstances != 1 {
		t.Errorf("Expected 1 instance, got %d", stats.TotalInstances)
	}

	if stats.ActiveInstances != 1 {
		t.Errorf("Expected 1 active instance, got %d", stats.ActiveInstances)
	}
}

func TestHotPathOptimization(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	projectID := "test-project"

	// First acquire (cold start)
	db1, release1, err := mgr.Acquire(projectID)
	if err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}
	release1()

	// Second acquire (hot path - should be fast)
	db2, release2, err := mgr.Acquire(projectID)
	if err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}
	defer release2()

	// Should be same instance
	if db1 != db2 {
		t.Error("Expected same database instance on hot path")
	}

	stats := mgr.GetStats()
	if stats.TotalInstances != 1 {
		t.Errorf("Expected 1 instance, got %d", stats.TotalInstances)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	const numGoroutines = 50
	const numProjects = 10

	var successCount atomic.Int32
	var wg sync.WaitGroup

	// Concurrent acquires
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			projectID := "project-" + string(rune('0'+workerID%numProjects))

			db, release, err := mgr.Acquire(projectID)
			if err != nil {
				t.Errorf("Worker %d failed to acquire: %v", workerID, err)
				return
			}
			defer release()

			if db == nil {
				t.Errorf("Worker %d got nil database", workerID)
				return
			}

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	if successCount.Load() != numGoroutines {
		t.Errorf("Expected %d successful acquires, got %d", numGoroutines, successCount.Load())
	}

	stats := mgr.GetStats()
	if stats.TotalInstances != numProjects {
		t.Errorf("Expected %d instances, got %d", numProjects, stats.TotalInstances)
	}
}

func TestEviction(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	opts.IdleTTL = 100 * time.Millisecond
	opts.EvictInterval = 50 * time.Millisecond

	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Acquire and release
	db, release, err := mgr.Acquire("test-project")
	if err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	release()

	// Wait for eviction
	time.Sleep(200 * time.Millisecond)

	stats := mgr.GetStats()
	if stats.TotalInstances != 0 {
		t.Errorf("Expected 0 instances after eviction, got %d", stats.TotalInstances)
	}
}

func TestNoEvictionWhileActive(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	opts.IdleTTL = 100 * time.Millisecond
	opts.EvictInterval = 50 * time.Millisecond

	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Acquire and keep holding
	db, release, err := mgr.Acquire("test-project")
	if err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}
	defer release()

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	// Wait past TTL
	time.Sleep(200 * time.Millisecond)

	// Should NOT be evicted (still active)
	stats := mgr.GetStats()
	if stats.TotalInstances != 1 {
		t.Errorf("Expected 1 instance (should not evict active), got %d", stats.TotalInstances)
	}

	if stats.ActiveInstances != 1 {
		t.Errorf("Expected 1 active instance, got %d", stats.ActiveInstances)
	}
}

func TestManagerClose(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultManagerOptions(tmpdir)
	mgr, err := NewInstanceManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Acquire some instances
	_, release1, _ := mgr.Acquire("project-1")
	_, release2, _ := mgr.Acquire("project-2")
	release1()
	release2()

	// Close manager
	err = mgr.Close()
	if err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}

	// Should not be able to acquire after close
	_, _, err = mgr.Acquire("project-3")
	if err == nil {
		t.Error("Expected error when acquiring from closed manager")
	}
}
