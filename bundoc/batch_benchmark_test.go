package bundoc

import (
	"fmt"
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func BenchmarkInsertBatch(b *testing.B) {
	// Setup database
	dir, _ := os.MkdirTemp("", "bench-batch")
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultOptions(dir))
	defer db.Close()

	col, _ := db.CreateCollection("users")

	// Pre-allocate documents
	batchSizes := []int{10, 100, 1000}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize-%d", batchSize), func(b *testing.B) {
			// Generate batch
			docs := make([]storage.Document, batchSize)
			for i := 0; i < batchSize; i++ {
				doc := make(storage.Document)
				doc["name"] = fmt.Sprintf("User-%d", i)
				doc["age"] = i % 100
				docs[i] = doc
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
				err := col.InsertBatch(txn, docs)
				if err != nil {
					b.Fatal(err)
				}
				err = db.CommitTransaction(txn)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSingleInserts benchmarks inserting same amount of documents but one by one in one transaction
func BenchmarkSingleInsertsInTxn(b *testing.B) {
	// Setup database
	dir, _ := os.MkdirTemp("", "bench-single")
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultOptions(dir))
	defer db.Close()

	col, _ := db.CreateCollection("users")

	batchSizes := []int{10, 100, 1000}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize-%d", batchSize), func(b *testing.B) {
			docs := make([]storage.Document, batchSize)
			for i := 0; i < batchSize; i++ {
				doc := make(storage.Document)
				doc["name"] = fmt.Sprintf("User-%d", i)
				doc["age"] = i % 100
				docs[i] = doc
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
				for _, doc := range docs {
					err := col.Insert(nil, txn, doc)
					if err != nil {
						b.Fatal(err)
					}
				}
				err := db.CommitTransaction(txn)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
