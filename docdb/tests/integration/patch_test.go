package integration

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
)

func TestPatchSetOperation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create initial document
	initialPayload := []byte(`{"name":"Alice","age":30}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch: set name field
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/name", Value: "Bob"},
	}
	if err := db.Patch("_default", 1, patchOps); err != nil {
		t.Fatalf("Failed to patch document: %v", err)
	}

	// Verify patch
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	if doc["name"] != "Bob" {
		t.Errorf("Expected name='Bob', got %v", doc["name"])
	}
	if doc["age"] != float64(30) {
		t.Errorf("Expected age=30, got %v", doc["age"])
	}
}

func TestPatchSetNestedField(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document with nested structure
	initialPayload := []byte(`{"user":{"name":"Alice","address":{"city":"NYC"}}}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch: set nested field
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/user/address/city", Value: "LA"},
	}
	if err := db.Patch("_default", 1, patchOps); err != nil {
		t.Fatalf("Failed to patch document: %v", err)
	}

	// Verify patch
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	user := doc["user"].(map[string]interface{})
	address := user["address"].(map[string]interface{})
	if address["city"] != "LA" {
		t.Errorf("Expected city='LA', got %v", address["city"])
	}
}

func TestPatchDeleteOperation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document
	initialPayload := []byte(`{"name":"Alice","age":30,"email":"alice@example.com"}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch: delete email field
	patchOps := []types.PatchOperation{
		{Op: "delete", Path: "/email"},
	}
	if err := db.Patch("_default", 1, patchOps); err != nil {
		t.Fatalf("Failed to patch document: %v", err)
	}

	// Verify patch
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	if _, exists := doc["email"]; exists {
		t.Error("Email field should be deleted")
	}
	if doc["name"] != "Alice" {
		t.Errorf("Name should still be Alice, got %v", doc["name"])
	}
}

func TestPatchInsertOperation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document with array
	initialPayload := []byte(`{"tags":["tag1","tag2"]}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch: insert at index 1
	patchOps := []types.PatchOperation{
		{Op: "insert", Path: "/tags/1", Value: "tag1.5"},
	}
	if err := db.Patch("_default", 1, patchOps); err != nil {
		t.Fatalf("Failed to patch document: %v", err)
	}

	// Verify patch
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	tags := doc["tags"].([]interface{})
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}
	if tags[1] != "tag1.5" {
		t.Errorf("Expected tag1.5 at index 1, got %v", tags[1])
	}
}

func TestPatchMultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document
	initialPayload := []byte(`{"name":"Alice","age":30}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch: multiple operations (atomic)
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/name", Value: "Bob"},
		{Op: "set", Path: "/age", Value: 31},
		{Op: "set", Path: "/email", Value: "bob@example.com"},
	}
	if err := db.Patch("_default", 1, patchOps); err != nil {
		t.Fatalf("Failed to patch document: %v", err)
	}

	// Verify all patches applied
	data, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	if doc["name"] != "Bob" {
		t.Errorf("Expected name='Bob', got %v", doc["name"])
	}
	if doc["age"] != float64(31) {
		t.Errorf("Expected age=31, got %v", doc["age"])
	}
	if doc["email"] != "bob@example.com" {
		t.Errorf("Expected email='bob@example.com', got %v", doc["email"])
	}
}

func TestPatchInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document
	initialPayload := []byte(`{"name":"Alice"}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Patch with invalid path
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/nonexistent", Value: "value"},
	}
	if err := db.Patch("_default", 1, patchOps); err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestPatchNonObjectDocument(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create non-object document (array)
	initialPayload := []byte(`["item1","item2"]`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Try to patch array (should fail)
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/name", Value: "value"},
	}
	if err := db.Patch("_default", 1, patchOps); err == nil {
		t.Error("Expected error when patching non-object document")
	} else if err != errors.ErrNotJSONObject {
		t.Errorf("Expected ErrNotJSONObject, got %v", err)
	}
}

func TestPatchDocumentNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to patch non-existent document
	patchOps := []types.PatchOperation{
		{Op: "set", Path: "/name", Value: "value"},
	}
	if err := db.Patch("_default", 999, patchOps); err == nil {
		t.Error("Expected error when patching non-existent document")
	}
}

func TestPatchEmptyOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, memory.NewBufferPool(cfg.Memory.BufferSizes), log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document
	initialPayload := []byte(`{"name":"Alice"}`)
	if err := db.Create("_default", 1, initialPayload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Try to patch with empty operations
	patchOps := []types.PatchOperation{}
	if err := db.Patch("_default", 1, patchOps); err == nil {
		t.Error("Expected error for empty patch operations")
	} else if err != errors.ErrInvalidPatch {
		t.Errorf("Expected ErrInvalidPatch, got %v", err)
	}
}
