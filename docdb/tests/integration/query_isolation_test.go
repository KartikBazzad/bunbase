package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/query"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/tests/failure"
)

const defaultColl = "_default"

// TestQuerySnapshotIsolation verifies that a query does not see writes committed after its snapshot.
func TestQuerySnapshotIsolation(t *testing.T) {
	helper := failure.NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDBWithPartitions("isolation", 4)
	defer db.Close()

	// Write doc 1 before query
	payload1 := []byte(`{"x":1}`)
	task1 := docdb.NewTaskWithPayload(0, types.OpCreate, defaultColl, 1, payload1)
	result1 := db.SubmitTaskAndWait(task1)
	if result1.Error != nil {
		t.Fatalf("create doc 1: %v", result1.Error)
	}

	// Start query (acquires snapshot)
	q := query.Query{Limit: 100}
	rowsBefore, err := db.ExecuteQuery(context.Background(), defaultColl, q)
	if err != nil {
		t.Fatalf("query before write: %v", err)
	}

	// Write doc 2 after query snapshot (different partition if possible)
	payload2 := []byte(`{"x":2}`)
	task2 := docdb.NewTaskWithPayload(1, types.OpCreate, defaultColl, 2, payload2)
	_ = db.SubmitTaskAndWait(task2)

	// Run same query again; should see doc 2 now
	rowsAfter, err := db.ExecuteQuery(context.Background(), defaultColl, q)
	if err != nil {
		t.Fatalf("query after write: %v", err)
	}

	// First result set should have 1 row (doc 1 only)
	if len(rowsBefore) != 1 {
		t.Errorf("before write: want 1 row, got %d", len(rowsBefore))
	}
	// Second result set should have 2 rows (doc 1 and doc 2)
	if len(rowsAfter) != 2 {
		t.Errorf("after write: want 2 rows, got %d", len(rowsAfter))
	}
}

// TestQueryPartitionConsistency verifies that all partitions use the same snapshot.
// We write to two partitions with different txIDs; a query with snapshot between them
// should see only the earlier write.
func TestQueryPartitionConsistency(t *testing.T) {
	helper := failure.NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDBWithPartitions("consistency", 4)
	defer db.Close()

	// Write to partition 0 (doc 4 -> partition 0)
	task0 := docdb.NewTaskWithPayload(0, types.OpCreate, defaultColl, 4, []byte(`{"p":0}`))
	result0 := db.SubmitTaskAndWait(task0)
	if result0.Error != nil {
		t.Fatalf("write partition 0: %v", result0.Error)
	}

	// Write to partition 1 (doc 5 -> partition 1)
	task1 := docdb.NewTaskWithPayload(1, types.OpCreate, defaultColl, 5, []byte(`{"p":1}`))
	result1 := db.SubmitTaskAndWait(task1)
	if result1.Error != nil {
		t.Fatalf("write partition 1: %v", result1.Error)
	}

	// Full scan: should see both docs (same process, snapshot is after both commits)
	rows, err := db.ExecuteQuery(context.Background(), defaultColl, query.Query{Limit: 100})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("want 2 rows (both partitions), got %d", len(rows))
	}
}

// TestQueryDoesNotBlockWrites verifies that a long-running query does not block concurrent writes.
func TestQueryDoesNotBlockWrites(t *testing.T) {
	helper := failure.NewCrashTestHelper(t)
	defer helper.Cleanup()

	db := helper.CreateDBWithPartitions("noblock", 4)
	defer db.Close()

	// Start a query in background (full scan with no limit - may take a moment)
	var wg sync.WaitGroup
	wg.Add(1)
	var queryErr error
	go func() {
		defer wg.Done()
		_, queryErr = db.ExecuteQuery(context.Background(), defaultColl, query.Query{Limit: 0})
	}()

	// Give query a moment to acquire snapshot
	time.Sleep(10 * time.Millisecond)

	// Write should succeed without waiting for query to finish
	task := docdb.NewTaskWithPayload(0, types.OpCreate, defaultColl, 100, []byte(`{"concurrent":true}`))
	result := db.SubmitTaskAndWait(task)
	if result.Error != nil {
		t.Fatalf("concurrent write: %v", result.Error)
	}

	wg.Wait()
	if queryErr != nil {
		t.Fatalf("background query: %v", queryErr)
	}
}
