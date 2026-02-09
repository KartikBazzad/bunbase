package integration

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc-server/internal/handlers"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
)

// serverURL is set by TestMain after starting the server on a dynamic port.
var serverURL string

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

	// Integration tests do not require Buncast; pass nil client for now.
	docHandlers := handlers.NewDocumentHandlers(mgr, nil)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API endpoints (routing aligned with main.go for collections + documents)
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimSuffix(r.URL.Path, "/")
		// POST .../collections -> CreateCollection
		if strings.HasSuffix(path, "/collections") && r.Method == "POST" {
			docHandlers.HandleCreateCollection(w, r)
			return
		}
		// PATCH .../collections/{name} (no /documents/ in path) -> UpdateCollection
		if strings.Contains(path, "/collections/") && !strings.Contains(path, "/documents/") && r.Method == "PATCH" {
			docHandlers.HandleUpdateCollection(w, r)
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

	// 3. Start Server on a dynamic port to avoid 8080 conflicts
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	serverURL = fmt.Sprintf("http://localhost:%d", port)
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Test Server failed: %v", err)
		}
	}()

	// Wait for server to start
	if !waitForServer(serverURL + "/health") {
		log.Fatalf("Server failed to start")
	}

	fmt.Printf("🚀 Integration Test Server running on %s\n", serverURL)

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
