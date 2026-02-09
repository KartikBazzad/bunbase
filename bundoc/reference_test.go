package bundoc

import (
	"errors"
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// --- Schema parsing (parseReferenceRules) ---

func TestParseReferenceRules_Valid(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"author_id": {
				"type": "string",
				"x-bundoc-ref": {
					"collection": "users",
					"field": "_id",
					"on_delete": "set_null"
				}
			},
			"name": { "type": "string" }
		}
	}`
	rules, err := parseReferenceRules("posts", schema)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	r := rules[0]
	if r.SourceCollection != "posts" || r.SourceField != "author_id" || r.TargetCollection != "users" || r.TargetField != "_id" || r.OnDelete != onDeleteSetNull {
		t.Errorf("unexpected rule: %+v", r)
	}
}

func TestParseReferenceRules_DefaultOnDelete(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"user_id": {
				"type": "string",
				"x-bundoc-ref": { "collection": "users" }
			}
		}
	}`
	rules, err := parseReferenceRules("orders", schema)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(rules) != 1 || rules[0].OnDelete != onDeleteSetNull {
		t.Errorf("expected default on_delete set_null, got %+v", rules)
	}
}

func TestParseReferenceRules_Malformed(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{"invalid JSON", `{ "properties": { "f": `},
		{"x-bundoc-ref not object", `{ "properties": { "f": { "x-bundoc-ref": "not-an-object" } } }`},
		{"missing collection", `{ "properties": { "f": { "x-bundoc-ref": { "field": "_id" } } } }`},
		{"empty collection", `{ "properties": { "f": { "x-bundoc-ref": { "collection": "" } } } }`},
		{"field not _id in v1", `{ "properties": { "f": { "x-bundoc-ref": { "collection": "users", "field": "other" } } } }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseReferenceRules("coll", tt.schema)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidReferenceSchema) {
				t.Errorf("expected ErrInvalidReferenceSchema, got %v", err)
			}
		})
	}
}

func TestParseReferenceRules_UnsupportedOnDelete(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"ref": {
				"type": "string",
				"x-bundoc-ref": { "collection": "users", "on_delete": "no_such_action" }
			}
		}
	}`
	_, err := parseReferenceRules("coll", schema)
	if err == nil {
		t.Fatal("expected error for unsupported on_delete")
	}
	if !errors.Is(err, ErrInvalidReferenceSchema) {
		t.Errorf("expected ErrInvalidReferenceSchema, got %v", err)
	}
}

func TestParseReferenceRules_EmptySchema(t *testing.T) {
	rules, err := parseReferenceRules("coll", "")
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected no rules, got %d", len(rules))
	}
}

// --- Write validation (Insert / Update / Patch) ---

func TestReference_InsertSucceedsWithExistingTarget(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "set_null" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "Alice"}); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"}); err != nil {
		t.Fatalf("insert post (expected success): %v", err)
	}
	if err := db.CommitTransaction(txn); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestReference_InsertFailsWhenTargetMissing(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "set_null" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}
	// Do not create "users" or insert any user.

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	err = posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"})
	if err == nil {
		db.RollbackTransaction(txn)
		t.Fatal("expected error when target does not exist")
	}
	if !errors.Is(err, ErrReferenceTargetNotFound) {
		t.Errorf("expected ErrReferenceTargetNotFound, got %v", err)
	}
	db.RollbackTransaction(txn)
}

func TestReference_UpdateFailsWhenTargetMissing(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "set_null" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "A"})
	posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"})
	db.CommitTransaction(txn)

	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	// Update to non-existent user
	err = posts.Update(nil, txn2, "p1", storage.Document{"_id": "p1", "author_id": "nonexistent"})
	if err == nil {
		db.RollbackTransaction(txn2)
		t.Fatal("expected error when updating to missing target")
	}
	if !errors.Is(err, ErrReferenceTargetNotFound) {
		t.Errorf("expected ErrReferenceTargetNotFound, got %v", err)
	}
	db.RollbackTransaction(txn2)
}

func TestReference_PatchFailsWhenTargetMissing(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "set_null" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "A"})
	posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"})
	db.CommitTransaction(txn)

	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	err = posts.Patch(nil, txn2, "p1", map[string]interface{}{"author_id": "nonexistent"})
	if err == nil {
		db.RollbackTransaction(txn2)
		t.Fatal("expected error when patching to missing target")
	}
	if !errors.Is(err, ErrReferenceTargetNotFound) {
		t.Errorf("expected ErrReferenceTargetNotFound, got %v", err)
	}
	db.RollbackTransaction(txn2)
}

// --- Delete policy: restrict ---

func TestReference_DeleteRestrictBlocksWhenDependentsExist(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "restrict" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "A"})
	posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"})
	db.CommitTransaction(txn)

	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	err = users.Delete(nil, txn2, "u1")
	if err == nil {
		db.RollbackTransaction(txn2)
		t.Fatal("expected restrict to block delete when dependents exist")
	}
	if !errors.Is(err, ErrReferenceRestrictViolation) {
		t.Errorf("expected ErrReferenceRestrictViolation, got %v", err)
	}
	db.RollbackTransaction(txn2)
}

// --- Delete policy: set_null ---

func TestReference_DeleteSetNullNullsDependentFields(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	// Schema allows null so set_null patch is valid
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": ["string", "null"], "x-bundoc-ref": { "collection": "users", "on_delete": "set_null" } },
			"title": { "type": "string" }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "A"})
	posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1", "title": "Post"})
	db.CommitTransaction(txn)

	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := users.Delete(nil, txn2, "u1"); err != nil {
		db.RollbackTransaction(txn2)
		t.Fatalf("delete user (set_null): %v", err)
	}
	if err := db.CommitTransaction(txn2); err != nil {
		t.Fatalf("commit: %v", err)
	}

	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	doc, err := posts.FindByID(nil, txn3, "p1")
	db.RollbackTransaction(txn3)
	if err != nil {
		t.Fatalf("find post: %v", err)
	}
	if doc["author_id"] != nil {
		t.Errorf("expected author_id to be null after set_null, got %v", doc["author_id"])
	}
	if doc["title"] != "Post" {
		t.Errorf("expected title unchanged, got %v", doc["title"])
	}
}

// --- Delete policy: cascade ---

func TestReference_DeleteCascadeDeletesDependents(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	users, _ := db.CreateCollection("users")
	posts, _ := db.CreateCollection("posts")
	schema := `{
		"type": "object",
		"properties": {
			"author_id": { "type": "string", "x-bundoc-ref": { "collection": "users", "on_delete": "cascade" } }
		}
	}`
	if err := posts.SetSchema(schema); err != nil {
		t.Fatalf("set schema: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	users.Insert(nil, txn, storage.Document{"_id": "u1", "name": "A"})
	posts.Insert(nil, txn, storage.Document{"_id": "p1", "author_id": "u1"})
	posts.Insert(nil, txn, storage.Document{"_id": "p2", "author_id": "u1"})
	db.CommitTransaction(txn)

	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := users.Delete(nil, txn2, "u1"); err != nil {
		db.RollbackTransaction(txn2)
		t.Fatalf("delete user (cascade): %v", err)
	}
	if err := db.CommitTransaction(txn2); err != nil {
		t.Fatalf("commit: %v", err)
	}

	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	_, err1 := users.FindByID(nil, txn3, "u1")
	_, err2 := posts.FindByID(nil, txn3, "p1")
	_, err3 := posts.FindByID(nil, txn3, "p2")
	db.RollbackTransaction(txn3)
	if err1 == nil {
		t.Error("user u1 should be deleted")
	}
	if err2 == nil {
		t.Error("post p1 should be cascade-deleted")
	}
	if err3 == nil {
		t.Error("post p2 should be cascade-deleted")
	}
}

// --- Cycle guard: cascade does not infinite loop ---

func TestReference_CascadeCycleGuard(t *testing.T) {
	tmpdir := t.TempDir()
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// A -> B -> A (cycle). Each references the other with cascade.
	// We only define one direction in schema: e.g. B references A. So when we delete A, cascade deletes B. No cycle in schema.
	// For a real cycle we'd need A refs B and B refs A. Then deleting A would cascade to B, and deleting B would cascade to A.
	// With visited set, when we process B's delete we try to cascade to A, but A is already in visited so we skip (already deleted).
	collA, _ := db.CreateCollection("a")
	collB, _ := db.CreateCollection("b")
	schemaA := `{
		"type": "object",
		"properties": {
			"ref_b": { "type": "string", "x-bundoc-ref": { "collection": "b", "on_delete": "cascade" } }
		}
	}`
	schemaB := `{
		"type": "object",
		"properties": {
			"ref_a": { "type": "string", "x-bundoc-ref": { "collection": "a", "on_delete": "cascade" } }
		}
	}`
	if err := collA.SetSchema(schemaA); err != nil {
		t.Fatalf("set schema A: %v", err)
	}
	if err := collB.SetSchema(schemaB); err != nil {
		t.Fatalf("set schema B: %v", err)
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	collA.Insert(nil, txn, storage.Document{"_id": "a1", "ref_b": "b1"})
	collB.Insert(nil, txn, storage.Document{"_id": "b1", "ref_a": "a1"})
	db.CommitTransaction(txn)

	// Delete a1: cascade deletes b1 (which has ref_a a1). When deleting b1, applyOnDeletePolicies for "b" would find ref_a -> a.
	// So we'd try to cascade delete a1 again; visited set should prevent re-entering and we just proceed. No infinite loop.
	txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
	err = collA.Delete(nil, txn2, "a1")
	if err != nil {
		db.RollbackTransaction(txn2)
		t.Fatalf("delete a1 (cycle): %v", err)
	}
	if err := db.CommitTransaction(txn2); err != nil {
		t.Fatalf("commit: %v", err)
	}

	txn3, _ := db.BeginTransaction(mvcc.ReadCommitted)
	_, e1 := collA.FindByID(nil, txn3, "a1")
	_, e2 := collB.FindByID(nil, txn3, "b1")
	db.RollbackTransaction(txn3)
	if e1 == nil {
		t.Error("a1 should be deleted")
	}
	if e2 == nil {
		t.Error("b1 should be cascade-deleted")
	}
}

// --- Backward compatibility: collections without references unchanged ---

func TestReference_NoReferencesUnchanged(t *testing.T) {
	tmpdir := t.TempDir()
	defer os.RemoveAll(tmpdir)
	opts := DefaultOptions(tmpdir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	coll, _ := db.CreateCollection("users")
	// No SetSchema with x-bundoc-ref
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := coll.Insert(nil, txn, storage.Document{"_id": "u1", "name": "Alice"}); err != nil {
		t.Fatalf("insert without refs: %v", err)
	}
	if err := db.CommitTransaction(txn); err != nil {
		t.Fatalf("commit: %v", err)
	}
}
