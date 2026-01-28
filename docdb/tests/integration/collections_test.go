package integration

import (
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func TestCollectionCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a collection
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Verify collection exists
	collections := db.ListCollections()
	found := false
	for _, coll := range collections {
		if coll == "users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Collection 'users' not found")
	}

	// Try to create duplicate collection
	if err := db.CreateCollection("users"); err == nil {
		t.Error("Expected error when creating duplicate collection")
	} else if err != errors.ErrCollectionExists {
		t.Errorf("Expected ErrCollectionExists, got %v", err)
	}
}

func TestCollectionDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a collection
	if err := db.CreateCollection("temp"); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Delete empty collection
	if err := db.DeleteCollection("temp"); err != nil {
		t.Fatalf("Failed to delete empty collection: %v", err)
	}

	// Verify collection is gone
	collections := db.ListCollections()
	for _, coll := range collections {
		if coll == "temp" {
			t.Error("Collection 'temp' should be deleted")
		}
	}

	// Try to delete non-existent collection
	if err := db.DeleteCollection("nonexistent"); err == nil {
		t.Error("Expected error when deleting non-existent collection")
	} else if err != errors.ErrCollectionNotFound {
		t.Errorf("Expected ErrCollectionNotFound, got %v", err)
	}
}

func TestCollectionNotEmptyDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a collection
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Add a document to the collection
	payload := []byte(`{"name":"user1"}`)
	if err := db.Create("users", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Try to delete non-empty collection
	if err := db.DeleteCollection("users"); err == nil {
		t.Error("Expected error when deleting non-empty collection")
	} else if err != errors.ErrCollectionNotEmpty {
		t.Errorf("Expected ErrCollectionNotEmpty, got %v", err)
	}
}

func TestDocumentIsolationBetweenCollections(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collections
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create users collection: %v", err)
	}
	if err := db.CreateCollection("products"); err != nil {
		t.Fatalf("Failed to create products collection: %v", err)
	}

	// Create document with same ID in different collections
	userPayload := []byte(`{"name":"user1"}`)
	productPayload := []byte(`{"name":"product1"}`)

	if err := db.Create("users", 1, userPayload); err != nil {
		t.Fatalf("Failed to create user document: %v", err)
	}
	if err := db.Create("products", 1, productPayload); err != nil {
		t.Fatalf("Failed to create product document: %v", err)
	}

	// Verify documents are isolated
	userData, err := db.Read("users", 1)
	if err != nil {
		t.Fatalf("Failed to read user: %v", err)
	}
	if string(userData) != string(userPayload) {
		t.Errorf("User data mismatch")
	}

	productData, err := db.Read("products", 1)
	if err != nil {
		t.Fatalf("Failed to read product: %v", err)
	}
	if string(productData) != string(productPayload) {
		t.Errorf("Product data mismatch")
	}
}

func TestCollectionNameValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test invalid collection names
	invalidNames := []string{
		"",                // empty
		"col/lection",     // contains /
		"col.lection",     // contains .
		string([]byte{0}), // null byte
	}

	for _, name := range invalidNames {
		if err := db.CreateCollection(name); err == nil {
			t.Errorf("Expected error for invalid collection name: %q", name)
		}
	}

	// Test valid collection name
	if err := db.CreateCollection("valid_name"); err != nil {
		t.Errorf("Failed to create valid collection: %v", err)
	}
}

func TestDefaultCollectionCannotBeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to delete _default collection
	if err := db.DeleteCollection("_default"); err == nil {
		t.Error("Expected error when deleting _default collection")
	}
}

func TestListCollections(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	log := logger.NewLogger("test", logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memory.NewCaps(1024*1024), memory.NewBufferPool(), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initially should have _default
	collections := db.ListCollections()
	if len(collections) != 1 || collections[0] != "_default" {
		t.Errorf("Expected only _default collection, got %v", collections)
	}

	// Create more collections
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create users: %v", err)
	}
	if err := db.CreateCollection("products"); err != nil {
		t.Fatalf("Failed to create products: %v", err)
	}

	collections = db.ListCollections()
	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}
}
