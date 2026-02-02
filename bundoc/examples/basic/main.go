package main

import (
	"fmt"
	"log"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func main() {
	// Open database
	opts := bundoc.DefaultOptions("./example-db")
	db, err := bundoc.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database opened successfully")

	// Create collection
	users, err := db.CreateCollection("users")
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	fmt.Println("✅ Collection 'users' created")

	// Insert documents
	fmt.Println("\n📝 Inserting documents...")

	// Transaction 1: Insert Alice
	txn1, _ := db.BeginTransaction(mvcc.ReadCommitted)
	alice := storage.Document{
		"_id":   "user1",
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   30,
	}
	users.Insert(txn1, alice)
	db.CommitTransaction(txn1)
	fmt.Println("  - Inserted Alice")

	// Transaction 2: Insert Bob
	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	bob := storage.Document{
		"_id":   "user2",
		"name":  "Bob",
		"email": "bob@example.com",
		"age":   25,
	}
	users.Insert(txn2, bob)
	db.CommitTransaction(txn2)
	fmt.Println("  - Inserted Bob")

	// Read documents
	fmt.Println("\n📖 Reading documents...")

	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	foundAlice, err := users.FindByID(txn3, "user1")
	if err == nil {
		fmt.Printf("  - Found: %v\n", foundAlice)
	}

	foundBob, err := users.FindByID(txn3, "user2")
	if err == nil {
		fmt.Printf("  - Found: %v\n", foundBob)
	}
	db.CommitTransaction(txn3)

	// Update document
	fmt.Println("\n✏️  Updating Alice's age...")

	txn4, _ := db.BeginTransaction(mvcc.ReadCommitted)
	updatedAlice := storage.Document{
		"_id":   "user1",
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   31, // Birthday!
	}
	users.Update(txn4, "user1", updatedAlice)
	db.CommitTransaction(txn4)
	fmt.Println("  - Updated Alice's age to 31")

	// Read updated document
	txn5, _ := db.BeginTransaction(mvcc.ReadCommitted)
	updated, _ := users.FindByID(txn5, "user1")
	fmt.Printf("  - Alice after update: %v\n", updated)
	db.CommitTransaction(txn5)

	// List collections
	fmt.Println("\n📚 All collections:")
	collections := db.ListCollections()
	for _, name := range collections {
		fmt.Printf("  - %s\n", name)
	}

	// Delete document
	fmt.Println("\n🗑️  Deleting Bob...")
	txn6, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Delete(txn6, "user2")
	db.CommitTransaction(txn6)
	fmt.Println("  - Bob deleted")

	// Try to read deleted document
	txn7, _ := db.BeginTransaction(mvcc.ReadCommitted)
	_, err = users.FindByID(txn7, "user2")
	if err != nil {
		fmt.Println("  - Confirmed: Bob no longer exists")
	}
	db.CommitTransaction(txn7)

	fmt.Println("\n✅ Example completed successfully!")
}
