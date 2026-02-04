package bundoc

import (
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
)

func TestNestedCollection(t *testing.T) {
	// Setup temporary directory
	tmpDir, err := os.MkdirTemp("", "bundoc_nested_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Open DB
	opts := DefaultOptions(tmpDir)
	db, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 1. Create Nested Collection
	nestedName := "users/admins/audit"
	coll, err := db.CreateCollection(nestedName)
	if err != nil {
		t.Fatalf("Failed to create nested collection: %v", err)
	}

	// 2. Verify Retrieval
	coll2, err := db.GetCollection(nestedName)
	if err != nil {
		t.Fatalf("Failed to get nested collection: %v", err)
	}
	if coll2.Name() != nestedName {
		t.Errorf("Expected name %s, got %s", nestedName, coll2.Name())
	}

	// 3. Insert Document
	txn, err := db.BeginTransaction(mvcc.ReadCommitted) // ReadWrite
	if err != nil {
		t.Fatal(err)
	}

	doc := map[string]interface{}{
		"_id":   "doc1",
		"event": "login",
	}

	if err := coll.Insert(nil, txn, doc); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	if err := db.CommitTransaction(txn); err != nil {
		t.Fatal(err)
	}

	// 4. Verify Data Persisted
	// Restart DB to check metadata persistence
	db.Close()
	db, err = Open(opts)
	if err != nil {
		t.Fatal(err)
	}

	coll3, err := db.GetCollection(nestedName)
	if err != nil {
		t.Fatalf("Failed to load nested collection after restart: %v", err)
	}

	txn, err = db.BeginTransaction(mvcc.ReadCommitted) // ReadOnly
	if err != nil {
		t.Fatal(err)
	}
	defer db.RollbackTransaction(txn)

	readDoc, err := coll3.FindByID(nil, txn, "doc1")
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}
	if readDoc["event"] != "login" {
		t.Errorf("Data mismatch")
	}

	// 5. Verify ListCollectionsWithPrefix
	// Create another collection to ensure filtering works
	otherName := "other/stuff"
	_, err = db.CreateCollection(otherName)
	if err != nil {
		t.Fatal(err)
	}

	// Filter "users/"
	usersColls := db.ListCollectionsWithPrefix("users/")
	if len(usersColls) != 1 || usersColls[0] != nestedName {
		t.Errorf("ListCollectionsWithPrefix('users/') failed. Got: %v, Expected: [%s]", usersColls, nestedName)
	}

	// Filter "" (all)
	allColls := db.ListCollectionsWithPrefix("")
	if len(allColls) != 2 {
		t.Errorf("ListCollectionsWithPrefix('') failed. Got: %v (len %d), Expected len 2", allColls, len(allColls))
	}

	// 6. Verify Indexes on Nested Collection
	// EnsureIndex on "event" field
	if err := coll3.EnsureIndex("event"); err != nil {
		t.Fatalf("Failed to create index on nested collection: %v", err)
	}

	// Insert another doc to test index lookup
	doc2 := map[string]interface{}{
		"_id":   "doc2",
		"event": "logout",
	}
	txn, err = db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		t.Fatal(err)
	}
	if err := coll3.Insert(nil, txn, doc2); err != nil {
		t.Fatalf("Failed to insert doc2: %v", err)
	}
	if err := db.CommitTransaction(txn); err != nil {
		t.Fatal(err)
	}

	// Query using Index
	txn, err = db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		t.Fatal(err)
	}
	defer db.CommitTransaction(txn)

	// Find "logout" event
	docs, err := coll3.Find(txn, "event", "logout")
	if err != nil {
		t.Fatalf("Failed to find using index: %v", err)
	}
	if len(docs) != 1 || docs[0]["_id"] != "doc2" {
		t.Errorf("Index lookup failed. Got: %v", docs)
	}
}
