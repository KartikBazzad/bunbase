package benchmark

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// BenchmarkWrite benchmarks write throughput
func BenchmarkWrite(b *testing.B) {
	tmpdir := b.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := bundoc.DefaultOptions(tmpdir)
	db, err := bundoc.Open(opts)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("benchmark")
	if err != nil {
		b.Fatalf("Failed to create collection: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
			"data":  "benchmark data for testing write performance",
		}

		if err := coll.Insert(nil, txn, doc); err != nil {
			b.Fatalf("Insert failed: %v", err)
		}

		if err := db.CommitTransaction(txn); err != nil {
			b.Fatalf("Commit failed: %v", err)
		}
	}

	b.StopTimer()

	// Calculate throughput
	duration := b.Elapsed()
	throughput := float64(b.N) / duration.Seconds()
	b.ReportMetric(throughput, "writes/sec")
}

// BenchmarkRead benchmarks read throughput
// NOTE: Temporarily disabled - index querying needs full integration
func BenchmarkRead_DISABLED(b *testing.B) {
	tmpdir := b.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := bundoc.DefaultOptions(tmpdir)
	db, err := bundoc.Open(opts)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("benchmark")
	if err != nil {
		b.Fatalf("Failed to create collection: %v", err)
	}

	// Pre-populate data
	const numDocs = 1000
	txnPrepare, _ := db.BeginTransaction(mvcc.ReadCommitted)
	for i := 0; i < numDocs; i++ {
		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		coll.Insert(nil, txnPrepare, doc)
	}
	db.CommitTransaction(txnPrepare)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		docID := fmt.Sprintf("doc-%d", i%numDocs)

		_, err := coll.FindByID(nil, txn, docID)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}

		db.CommitTransaction(txn)
	}

	b.StopTimer()

	duration := b.Elapsed()
	throughput := float64(b.N) / duration.Seconds()
	b.ReportMetric(throughput, "reads/sec")
}

// BenchmarkConcurrentWrites benchmarks concurrent write performance
func BenchmarkConcurrentWrites(b *testing.B) {
	tmpdir := b.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := bundoc.DefaultOptions(tmpdir)
	db, err := bundoc.Open(opts)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("benchmark")
	if err != nil {
		b.Fatalf("Failed to create collection: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

			doc := storage.Document{
				"_id":   fmt.Sprintf("doc-%d-%d", b.N, i),
				"value": i,
				"data":  "concurrent write benchmark",
			}

			coll.Insert(nil, txn, doc)
			db.CommitTransaction(txn)
			i++
		}
	})

	b.StopTimer()

	duration := b.Elapsed()
	throughput := float64(b.N) / duration.Seconds()
	b.ReportMetric(throughput, "concurrent-writes/sec")
}

// BenchmarkCommitLatency measures P99 commit latency
func BenchmarkCommitLatency(b *testing.B) {
	tmpdir := b.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := bundoc.DefaultOptions(tmpdir)
	db, err := bundoc.Open(opts)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("benchmark")
	if err != nil {
		b.Fatalf("Failed to create collection: %v", err)
	}

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		coll.Insert(nil, txn, doc)

		start := time.Now()
		db.CommitTransaction(txn)
		latency := time.Since(start)

		latencies = append(latencies, latency)
	}

	b.StopTimer()

	// Calculate P99 latency
	if len(latencies) > 0 {
		// Sort latencies
		for i := 0; i < len(latencies)-1; i++ {
			for j := 0; j < len(latencies)-i-1; j++ {
				if latencies[j] > latencies[j+1] {
					latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
				}
			}
		}

		p99Index := int(float64(len(latencies)) * 0.99)
		p99 := latencies[p99Index]

		b.ReportMetric(float64(p99.Microseconds()), "p99-µs")
		b.Logf("P99 latency: %v", p99)
	}
}

// BenchmarkMixedWorkload simulates realistic workload
func BenchmarkMixedWorkload(b *testing.B) {
	tmpdir := b.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := bundoc.DefaultOptions(tmpdir)
	db, err := bundoc.Open(opts)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("benchmark")
	if err != nil {
		b.Fatalf("Failed to create collection: %v", err)
	}

	// Pre-populate some data
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	for i := 0; i < 100; i++ {
		doc := storage.Document{
			"_id":   fmt.Sprintf("doc-%d", i),
			"value": i,
		}
		coll.Insert(nil, txn, doc)
	}
	db.CommitTransaction(txn)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		// 70% reads, 20% writes, 10% updates
		switch i % 10 {
		case 0, 1, 2, 3, 4, 5, 6: // Read
			docID := fmt.Sprintf("doc-%d", i%100)
			coll.FindByID(nil, txn, docID)

		case 7, 8: // Write
			doc := storage.Document{
				"_id":   fmt.Sprintf("new-doc-%d", i),
				"value": i,
			}
			coll.Insert(nil, txn, doc)

		case 9: // Update
			docID := fmt.Sprintf("doc-%d", i%100)
			doc := storage.Document{
				"value": i * 2,
			}
			coll.Update(nil, txn, docID, doc)
		}

		db.CommitTransaction(txn)
	}

	b.StopTimer()

	duration := b.Elapsed()
	throughput := float64(b.N) / duration.Seconds()
	b.ReportMetric(throughput, "ops/sec")
}
