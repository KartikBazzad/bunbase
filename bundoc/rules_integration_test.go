package bundoc

import (
	"os"
	"testing"

	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/rules"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

func TestRulesIntegration(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "bundoc_rules_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	db, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	colName := "posts"
	coll, err := db.CreateCollection(colName)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Set Rules
	// Rule: Allow read/list if true (public).
	// Rule: Allow create if user is authenticated (auth != nil).
	// Rule: Allow update/delete if user owns the document (resource.data.owner == request.auth.uid).
	rulesMap := map[string]string{
		"read":   "true",
		"list":   "true",
		"create": "request.auth != null",
		"update": "resource.data.owner == request.auth.uid",
		"delete": "resource.data.owner == request.auth.uid",
	}
	if err := coll.SetRules(rulesMap); err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}

	// 2. Test Scenarios

	// Scenario A: Unauthenticated Create (Should Fail)
	t.Run("Unauthenticated Create", func(t *testing.T) {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		defer db.RollbackTransaction(txn)

		doc := storage.Document{"_id": "doc1", "title": "Test", "owner": "user1"}
		err := coll.Insert(nil, txn, doc) // Auth is nil
		if err == nil {
			t.Error("Expected error for unauthenticated create, got nil")
		}
	})

	// Scenario B: Authenticated Create (Should Succeed)
	user1Auth := &rules.AuthContext{UID: "user1"}
	t.Run("Authenticated Create", func(t *testing.T) {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		defer db.CommitTransaction(txn)

		doc := storage.Document{"_id": "doc1", "title": "Test", "owner": "user1"}
		err := coll.Insert(user1Auth, txn, doc)
		if err != nil {
			t.Errorf("Expected success for authenticated create, got: %v", err)
		}
	})

	// Scenario C: Unauthorized Update (User2 trying to update User1's doc) (Should Fail)
	user2Auth := &rules.AuthContext{UID: "user2"}
	t.Run("Unauthorized Update", func(t *testing.T) {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		defer db.RollbackTransaction(txn)

		updates := map[string]interface{}{"title": "Hacked"}
		err := coll.Patch(user2Auth, txn, "doc1", updates)
		if err == nil {
			t.Error("Expected error for unauthorized update, got nil")
		}
	})

	// Scenario D: Authorized Update (User1 updating User1's doc) (Should Succeed)
	t.Run("Authorized Update", func(t *testing.T) {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		defer db.CommitTransaction(txn)

		updates := map[string]interface{}{"title": "Updated Title"}
		err := coll.Patch(user1Auth, txn, "doc1", updates)
		if err != nil {
			t.Errorf("Expected success for authorized update, got: %v", err)
		}

		// Verify update check
		doc, _ := coll.FindByID(nil, txn, "doc1") // Read is public
		if doc["title"] != "Updated Title" {
			t.Errorf("Update not applied properly")
		}
	})

	// Scenario E: Admin Bypass (Should always succeed)
	adminAuth := &rules.AuthContext{IsAdmin: true}
	t.Run("Admin Bypass", func(t *testing.T) {
		txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
		defer db.CommitTransaction(txn)

		// Admin deleting doc1 (even if owner rule exists, admin bypasses)
		// Wait, Delete rule is "resource.data.owner == request.auth.uid"
		// If Admin, it skips rule eval.
		err := coll.Delete(adminAuth, txn, "doc1")
		if err != nil {
			t.Errorf("Expected success for admin delete, got: %v", err)
		}
	})
}
