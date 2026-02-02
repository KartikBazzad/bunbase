package bundoc

import (
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestDatabaseOpenClose(t *testing.T) {
	tmpdir := t.TempDir()

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if db == nil {
		t.Fatal("Expected database instance, got nil")
	}

	if db.IsClosed() {
		t.Error("Database should not be closed after opening")
	}

	// Close database
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	if !db.IsClosed() {
		t.Error("Database should be closed after Close()")
	}
}

func TestCreateCollection(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	if coll == nil {
		t.Fatal("Expected collection, got nil")
	}

	if coll.Name() != "users" {
		t.Errorf("Expected collection name 'users', got '%s'", coll.Name())
	}

	// Try creating duplicate collection
	_, err = db.CreateCollection("users")
	if err == nil {
		t.Error("Expected error when creating duplicate collection")
	}
}

func TestListCollections(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initially empty
	colls := db.ListCollections()
	if len(colls) != 0 {
		t.Errorf("Expected 0 collections, got %d", len(colls))
	}

	// Create collections
	db.CreateCollection("users")
	db.CreateCollection("posts")
	db.CreateCollection("comments")

	colls = db.ListCollections()
	if len(colls) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(colls))
	}
}

func TestDropCollection(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create and drop collection
	db.CreateCollection("temp")

	err = db.DropCollection("temp")
	if err != nil {
		t.Fatalf("Failed to drop collection: %v", err)
	}

	// Verify it's gone
	_, err = db.GetCollection("temp")
	if err == nil {
		t.Error("Expected error when getting dropped collection")
	}
}

func TestInsertAndFind(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Begin transaction
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert document
	doc := storage.Document{
		"_id":  "user1",
		"name": "Alice",
		"age":  30,
	}

	err = coll.Insert(txn, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Find document in same transaction
	found, err := coll.FindByID(txn, "user1")
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	id, hasID := found.GetID()
	if !hasID || id != "user1" {
		t.Errorf("Expected ID 'user1', got '%s'", id)
	}

	if found["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", found["name"])
	}

	// Commit transaction
	err = db.txnMgr.Commit(txn)
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func TestUpdateDocument(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, _ := db.CreateCollection("users")
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

	// Insert
	doc := storage.Document{
		"_id":  "user1",
		"name": "Alice",
		"age":  30,
	}
	coll.Insert(txn, doc)

	// Update
	updatedDoc := storage.Document{
		"_id":  "user1",
		"name": "Alice Updated",
		"age":  31,
	}
	err = coll.Update(txn, "user1", updatedDoc)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	found, _ := coll.FindByID(txn, "user1")
	if found["name"] != "Alice Updated" {
		t.Errorf("Expected updated name, got '%v'", found["name"])
	}

	db.txnMgr.Commit(txn)
}

func TestDeleteDocument(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)

	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll, _ := db.CreateCollection("users")
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

	// Insert
	doc := storage.Document{
		"_id":  "user1",
		"name": "Alice",
	}
	coll.Insert(txn, doc)

	// Delete
	err = coll.Delete(txn, "user1")
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	db.txnMgr.Commit(txn)
}
