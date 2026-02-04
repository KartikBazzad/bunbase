package bundoc

import (
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
)

func TestPatchDeepMerge(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	opts := DefaultOptions(tmpDir)
	db, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatal(err)
	}

	// Insert initial doc
	doc := map[string]interface{}{
		"_id": "user1",
		"settings": map[string]interface{}{
			"theme": "light",
			"notifications": map[string]interface{}{
				"email": true,
				"push":  true,
			},
		},
		"active": true,
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := coll.Insert(nil, txn, doc); err != nil {
		t.Fatal(err)
	}
	db.CommitTransaction(txn)

	// Apply Patch: Update nested field + top level field
	patch := map[string]interface{}{
		"settings.notifications.email": false,
		"settings.theme":               "dark",
		"active":                       false,
		"new_field":                    "ok",
	}

	txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
	if err := coll.Patch(nil, txn, "user1", patch); err != nil {
		t.Fatalf("Patch failed: %v", err)
	}
	db.CommitTransaction(txn)

	// Verify
	txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
	readDoc, err := coll.FindByID(nil, txn, "user1")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Check nested update
	settings := readDoc["settings"].(map[string]interface{})
	notifs := settings["notifications"].(map[string]interface{})

	if notifs["email"] != false {
		t.Errorf("Expected email=false, got %v", notifs["email"])
	}
	if notifs["push"] != true { // Should be preserved
		t.Errorf("Expected push=true (preserved), got %v", notifs["push"])
	}

	// 2. Check sibling update
	if settings["theme"] != "dark" {
		t.Errorf("Expected theme=dark, got %v", settings["theme"])
	}

	// 3. Check top level update
	if readDoc["active"] != false {
		t.Errorf("Expected active=false, got %v", readDoc["active"])
	}

	// 4. Check new field
	if readDoc["new_field"] != "ok" {
		t.Errorf("Expected new_field=ok, got %v", readDoc["new_field"])
	}

	// 5. Test Field Deletion via $unset and Null Preservation
	patch2 := map[string]interface{}{
		"should_be_null": nil, // Should be preserved as nil
		"$unset": map[string]interface{}{
			"new_field":      "", // Should be deleted
			"settings.theme": "", // Should be deleted
		},
	}

	txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
	if err := coll.Patch(nil, txn, "user1", patch2); err != nil {
		t.Fatalf("Patch 2 failed: %v", err)
	}
	db.CommitTransaction(txn)

	txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
	readDoc2, _ := coll.FindByID(nil, txn, "user1")

	// Check Null Preservation
	val, ok := readDoc2["should_be_null"]
	if !ok {
		// Nil value field should exist
		// Wait, simple map retrieval returns nil, false if key missing.
		// If key exists and value is nil, it returns nil, true?
		// In Go map, yes.
		t.Error("Expected should_be_null to exist")
	}
	if val != nil {
		t.Errorf("Expected should_be_null=nil, got %v", val)
	}

	// Check Deletions
	if _, exists := readDoc2["new_field"]; exists {
		t.Error("Expected new_field to be deleted")
	}

	settings2 := readDoc2["settings"].(map[string]interface{})
	if _, exists := settings2["theme"]; exists {
		t.Error("Expected settings.theme to be deleted")
	}
}
