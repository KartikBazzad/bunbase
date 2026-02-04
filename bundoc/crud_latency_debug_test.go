package bundoc

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestCRUDLatencyProfileDebug(t *testing.T) {
	// Setup
	dir, _ := os.MkdirTemp("", "crud-stats-debug")
	defer os.RemoveAll(dir)
	db, _ := Open(DefaultOptions(dir))
	defer db.Close()
	col, _ := db.CreateCollection("users")

	count := 100 // SMALL COUNT
	fmt.Printf("Initializing DB with %d docs...\n", count)

	// 1. INSERT
	insertDurations := make([]time.Duration, count)
	for i := 0; i < count; i++ {
		doc := make(storage.Document)
		doc["email"] = fmt.Sprintf("user%d@example.com", i)
		doc["age"] = i % 100
		id := fmt.Sprintf("user%d", i)
		doc.SetID(storage.DocumentID(id))

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Insert(nil, txn, doc)
		db.CommitTransaction(txn)
		insertDurations[i] = time.Since(start)
	}
	printStats("INSERT", insertDurations)

	// Trigger Index Build
	txnBuild, _ := db.BeginTransaction(mvcc.ReadCommitted)
	col.Find(txnBuild, "email", "xyz")
	db.CommitTransaction(txnBuild)

	// 2. READ
	readDurations := make([]time.Duration, count)
	rand.Seed(time.Now().UnixNano())
	ids := rand.Perm(count)

	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)
		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.FindByID(nil, txn, id)
		db.CommitTransaction(txn)
		readDurations[i] = time.Since(start)
	}
	printStats("READ (PK)", readDurations)

	// 3. UPDATE
	updateDurations := make([]time.Duration, count)
	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)
		doc := make(storage.Document)
		doc["email"] = fmt.Sprintf("user%d_updated@example.com", idx)
		doc["age"] = (idx + 1) % 100

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Update(nil, txn, id, doc)
		db.CommitTransaction(txn)
		updateDurations[i] = time.Since(start)
	}
	printStats("UPDATE", updateDurations)

	// 4. DELETE
	deleteDurations := make([]time.Duration, count)
	for i, idx := range ids {
		id := fmt.Sprintf("user%d", idx)

		start := time.Now()
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		col.Delete(nil, txn, id)
		db.CommitTransaction(txn)
		deleteDurations[i] = time.Since(start)
	}
	printStats("DELETE", deleteDurations)
}
