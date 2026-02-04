package bundoc

import (
	"fmt"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// BenchmarkInsertNoSchema measures insert performance without schema
func BenchmarkInsertNoSchema(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_no_schema")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		doc := storage.Document{
			"_id":    fmt.Sprintf("doc-%d", i),
			"name":   "User Name",
			"age":    i,
			"active": true,
		}
		col.Insert(nil, txn, doc)
		db.txnMgr.Commit(txn)
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}

// BenchmarkInsertWithSchema measures insert performance WITH schema validation
func BenchmarkInsertWithSchema(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	col := createBenchCollection(b, db, "bench_with_schema")

	// Set Schema
	schema := `{
		"type": "object",
		"properties": {
			"name": { "type": "string" },
			"age": { "type": "integer" },
			"active": { "type": "boolean" }
		},
		"required": ["name", "age"]
	}`
	if err := col.SetSchema(schema); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

		doc := storage.Document{
			"_id":    fmt.Sprintf("doc-%d", i),
			"name":   "User Name",
			"age":    i,
			"active": true,
		}
		if err := col.Insert(nil, txn, doc); err != nil {
			b.Fatal(err)
		}
		db.txnMgr.Commit(txn)
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}
