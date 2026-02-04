package bundoc

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestCRUDLatencyProfile(t *testing.T) {
	// Setup
	dir, _ := os.MkdirTemp("", "crud-stats")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()
	col, _ := db.CreateCollection("users")

	count := 1000
	fmt.Printf("Initializing DB with %d docs...\n", count)

	// ----------------------------------------------------------------
	// 1. INSERT Latency
	// ----------------------------------------------------------------
	insertDurations := make([]time.Duration, count)
	for i := 0; i < count; i++ {
		doc := make(storage.Document)
		doc["email"] = fmt.Sprintf("user%d@example.com", i)
		doc["age"] = i % 100
		id := fmt.Sprintf("user%d", i)
		doc.SetID(storage.DocumentID(id))

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Insert(nil, txn, doc)
		db.CommitTransaction(txn)
		insertDurations[i] = time.Since(start)
	}
	printStats("INSERT", insertDurations)

	// Trigger Index Build (Lazy)
	txnBuild, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Find(txnBuild, "email", "xyz")
	db.CommitTransaction(txnBuild)

	// ----------------------------------------------------------------
	// 2. READ Latency (Primary Key)
	// ----------------------------------------------------------------
	readDurations := make([]time.Duration, count)
	rand.Seed(time.Now().UnixNano())
	ids := rand.Perm(count) // Random order

	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)
		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.FindByID(nil, txn, id)
		db.CommitTransaction(txn)
		readDurations[i] = time.Since(start)
	}
	printStats("READ (PK)", readDurations)

	// ----------------------------------------------------------------
	// 3. UPDATE Latency (Requires Read + Index Maint)
	// ----------------------------------------------------------------
	updateDurations := make([]time.Duration, count)
	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)
		doc := make(storage.Document)
		doc["email"] = fmt.Sprintf("user%d_updated@example.com", idx) // Change indexed field
		doc["age"] = (idx + 1) % 100

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Update(nil, txn, id, doc)
		db.CommitTransaction(txn)
		updateDurations[i] = time.Since(start)
	}
	printStats("UPDATE", updateDurations)

	// ----------------------------------------------------------------
	// 4. DELETE Latency (Requires Read + Index Cleanup)
	// ----------------------------------------------------------------
	deleteDurations := make([]time.Duration, count)
	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Delete(nil, txn, id)
		db.CommitTransaction(txn)
		deleteDurations[i] = time.Since(start)
	}
	printStats("DELETE", deleteDurations)
}

func printStats(name string, durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	n := len(durations)
	p50 := durations[int(float64(n)*0.50)]
	p90 := durations[int(float64(n)*0.90)]
	p95 := durations[int(float64(n)*0.95)]
	p99 := durations[int(float64(n)*0.99)]
	max := durations[n-1]

	fmt.Printf("\n--- %s Latency (N=%d) ---\n", name, n)
	fmt.Printf("P50: %v\n", p50)
	fmt.Printf("P90: %v\n", p90)
	fmt.Printf("P95: %v\n", p95)
	fmt.Printf("P99: %v\n", p99)
	fmt.Printf("Max: %v\n", max)
}
