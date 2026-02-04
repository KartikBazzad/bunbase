package bundoc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// BenchmarkInsert measures single-threaded insert performance
func BenchmarkInsert(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_insert")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
			"data":  "benchmark data",
		}
		col.Insert(nil, txn, doc)
		db.txnMgr.Commit(txn)
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkFindByID measures point lookup performance
func BenchmarkFindByID(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_find")

	// Seed data
	const seedCount = 10000
	for i := 0; i < seedCount; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		col.Insert(nil, txn, doc)
		db.txnMgr.Commit(txn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		docID := fmt.Sprintf("doc-%d", i%seedCount)
		col.FindByID(nil, txn, docID)
		db.txnMgr.Commit(txn)
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkUpdate measures update performance
func BenchmarkUpdate(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_update")

	// Seed data
	const seedCount = 1000
	for i := 0; i < seedCount; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		col.Insert(nil, txn, doc)
		db.txnMgr.Commit(txn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		docID := fmt.Sprintf("doc-%d", i%seedCount)
		update := storage.Document{
			"_id":     docID,
			"value":   i,
			"updated": true,
		}
		col.Update(nil, txn, docID, update)
		db.txnMgr.Commit(txn)
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkConcurrentWrites measures concurrent write performance
func BenchmarkConcurrentWrites(b *testing.B) {
	workers := []int{1, 10, 50}

	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("Workers-%d", numWorkers), func(b *testing.B) {
			db, cleanup := setupBenchDB(b)
			defer cleanup()

			col := createBenchCollection(b, db, "concurrent_writes")

			var wg sync.WaitGroup
			opsPerWorker := b.N / numWorkers

			b.ResetTimer()
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < opsPerWorker; i++ {
						txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
						doc := storage.Document{
							"_id":    fmt.Sprintf("w%d-doc%d", workerID, i),
							"worker": workerID,
							"value":  i,
						}
						col.Insert(nil, txn, doc)
						db.txnMgr.Commit(txn)
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

// BenchmarkConcurrentReads measures concurrent read performance
func BenchmarkConcurrentReads(b *testing.B) {
	workers := []int{1, 10, 50}

	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("Workers-%d", numWorkers), func(b *testing.B) {
			db, cleanup := setupBenchDB(b)
			defer cleanup()

			col := createBenchCollection(b, db, "concurrent_reads")

			// Seed data
			const seedCount = 10000
			for i := 0; i < seedCount; i++ {
				txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
				doc := storage.Document{
					"_id":   fmt.Sprintf("doc-%d", i),
					"value": i,
				}
				col.Insert(nil, txn, doc)
				db.txnMgr.Commit(txn)
			}

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
						col.FindByID(nil, txn, docID)
						db.txnMgr.Commit(txn)
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

// BenchmarkMixedWorkload measures realistic mixed read/write performance
func BenchmarkMixedWorkload(b *testing.B) {
	ratios := []struct {
		name      string
		readPct   float64
		writePct  float64
		updatePct float64
	}{
		{"ReadHeavy-80-20", 0.80, 0.15, 0.05},
		{"Balanced-50-50", 0.50, 0.30, 0.20},
		{"WriteHeavy-20-80", 0.20, 0.50, 0.30},
	}

	for _, ratio := range ratios {
		b.Run(ratio.name, func(b *testing.B) {
			db, cleanup := setupBenchDB(b)
			defer cleanup()

			col := createBenchCollection(b, db, "mixed_workload")

			// Seed initial data
			const seedCount = 5000
			for i := 0; i < seedCount; i++ {
				txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
				doc := storage.Document{
					"_id":   fmt.Sprintf("doc-%d", i),
					"value": i,
				}
				col.Insert(nil, txn, doc)
				db.txnMgr.Commit(txn)
			}

			var docCounter atomic.Int32
			docCounter.Store(seedCount)

			b.ResetTimer()
			var wg sync.WaitGroup
			const numWorkers = 10
			opsPerWorker := b.N / numWorkers

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < opsPerWorker; i++ {
						roll := float64(i%100) / 100.0
						txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

						if roll < ratio.readPct {
							// Read
							docID := fmt.Sprintf("doc-%d", i%seedCount)
							col.FindByID(nil, txn, docID)
						} else if roll < ratio.readPct+ratio.writePct {
							// Write
							newID := docCounter.Add(1)
							doc := storage.Document{
								"_id":    fmt.Sprintf("doc-%d", newID),
								"worker": workerID,
								"value":  newID,
							}
							col.Insert(nil, txn, doc)
						} else {
							// Update
							docID := fmt.Sprintf("doc-%d", i%seedCount)
							update := storage.Document{
								"_id":     docID,
								"updated": true,
								"value":   i,
							}
							col.Update(nil, txn, docID, update)
						}
						db.txnMgr.Commit(txn)
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

// Helper functions

func setupBenchDB(b *testing.B) (*Database, func()) {
	b.Helper()

	dbPath := b.TempDir() + "/bench.db"
	opts := DefaultOptions(dbPath)
	opts.BufferPoolSize = 1000 // Larger pool for benchmarks

	db, err := Open(opts)
	if err != nil {
		b.Fatal(err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createBenchCollection(b *testing.B, db *Database, name string) *Collection {
	b.Helper()

	col, err := db.CreateCollection(name)
	if err != nil {
		b.Fatal(err)
	}

	return col
}
