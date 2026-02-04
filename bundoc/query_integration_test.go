package bundoc

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestFindQueryIntegration(t *testing.T) {
	// Setup DB
	dbPath := "./test_query_db"
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	opts := DefaultOptions(dbPath)
	db, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	col, err := db.CreateCollection("users")
	if err != nil {
		t.Fatal(err)
	}

	// Insert Data
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users := []storage.Document{
		{"_id": "1", "name": "Alice", "age": 25, "role": "user", "active": true},
		{"_id": "2", "name": "Bob", "age": 30, "role": "admin", "active": true},
		{"_id": "3", "name": "Charlie", "age": 35, "role": "user", "active": false},
		{"_id": "4", "name": "Dave", "age": 40, "role": "admin", "active": true},
	}
	for _, u := range users {
		col.Insert(nil, txn, u)
	}
	db.CommitTransaction(txn)

	// Test Case 1: Simple Filter (Age > 28)
	// Expected: Bob (30), Charlie (35), Dave (40)
	txnRead, _ := db.BeginTransaction(mvcc.ReadCommitted)
	defer db.RollbackTransaction(txnRead)

	q1 := map[string]interface{}{
		"age": map[string]interface{}{"$gt": 28},
	}
	results, err := col.FindQuery(nil, txnRead, q1)
	if err != nil {
		t.Fatalf("FindQuery failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results (Bob, Charlie, Dave), got %d", len(results))
	}

	// Test Case 2: Compound Filter (Role=admin AND Age < 35)
	// Expected: Bob (30) ONLY. (Dave is 40)
	q2 := map[string]interface{}{
		"role": "admin",
		"age":  map[string]interface{}{"$lt": 35},
	}
	results2, err := col.FindQuery(nil, txnRead, q2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results2) != 1 {
		t.Errorf("Expected 1 result (Bob), got %d", len(results2))
	}
	if len(results2) > 0 && results2[0]["name"] != "Bob" {
		t.Errorf("Expected Bob, got %v", results2[0]["name"])
	}

	// Test Case 3: Logical OR ($or: [ {age: 25}, {age: 40} ])
	// Expected: Alice (25), Dave (40)
	q3 := map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"age": 25},
			map[string]interface{}{"age": 40},
		},
	}
	results3, err := col.FindQuery(nil, txnRead, q3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results3) != 2 {
		t.Errorf("Expected 2 results (Alice, Dave), got %d", len(results3))
	}

	names := []string{}
	for _, doc := range results3 {
		names = append(names, fmt.Sprintf("%v", doc["name"]))
	}
	sort.Strings(names)
	if names[0] != "Alice" || names[1] != "Dave" {
		t.Errorf("Expected Alice and Dave, got %v", names)
	}

	// Test Case 4: Sort (Age Descending)
	// Expected Order: Dave (40), Charlie (35), Bob (30), Alice (25)
	optSort := QueryOptions{
		SortField: "age",
		SortDesc:  true,
	}
	// Query all
	qAll := map[string]interface{}{}
	results4, err := col.FindQuery(nil, txnRead, qAll, optSort)
	if err != nil {
		t.Fatal(err)
	}
	if len(results4) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results4))
	}
	if results4[0]["name"] != "Dave" || results4[3]["name"] != "Alice" {
		t.Errorf("Sort Failed. Expected Dave...Alice, got %v...%v", results4[0]["name"], results4[3]["name"])
	}

	// Test Case 5: Limit & Skip
	// Sort by Age Asc: Alice (25), Bob (30), Charlie (35), Dave (40)
	// Skip 1, Limit 2 -> Bob, Charlie
	optPage := QueryOptions{
		SortField: "age",
		SortDesc:  false,
		Skip:      1,
		Limit:     2,
	}
	results5, err := col.FindQuery(nil, txnRead, qAll, optPage)
	if err != nil {
		t.Fatal(err)
	}
	if len(results5) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results5))
	}
	if results5[0]["name"] != "Bob" || results5[1]["name"] != "Charlie" {
		t.Errorf("Pagination Failed. Expected Bob, Charlie. Got %v, %v", results5[0]["name"], results5[1]["name"])
	}

	// Test Case 6: Index Usage
	// Create Index on Age
	if err := col.EnsureIndex("age"); err != nil {
		t.Fatal(err)
	}

	// Query { age: { $gt: 28 } } again. Should use Index Scan now.
	// Functional check: logic should remain correct.
	fmt.Println("--- Testing Index Scan ---")
	optIndex := QueryOptions{}
	results6, err := col.FindQuery(nil, txnRead, q1, optIndex) // q1 is age > 28
	if err != nil {
		t.Fatal(err)
	}
	if len(results6) != 3 {
		t.Errorf("Index Scan Failed. Expected 3 results, got %d", len(results6))
	}
}
