package bundoc

import (
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestPersistence_DataRecovery(t *testing.T) {
	dbPath := "./test_persistence_db"
	_ = os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	opts := DefaultOptions(dbPath)

	// 1. Initial Insert
	{
		db, err := Open(opts)
		if err != nil {
			t.Fatalf("Failed to open DB: %v", err)
		}

		coll, err := db.CreateCollection("users")
		if err != nil {
			t.Fatalf("Failed to create collection: %v", err)
		}

		txn, _ := db.BeginTransaction(mvcc.RepeatableRead)
		doc := storage.Document{"name": "Alice", "age": 30}
		if err := coll.Insert(txn, doc); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if err := db.CommitTransaction(txn); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		if err := db.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}

	// 2. Reopen and Verify
	{
		db, err := Open(opts)
		if err != nil {
			t.Fatalf("Failed to reopen DB: %v", err)
		}

		coll, err := db.GetCollection("users")
		if err != nil {
			t.Fatalf("Failed to get collection: %v", err)
		}

		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		results, err := coll.FindQuery(txn, map[string]interface{}{
			"name": "Alice",
		})
		if err != nil {
			t.Fatalf("FindQuery failed: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0]["name"] != "Alice" {
			t.Errorf("Expected Alice, got %v", results[0])
		}

		db.Close()
	}
}

func TestPersistence_IndexRecovery(t *testing.T) {
	dbPath := "./test_index_persistence_db"
	_ = os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	opts := DefaultOptions(dbPath)

	// 1. Create Index and Insert
	{
		db, err := Open(opts)
		if err != nil {
			t.Fatalf("Failed to open DB: %v", err)
		}

		coll, err := db.CreateCollection("items")
		if err != nil {
			t.Fatalf("Failed to create collection: %v", err)
		}

		if err := coll.EnsureIndex("category"); err != nil {
			t.Fatalf("EnsureIndex failed: %v", err)
		}

		txn, _ := db.BeginTransaction(mvcc.RepeatableRead)
		coll.Insert(txn, storage.Document{"name": "Laptop", "category": "electronics"})
		coll.Insert(txn, storage.Document{"name": "Chair", "category": "furniture"})
		db.CommitTransaction(txn)

		db.Close()
	}

	// 2. Reopen and Query via Index
	{
		db, err := Open(opts)
		if err != nil {
			t.Fatalf("Failed to reopen DB: %v", err)
		}

		coll, err := db.GetCollection("items")
		if err != nil {
			t.Fatalf("Failed to get collection: %v", err)
		}

		// Verify index exists in memory
		coll.mu.RLock()
		_, exists := coll.indexes["category"]
		coll.mu.RUnlock()
		if !exists {
			t.Fatalf("Index 'category' not restored")
		}

		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		// Query using index (mocked verify via planner log or result count)
		results, err := coll.Find(txn, "category", "electronics")
		if err != nil {
			t.Fatalf("Find(Index) failed: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0]["name"] != "Laptop" {
			t.Errorf("Expected Laptop, got %v", results[0])
		}

		db.Close()
	}
}
