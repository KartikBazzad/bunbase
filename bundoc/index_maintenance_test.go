package bundoc

import (
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestSecondaryIndexMaintenance(t *testing.T) {
	// Setup
	dir, _ := os.MkdirTemp("", "test-index-maint")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()

	col, _ := db.CreateCollection("users")

	// 1. Insert Document (Age 25)
	doc := make(storage.Document)
	doc["name"] = "Alice"
	doc["age"] = 25
	// fmt.Printf("DEBUG: Insert Age Type: %T Value: %v\n", doc["age"], doc["age"])
	doc.SetID("user1")

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Insert(txn, doc)
	db.CommitTransaction(txn)

	// Force Index Create by querying
	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results, err := col.Find(txn2, "age", 25)
	db.CommitTransaction(txn2)

	if err != nil || len(results) != 1 {
		t.Fatalf("Step 1: Expected 1 result for age 25, got %d. Err: %v", len(results), err)
	}

	// 2. Update Document (Age 25 -> 26)
	doc["age"] = 26
	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Update(txn3, "user1", doc)
	db.CommitTransaction(txn3)

	// Verify Age 25 is GONE
	txn4, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results25, _ := col.Find(txn4, "age", 25)
	db.CommitTransaction(txn4)
	if len(results25) != 0 {
		t.Errorf("Step 2a: Expected 0 results for age 25 (Old value), got %d (Ghost entry!)", len(results25))
	}

	// Verify Age 26 is PRESENT
	txn5, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results26, _ := col.Find(txn5, "age", 26)
	db.CommitTransaction(txn5)
	if len(results26) != 1 {
		t.Errorf("Step 2b: Expected 1 result for age 26 (New value), got %d", len(results26))
	}

	// 3. Delete Document
	txn6, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Delete(txn6, "user1")
	db.CommitTransaction(txn6)

	// Verify Age 26 is GONE
	txn7, _ := db.BeginTransaction(mvcc.ReadCommitted)
	results26d, _ := col.Find(txn7, "age", 26)
	db.CommitTransaction(txn7)
	if len(results26d) != 0 {
		t.Errorf("Step 3: Expected 0 results for age 26 after delete, got %d (Ghost entry!)", len(results26d))
	}
}
