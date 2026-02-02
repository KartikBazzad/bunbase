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

func TestLatencyPercentiles(t *testing.T) {
	// Setup
	dir, _ := os.MkdirTemp("", "stats-idx")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()
	col, _ := db.CreateCollection("users")

	// 1. Insert 10,000 docs
	count := 10000
	batchSize := 1000
	fmt.Printf("Inserting %d documents...\n", count)

	for i := 0; i < count; i += batchSize {
		docs := make([]storage.Document, batchSize)
		for j := 0; j < batchSize; j++ {
			doc := make(storage.Document)
			doc["email"] = fmt.Sprintf("user%d@example.com", i+j)
			doc["age"] = (i + j) % 100
			docs[j] = doc
		}
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.InsertBatch(txn, docs)
		db.CommitTransaction(txn)
	}

	// 2. Trigger Index Build
	fmt.Println("Building Index...")
	txnBuild, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Find(txnBuild, "email", "user0@example.com")
	db.CommitTransaction(txnBuild)

	// 3. Measure Latencies
	sampleSize := 5000
	durations := make([]time.Duration, sampleSize)
	rand.Seed(time.Now().UnixNano())

	fmt.Printf("Measuring %d random lookups...\n", sampleSize)

	for i := 0; i < sampleSize; i++ {
		targetID := rand.Intn(count)
		targetEmail := fmt.Sprintf("user%d@example.com", targetID)

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		results, err := col.Find(txn, "email", targetEmail)
		db.CommitTransaction(txn)
		durations[i] = time.Since(start)

		if err != nil || len(results) == 0 {
			t.Fatalf("Lookup failed for %s: %v", targetEmail, err)
		}
	}

	// 4. Calculate Stats
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p50 := durations[int(float64(sampleSize)*0.50)]
	p90 := durations[int(float64(sampleSize)*0.90)]
	p95 := durations[int(float64(sampleSize)*0.95)]
	p99 := durations[int(float64(sampleSize)*0.99)]
	min := durations[0]
	max := durations[sampleSize-1]
	avg := time.Duration(0)
	for _, d := range durations {
		avg += d
	}
	avg /= time.Duration(sampleSize)

	fmt.Printf("\n--- Latency Performance Stats (N=%d) ---\n", sampleSize)
	fmt.Printf("Min: %v\n", min)
	fmt.Printf("P50: %v\n", p50)
	fmt.Printf("P90: %v\n", p90)
	fmt.Printf("P95: %v\n", p95)
	fmt.Printf("P99: %v\n", p99)
	fmt.Printf("Max: %v\n", max)
	fmt.Printf("Avg: %v\n", avg)
	fmt.Println("----------------------------------------")
}
