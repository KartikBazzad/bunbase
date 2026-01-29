package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/pool"
	"github.com/kartikbazzad/docdb/internal/types"
)

// TestShellRejectsRawPrefix verifies shell parser rejects raw: prefix
func TestShellRejectsRawPrefix(t *testing.T) {
	_, err := parser.DecodePayload("raw:somebinarydata")
	if err != errors.ErrInvalidJSON {
		t.Errorf("expected ErrInvalidJSON, got: %v", err)
	}
}

// TestShellRejectsHexPrefix verifies shell parser rejects hex: prefix
func TestShellRejectsHexPrefix(t *testing.T) {
	_, err := parser.DecodePayload("hex:48656c6c6f")
	if err != errors.ErrInvalidJSON {
		t.Errorf("expected ErrInvalidJSON, got: %v", err)
	}
}

// TestShellAcceptsValidJSON verifies shell parser accepts valid JSON
func TestShellAcceptsValidJSON(t *testing.T) {
	payload, err := parser.DecodePayload(`{"name":"Alice","age":30}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var v map[string]interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Errorf("payload is not valid JSON: %v", err)
	}
}

// TestIPCValidatesJSON verifies IPC handler validates JSON
func TestIPCValidatesJSON(t *testing.T) {
	p, tmpDir, cleanup := setupTestPool(t)
	defer cleanup()

	dbID, err := p.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Try to create document with invalid JSON
	req := &pool.Request{
		DBID:     dbID,
		DocID:    1,
		OpType:   types.OpCreate,
		Payload:  []byte("not json"),
		Response: make(chan pool.Response, 1),
	}
	p.Execute(req)

	resp := <-req.Response
	if resp.Error != errors.ErrInvalidJSON {
		t.Errorf("expected ErrInvalidJSON, got: %v", resp.Error)
	}

	// Verify WAL was not corrupted
	walPath := filepath.Join(tmpDir, "wal", "testdb.wal")
	walInfo, err := os.Stat(walPath)
	if err != nil {
		t.Fatalf("failed to stat WAL: %v", err)
	}

	// WAL should be empty or only contain metadata
	if walInfo.Size() > 1024 {
		t.Errorf("WAL size too large, possibly corrupted: %d bytes", walInfo.Size())
	}
}

// TestWALUnchangedOnInvalidInput verifies WAL is unchanged when invalid input is rejected
func TestWALUnchangedOnInvalidInput(t *testing.T) {
	p, tmpDir, cleanup := setupTestPool(t)
	defer cleanup()

	dbID, err := p.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Get initial WAL size
	walPath := filepath.Join(tmpDir, "wal", "testdb.wal")
	initialSize := getWALSize(t, walPath)

	// Try to create multiple invalid documents
	invalidPayloads := [][]byte{
		[]byte("not json"),
		[]byte(`{invalid`),
		[]byte(`"`),
		[]byte("raw:binary"),
		[]byte("hex:deadbeef"),
	}

	const coll = "_default"
	for _, payload := range invalidPayloads {
		err = db.Create(coll, 1, payload)
		if err == nil {
			t.Errorf("expected error for invalid payload: %s", string(payload))
		}
	}

	// Verify WAL size hasn't changed
	finalSize := getWALSize(t, walPath)
	if finalSize != initialSize {
		t.Errorf("WAL size changed from %d to %d, WAL may be corrupted", initialSize, finalSize)
	}
}

// TestEngineValidatesJSONBeforeWAL verifies engine validates before writing to WAL
func TestEngineValidatesJSONBeforeWAL(t *testing.T) {
	p, tmpDir, cleanup := setupTestPool(t)
	defer cleanup()

	dbID, err := p.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create a valid document first
	const coll = "_default"
	validPayload := []byte(`{"name":"Alice","age":30}`)
	err = db.Create(coll, 1, validPayload)
	if err != nil {
		t.Fatalf("failed to create valid document: %v", err)
	}

	// Get WAL size after valid write
	walPath := filepath.Join(tmpDir, "wal", "testdb.wal")
	sizeAfterValid := getWALSize(t, walPath)

	// Try to create invalid document
	invalidPayload := []byte("not json")
	err = db.Create(coll, 2, invalidPayload)
	if err != errors.ErrInvalidJSON {
		t.Errorf("expected ErrInvalidJSON, got: %v", err)
	}

	// Verify WAL size hasn't changed
	sizeAfterInvalid := getWALSize(t, walPath)
	if sizeAfterInvalid != sizeAfterValid {
		t.Errorf("WAL size changed from %d to %d after invalid input", sizeAfterValid, sizeAfterInvalid)
	}
}

func getWALSize(t *testing.T, walPath string) int64 {
	info, err := os.Stat(walPath)
	if err != nil {
		t.Fatalf("failed to stat WAL: %v", err)
	}
	return info.Size()
}
