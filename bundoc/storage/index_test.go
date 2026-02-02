package storage

import (
	"fmt"
	"os"
	"testing"
)

func TestBPlusTreeBasicOperations(t *testing.T) {
	// Create temp file
	tmpfile := "test_btree.db"
	defer os.Remove(tmpfile)

	// Create pager and buffer pool
	pager, err := NewPager(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create pager: %v", err)
	}
	defer pager.Close()

	bp := NewBufferPool(100, pager)

	// Create B+ tree
	tree, err := NewBPlusTree(bp)
	if err != nil {
		t.Fatalf("Failed to create B+ tree: %v", err)
	}

	// Insert some entries
	testData := map[string]string{
		"apple":  "red fruit",
		"banana": "yellow fruit",
		"cherry": "red fruit",
		"date":   "brown fruit",
	}

	for key, value := range testData {
		if err := tree.Insert([]byte(key), []byte(value)); err != nil {
			t.Fatalf("Failed to insert %s: %v", key, err)
		}
	}

	// Search for entries
	for key, expectedValue := range testData {
		value, err := tree.Search([]byte(key))
		if err != nil {
			t.Errorf("Failed to find key %s: %v", key, err)
		}
		if string(value) != expectedValue {
			t.Errorf("For key %s, expected %s, got %s", key, expectedValue, string(value))
		}
	}

	// Search for non-existent key
	_, err = tree.Search([]byte("elderberry"))
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestBPlusTreeRangeScan(t *testing.T) {
	// Create temp file
	tmpfile := "test_btree_range.db"
	defer os.Remove(tmpfile)

	// Create pager and buffer pool
	pager, err := NewPager(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create pager: %v", err)
	}
	defer pager.Close()

	bp := NewBufferPool(100, pager)

	// Create B+ tree
	tree, err := NewBPlusTree(bp)
	if err != nil {
		t.Fatalf("Failed to create B+ tree: %v", err)
	}

	// Insert ordered entries
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key%02d", i)
		value := fmt.Sprintf("value%02d", i)
		if err := tree.Insert([]byte(key), []byte(value)); err != nil {
			t.Fatalf("Failed to insert %s: %v", key, err)
		}
	}

	// Perform range scan
	results, err := tree.RangeScan([]byte("key03"), []byte("key07"))
	if err != nil {
		t.Fatalf("Range scan failed: %v", err)
	}

	// Verify results
	expectedCount := 5 // key03, key04, key05, key06, key07
	if len(results) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(results))
	}

	// Verify first and last keys
	if len(results) > 0 {
		if string(results[0].Key) != "key03" {
			t.Errorf("Expected first key to be key03, got %s", string(results[0].Key))
		}
		if string(results[len(results)-1].Key) != "key07" {
			t.Errorf("Expected last key to be key07, got %s", string(results[len(results)-1].Key))
		}
	}
}

func TestBPlusTreeUpdate(t *testing.T) {
	// Create temp file
	tmpfile := "test_btree_update.db"
	defer os.Remove(tmpfile)

	// Create pager and buffer pool
	pager, err := NewPager(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create pager: %v", err)
	}
	defer pager.Close()

	bp := NewBufferPool(100, pager)

	// Create B+ tree
	tree, err := NewBPlusTree(bp)
	if err != nil {
		t.Fatalf("Failed to create B+ tree: %v", err)
	}

	// Insert entry
	key := []byte("test_key")
	value1 := []byte("initial_value")
	if err := tree.Insert(key, value1); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify initial value
	result, err := tree.Search(key)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if string(result) != string(value1) {
		t.Errorf("Expected %s, got %s", string(value1), string(result))
	}

	// Update value
	value2 := []byte("updated_value")
	if err := tree.Insert(key, value2); err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify updated value
	result, err = tree.Search(key)
	if err != nil {
		t.Fatalf("Failed to search after update: %v", err)
	}
	if string(result) != string(value2) {
		t.Errorf("Expected %s, got %s", string(value2), string(result))
	}
}
