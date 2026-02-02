package storage

import (
	"os"
	"testing"
)

func TestPageOperations(t *testing.T) {
	// Test page creation
	page := NewPage(1, PageTypeLeaf)
	if page.ID != 1 {
		t.Errorf("Expected page ID 1, got %d", page.ID)
	}
	if page.GetPageType() != PageTypeLeaf {
		t.Errorf("Expected page type %d, got %d", PageTypeLeaf, page.GetPageType())
	}

	// Test pin/unpin
	page.Pin()
	if !page.IsPinned() {
		t.Error("Expected page to be pinned")
	}
	page.Unpin()
	if page.IsPinned() {
		t.Error("Expected page to be unpinned")
	}

	// Test key count
	page.SetKeyCount(5)
	if page.GetKeyCount() != 5 {
		t.Errorf("Expected key count 5, got %d", page.GetKeyCount())
	}

	// Test free space
	page.SetFreeSpace(100)
	if page.GetFreeSpace() != 100 {
		t.Errorf("Expected free space 100, got %d", page.GetFreeSpace())
	}

	// Test LSN
	page.SetLSN(12345)
	if page.GetLSN() != 12345 {
		t.Errorf("Expected LSN 12345, got %d", page.GetLSN())
	}

	// Test next/prev page links
	page.SetNextPage(10)
	if page.GetNextPage() != 10 {
		t.Errorf("Expected next page 10, got %d", page.GetNextPage())
	}
	page.SetPrevPage(0)
	if page.GetPrevPage() != 0 {
		t.Errorf("Expected prev page 0, got %d", page.GetPrevPage())
	}
}

func TestPager(t *testing.T) {
	// Create temp file
	tmpfile := "test_pager.db"
	defer os.Remove(tmpfile)

	// Create pager
	pager, err := NewPager(tmpfile, nil)
	if err != nil {
		t.Fatalf("Failed to create pager: %v", err)
	}
	defer pager.Close()

	// Allocate pages
	pageID1, err := pager.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if pageID1 != 0 {
		t.Errorf("Expected first page ID to be 0, got %d", pageID1)
	}

	pageID2, err := pager.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}
	if pageID2 != 1 {
		t.Errorf("Expected second page ID to be 1, got %d", pageID2)
	}

	// Create and write a page
	page := NewPage(pageID1, PageTypeIndex)
	page.SetKeyCount(3)
	if err := pager.WritePage(page); err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Read the page back
	readPage, err := pager.ReadPage(pageID1)
	if err != nil {
		t.Fatalf("Failed to read page: %v", err)
	}
	if readPage.GetPageType() != PageTypeIndex {
		t.Errorf("Expected page type %d, got %d", PageTypeIndex, readPage.GetPageType())
	}
	if readPage.GetKeyCount() != 3 {
		t.Errorf("Expected key count 3, got %d", readPage.GetKeyCount())
	}
}

func TestBufferPool(t *testing.T) {
	// Create temp file
	tmpfile := "test_buffer_pool.db"
	defer os.Remove(tmpfile)

	// Create pager and buffer pool
	pager, err := NewPager(tmpfile, nil)
	if err != nil {
		t.Fatalf("Failed to create pager: %v", err)
	}
	defer pager.Close()

	bp := NewBufferPool(3, pager) // Capacity of 3 pages

	// Create new pages
	page1, err := bp.NewPage(PageTypeLeaf)
	if err != nil {
		t.Fatalf("Failed to create new page: %v", err)
	}
	page1.SetKeyCount(10)
	bp.UnpinPage(page1.ID, true)

	page2, err := bp.NewPage(PageTypeLeaf)
	if err != nil {
		t.Fatalf("Failed to create new page: %v", err)
	}
	page2.SetKeyCount(20)
	bp.UnpinPage(page2.ID, true)

	// Fetch page1 back
	fetchedPage, err := bp.FetchPage(page1.ID)
	if err != nil {
		t.Fatalf("Failed to fetch page: %v", err)
	}
	if fetchedPage.GetKeyCount() != 10 {
		t.Errorf("Expected key count 10, got %d", fetchedPage.GetKeyCount())
	}
	bp.UnpinPage(fetchedPage.ID, false)

	// Flush all pages
	if err := bp.FlushAllPages(); err != nil {
		t.Fatalf("Failed to flush pages: %v", err)
	}

	// Verify buffer pool size
	if bp.Size() != 2 {
		t.Errorf("Expected buffer pool size 2, got %d", bp.Size())
	}
}

func TestDocument(t *testing.T) {
	// Create document
	doc := Document{
		"name":  "Alice",
		"age":   30,
		"email": "alice@example.com",
	}

	// Test serialization
	data, err := doc.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize document: %v", err)
	}

	// Test deserialization
	doc2, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize document: %v", err)
	}

	if doc2["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", doc2["name"])
	}
	if doc2["age"].(float64) != 30 {
		t.Errorf("Expected age 30, got %v", doc2["age"])
	}

	// Test ID operations
	doc.SetID("doc123")
	id, exists := doc.GetID()
	if !exists {
		t.Error("Expected document to have an ID")
	}
	if id != "doc123" {
		t.Errorf("Expected ID 'doc123', got %s", id)
	}

	// Test clone
	clone := doc.Clone()
	clone["name"] = "Bob"
	if doc["name"] == "Bob" {
		t.Error("Clone should not modify original document")
	}
}
