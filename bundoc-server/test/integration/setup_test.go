package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc-server/internal/handlers"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
)

// TestMain manages the lifecycle of the integration test suite
func TestMain(m *testing.M) {
	// 1. Setup Environment
	tmpDir, err := os.MkdirTemp("", "bundoc-integration-server")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Initialize Server Components
	mgrOpts := manager.DefaultManagerOptions(tmpDir)
	mgr, err := manager.NewInstanceManager(mgrOpts)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	docHandlers := handlers.NewDocumentHandlers(mgr)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Document endpoints (Copied from main.go)
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch r.Method {
		case "POST":
			docHandlers.HandleCreateDocument(w, r)
		case "GET":
			docHandlers.HandleGetDocument(w, r)
		case "PATCH":
			docHandlers.HandleUpdateDocument(w, r)
		case "DELETE":
			docHandlers.HandleDeleteDocument(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 3. Start Server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Test Server failed: %v", err)
		}
	}()

	// Wait for server to start
	if !waitForServer("http://localhost:8080/health") {
		log.Fatalf("Server failed to start")
	}

	fmt.Println("🚀 Integration Test Server running on :8080")

	// 4. Run Tests
	exitCode := m.Run()

	// 5. Cleanup
	server.Shutdown(context.Background())
	os.Exit(exitCode)
}

func waitForServer(url string) bool {
	for i := 0; i < 20; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
