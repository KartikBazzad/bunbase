package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	serverPkg "github.com/kartikbazzad/bunbase/bundoc-server/internal/server"
	"github.com/kartikbazzad/bunbase/bundoc/client"
	"github.com/kartikbazzad/bunbase/bundoc/security"
)

func TestE2E_NetworkFlow(t *testing.T) {
	// 1. Setup Server
	tmpDir, err := os.MkdirTemp("", "bundoc-e2e")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgrOpts := manager.DefaultManagerOptions(tmpDir)
	mgr, err := manager.NewInstanceManager(mgrOpts)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	// Use a random port (0) or fixed high port
	// port 0 often picks random free port, but we need to know it.
	// TCPServer takes addr string.
	// Let's try finding a free port first.
	// Or just use 4322
	port := 4322
	addr := fmt.Sprintf("localhost:%d", port)

	// Start TCP server
	tcpServer := serverPkg.NewTCPServer(addr, mgr, nil)
	go tcpServer.Start()
	defer tcpServer.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// 1.5 Create User (Backdoor via Manager/DB because client CreateUser isn't implemented)
	// We need to acquire DB for "proj1" to create user there?
	// Security is scoped to DB/Project.
	// Users are in 'admin.users' of the DB.
	// For "proj1".
	db, _, _ := mgr.Acquire("proj1")
	// Note: In real app, we might have global users or per-project.
	// Our SecurityManager is per Database instance.
	// So we need to create user in "proj1".

	// Create "admin" user with RoleReadWrite (Defined in security/types.go)
	err = db.Security.CreateUser("admin", "password123", []security.Role{security.RoleReadWrite})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Client Connect
	cli, err := client.Connect(addr)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	// 2.5 Login
	if err := cli.Login("admin", "password123", "proj1"); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// 3. Insert Data
	col := cli.Database("db1").Collection("users")
	doc := map[string]interface{}{
		"name": "Alice",
		"age":  30,
		"city": "Wonderland",
	}

	// Insert into project "proj1"
	if err := col.Insert("proj1", doc); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// 4. Find Data
	// Query: age > 25
	q := map[string]interface{}{
		"age": map[string]interface{}{"$gt": 25},
	}

	results, err := col.FindQuery("proj1", q)
	if err != nil {
		t.Fatalf("FindQuery failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if name, ok := results[0]["name"]; !ok || name != "Alice" {
		t.Errorf("Expected Alice, got %v", results[0])
	}

	// 5. Test Options (Sort, Limit)
	// Add more docs
	col.Insert("proj1", map[string]interface{}{"name": "Bob", "age": 20})
	col.Insert("proj1", map[string]interface{}{"name": "Charlie", "age": 40})

	// Query All, Sort by Age Desc
	qAll := map[string]interface{}{}
	opts := bundoc.QueryOptions{
		SortField: "age",
		SortDesc:  true,
	}

	resultsSorted, err := col.FindQuery("proj1", qAll, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(resultsSorted) != 3 {
		t.Errorf("Expected 3 results, got %d", len(resultsSorted))
	}
	if resultsSorted[0]["name"] != "Charlie" { // Age 40 (Highest)
		t.Errorf("Sort Failed. Expected Charlie first, got %v", resultsSorted[0]["name"])
	}
	if resultsSorted[2]["name"] != "Bob" { // Age 20 (Lowest)
		t.Errorf("Sort Failed. Expected Bob last, got %v", resultsSorted[2]["name"])
	}
}
