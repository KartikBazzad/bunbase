package pool

import (
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
)

func TestNewPool(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 3
	poolOpts.MaxSize = 10

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	stats := pool.GetStats()
	if stats.TotalConnections != 3 {
		t.Errorf("Expected 3 initial connections, got %d", stats.TotalConnections)
	}

	if stats.MinSize != 3 {
		t.Errorf("Expected min size 3, got %d", stats.MinSize)
	}

	if stats.MaxSize != 10 {
		t.Errorf("Expected max size 10, got %d", stats.MaxSize)
	}
}

func TestAcquireRelease(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 2

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire connection
	conn, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire connection: %v", err)
	}

	if conn == nil {
		t.Fatal("Expected connection, got nil")
	}

	if !conn.InUse.Load() {
		t.Error("Connection should be marked as in use")
	}

	// Release connection
	err = pool.Release(conn)
	if err != nil {
		t.Fatalf("Failed to release connection: %v", err)
	}

	if conn.InUse.Load() {
		t.Error("Connection should not be in use after release")
	}
}

func TestPoolExpansion(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 2
	poolOpts.MaxSize = 5

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire multiple connections
	conns := make([]*Connection, 0, 4)
	for i := 0; i < 4; i++ {
		conn, err := pool.Acquire()
		if err != nil {
			t.Fatalf("Failed to acquire connection %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// Pool should have expanded to 4 connections
	stats := pool.GetStats()
	if stats.TotalConnections != 4 {
		t.Errorf("Expected 4 connections, got %d", stats.TotalConnections)
	}

	if stats.ActiveConnections != 4 {
		t.Errorf("Expected 4 active connections, got %d", stats.ActiveConnections)
	}

	// Release all
	for _, conn := range conns {
		pool.Release(conn)
	}

	stats = pool.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveConnections)
	}
}

func TestPoolMaxSize(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 1
	poolOpts.MaxSize = 3

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire up to max
	conns := make([]*Connection, 0, 3)
	for i := 0; i < 3; i++ {
		conn, err := pool.Acquire()
		if err != nil {
			t.Fatalf("Failed to acquire connection %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// Try to acquire beyond max
	_, err = pool.Acquire()
	if err == nil {
		t.Error("Expected error when exceeding max pool size")
	}

	// Release one and retry
	pool.Release(conns[0])
	conn, err := pool.Acquire()
	if err != nil {
		t.Errorf("Should be able to acquire after release: %v", err)
	}

	if conn == nil {
		t.Error("Expected connection after release")
	}
}

func TestHealthChecker(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 2
	poolOpts.MaxSize = 5
	poolOpts.IdleTimeout = 100 * time.Millisecond
	poolOpts.HealthInterval = 50 * time.Millisecond

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire and release to create idle connections
	conn, _ := pool.Acquire()
	pool.Release(conn)

	// Expand pool
	for i := 0; i < 3; i++ {
		c, _ := pool.Acquire()
		pool.Release(c)
	}

	stats := pool.GetStats()
	initialTotal := stats.TotalConnections

	// Wait for health checker to run and prune idle connections
	time.Sleep(200 * time.Millisecond)

	stats = pool.GetStats()
	// Should have pruned back to minSize (excess idle connections)
	if stats.TotalConnections > initialTotal {
		t.Errorf("Expected connections to be pruned, got %d", stats.TotalConnections)
	}

	// Should maintain minimum
	if stats.TotalConnections < poolOpts.MinSize {
		t.Errorf("Expected at least %d connections, got %d", poolOpts.MinSize, stats.TotalConnections)
	}
}

func TestConcurrentAcquireRelease(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	dbOpts := bundoc.DefaultOptions(tmpdir)
	poolOpts := DefaultPoolOptions()
	poolOpts.MinSize = 5
	poolOpts.MaxSize = 20

	pool, err := NewPool(tmpdir, dbOpts, poolOpts)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Concurrent acquire/release
	const numWorkers = 10
	const iterations = 5

	done := make(chan bool, numWorkers)
	errors := make(chan error, numWorkers*iterations)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				conn, err := pool.Acquire()
				if err != nil {
					errors <- err
					continue
				}

				// Simulate work
				time.Sleep(10 * time.Millisecond)

				if err := pool.Release(conn); err != nil {
					errors <- err
				}
			}
			done <- true
		}()
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	close(errors)
	errCount := 0
	for err := range errors {
		t.Errorf("Worker error: %v", err)
		errCount++
	}

	if errCount > 0 {
		t.Fatalf("Had %d errors during concurrent access", errCount)
	}

	stats := pool.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections after completion, got %d", stats.ActiveConnections)
	}
}
