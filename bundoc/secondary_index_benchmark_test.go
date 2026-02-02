package bundoc

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestSecondaryIndexLazyCreation(t *testing.T) {
	// Setup
	dir, _ := os.MkdirTemp("", "test-index")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()

	col, _ := db.CreateCollection("users")

	// 1. Insert 1000 docs
	count := 1000
	docs := make([]storage.Document, count)
	for i := 0; i < count; i++ {
		doc := make(storage.Document)
		doc["email"] = fmt.Sprintf("user%d@example.com", i)
		doc["age"] = i % 50
		docs[i] = doc
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.InsertBatch(txn, docs)
	db.CommitTransaction(txn)

	// 2. Query WITHOUT index (First call - should trigger Lazy Index build)
	start := time.Now()
	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results, err := col.Find(txn2, "email", "user50@example.com")
	db.CommitTransaction(txn2)
	duration1 := time.Since(start)

	if err != nil {
		t.Fatalf("First Find failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	t.Logf("First Query (Lazy Build): %v", duration1)

	// 3. Query WITH index (Second call - should be fast)
	start = time.Now()
	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results2, err := col.Find(txn3, "email", "user50@example.com")
	db.CommitTransaction(txn3)
	duration2 := time.Since(start)

	if err != nil {
		t.Fatalf("Second Find failed: %v", err)
	}
	if len(results2) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results2))
	}
	t.Logf("Second Query (Indexed): %v", duration2)

	// 4. Verify Improvement
	if duration1 < duration2*2 {
		t.Errorf("First query should be significantly slower than second. Got %v vs %v", duration1, duration2)
	}
}

func BenchmarkSecondaryIndexLookup(b *testing.B) {
	// Setup
	dir, _ := os.MkdirTemp("", "bench-idx")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()
	col, _ := db.CreateCollection("users")

	// Insert 10k docs
	count := 10000
	batchSize := 100
	for i := 0; i < count; i += batchSize {
		docs := make([]storage.Document, batchSize)
		for j := 0; j < batchSize; j++ {
			doc := make(storage.Document)
			doc["email"] = fmt.Sprintf("user%d@example.com", i+j)
			docs[j] = doc
		}
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.InsertBatch(txn, docs)
		db.CommitTransaction(txn)
	}

	// Trigger index build once
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Find(txn, "email", "user0@example.com")
	db.CommitTransaction(txn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetEmail := fmt.Sprintf("user%d@example.com", i%count)
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		_, err := col.Find(txn, "email", targetEmail)
		if err != nil {
			b.Fatal(err)
		}
		db.CommitTransaction(txn) // Read-only commit is fast
	}
}
