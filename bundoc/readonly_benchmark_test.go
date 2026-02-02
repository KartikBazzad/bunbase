package bundoc

import (
	"fmt"
	"sync"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// BenchmarkReadOnlyTransaction measures read-only transaction performance
func BenchmarkReadOnlyTransaction(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_readonly")

	// Seed data
	const seedCount = 1000
	for i := 0; i < seedCount; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		col.Insert(txn, doc)
		db.txnMgr.Commit(txn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		docID := fmt.Sprintf("doc-%d", i%seedCount)
		col.FindByID(txn, docID)
		db.txnMgr.Commit(txn) // Should be FAST - no WAL!
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkReadOnlyConcurrent measures concurrent read-only transaction performance
func BenchmarkReadOnlyConcurrent(b *testing.B) {
	workers := []int{1, 10, 50, 100}

	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("Workers-%d", numWorkers), func(b *testing.B) {
			db, cleanup := setupBenchDB(b)
			defer cleanup()

			col := createBenchCollection(b, db, "concurrent_readonly")

			b.StopTimer() // Stop timer during seeding
			// Seed data
			const seedCount = 500
			for i := 0; i < seedCount; i++ {
				txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
				doc := storage.Document{
					"_id":   fmt.Sprintf("doc-%d", i),
					"value": i,
				}
				col.Insert(txn, doc)
				db.txnMgr.Commit(txn)
			}
			b.StartTimer() // Restart timer for actual test

			var wg sync.WaitGroup
			opsPerWorker := b.N / numWorkers

			b.ResetTimer()
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < opsPerWorker; i++ {
						txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
						docID := fmt.Sprintf("doc-%d", (workerID*opsPerWorker+i)%seedCount)
						col.FindByID(txn, docID)
						db.txnMgr.Commit(txn) // Read-only fast path!
					}
				}(w)
			}
			wg.Wait()
			b.StopTimer()

			totalOps := numWorkers * opsPerWorker
			b.ReportMetric(float64(totalOps)/b.Elapsed().Seconds(), "ops/sec")
		})
	}
}

// BenchmarkCompareReadWrite compares read-only vs read-write transaction performance
func BenchmarkCompareReadWrite(b *testing.B) {
	b.Run("ReadOnly", func(b *testing.B) {
		db, cleanup := setupBenchDB(b)
		defer cleanup()

		col := createBenchCollection(b, db, "ro_compare")

		// Seed
		for i := 0; i < 100; i++ {
			txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
			doc := storage.Document{"_id": fmt.Sprintf("doc-%d", i), "value": i}
			col.Insert(txn, doc)
			db.txnMgr.Commit(txn)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
			col.FindByID(txn, "doc-0")
			db.txnMgr.Commit(txn) // Read-only!
		}
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	})

	b.Run("ReadWrite", func(b *testing.B) {
		db, cleanup := setupBenchDB(b)
		defer cleanup()

		col := createBenchCollection(b, db, "rw_compare")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
			doc := storage.Document{
				"_id":   fmt.Sprintf("doc-%d", i),
				"value": i,
			}
			col.Insert(txn, doc)
			db.txnMgr.Commit(txn) // Write transaction - full WAL
		}
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	})
}
