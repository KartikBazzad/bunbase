package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kartikbazzad/bunbase/buncast/pkg/client"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/handlers"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/limits"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/rpc"
	serverPkg "github.com/kartikbazzad/bunbase/bundoc-server/internal/server"
	"github.com/kartikbazzad/bunbase/bundoc/raft"
)

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func envInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

// extractProjectID returns the project ID from path /v1/projects/{projectID}/...
func extractProjectID(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

type BundocFSM struct {
	Mgr *manager.InstanceManager
}

func (f *BundocFSM) Apply(cmd []byte) interface{} {
	// TODO: Decode command and apply to DB
	// For now just Log
	log.Printf("FSM Apply: %d bytes", len(cmd))
	return nil
}

func main() {
	// Parse Flags
	raftID := flag.String("raft-id", "", "Raft Node ID (e.g., node1)")
	raftPeers := flag.String("peers", "", "Comma-separated list of peer addresses (e.g., node2:4321,node3:4321)")
	port := flag.Int("port", 4321, "TCP Server Port")
	tlsCert := flag.String("tls-cert", "", "Path to TLS server certificate")
	tlsKey := flag.String("tls-key", "", "Path to TLS server private key")
	httpPort := flag.Int("http-port", 8080, "HTTP Server Port")
	rpcAddr := flag.String("rpc-addr", "", "TCP address for internal RPC server (e.g. :8081). If empty, RPC server is disabled.")
	buncastSocket := flag.String("buncast-socket", "", "Optional Buncast IPC socket path for realtime events")
	flag.Parse()

	// Override from environment variable if set
	if val := os.Getenv("BUNDOC_BUNCAST_SOCKET"); val != "" {
		*buncastSocket = val
	}
	if val := os.Getenv("BUNDOC_RPC_ADDR"); val != "" {
		*rpcAddr = val
	}

	addr := fmt.Sprintf(":%d", *port)

	// Create instance manager
	managerOpts := manager.DefaultManagerOptions("./data/projects")
	mgr, err := manager.NewInstanceManager(managerOpts)
	if err != nil {
		log.Fatalf("Failed to create instance manager: %v", err)
	}
	defer mgr.Close()

	// Optional Buncast client for realtime events
	var buncastClient *client.Client
	if *buncastSocket != "" {
		buncastClient = client.New(*buncastSocket)
		log.Printf("Realtime: Buncast client enabled (socket=%s)", *buncastSocket)
	} else {
		log.Printf("Realtime: Buncast client disabled (no -buncast-socket)")
	}

	// Resource limits (cloud mode / DoS protection); 0 = unlimited
	limitsConfig := limits.Config{
		MaxConnectionsPerProject: envInt("BUNDOC_LIMITS_MAX_CONNECTIONS_PER_PROJECT", 0),
		MaxExecutionTimeMs:       envInt("BUNDOC_LIMITS_MAX_EXECUTION_MS", 0),
		MaxScanDocs:              envInt("BUNDOC_LIMITS_MAX_SCAN_DOCS", 0),
		MaxDatabaseSizeBytes:     envInt64("BUNDOC_LIMITS_MAX_DATABASE_SIZE_BYTES", 0),
	}
	concurrencyLimiter := limits.NewConcurrencyLimiter(limitsConfig.MaxConnectionsPerProject)
	if limitsConfig.MaxConnectionsPerProject > 0 {
		log.Printf("Bundoc limits: max_connections_per_project=%d", limitsConfig.MaxConnectionsPerProject)
	}
	if limitsConfig.MaxScanDocs > 0 {
		log.Printf("Bundoc limits: max_scan_docs=%d", limitsConfig.MaxScanDocs)
	}
	if limitsConfig.MaxDatabaseSizeBytes > 0 {
		log.Printf("Bundoc limits: max_database_size_bytes=%d", limitsConfig.MaxDatabaseSizeBytes)
	}

	// Create handlers
	docHandlers := handlers.NewDocumentHandlers(mgr, buncastClient, &limitsConfig)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		stats := mgr.GetStats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","instances":%d,"active":%d}`,
			stats.TotalInstances, stats.ActiveInstances)
	})

	// Document endpoints
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Middleware: Extract Project ID and Auth
		// Simple validation for now - allow /projects or /v1/projects
		if !strings.Contains(r.URL.Path, "/projects/") {
			http.Error(w, "invalid path: must contain /projects/", http.StatusBadRequest)
			return
		}

		// Per-project connection limit (cloud mode / DoS protection)
		projectID := extractProjectID(r.URL.Path)
		if projectID != "" && !concurrencyLimiter.TryAcquire(projectID) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too many concurrent requests for this project"}`))
			return
		}
		if projectID != "" {
			defer concurrencyLimiter.Release(projectID)
		}

		// Routing logic based on suffix
		if strings.HasSuffix(r.URL.Path, "/collections") && r.Method == "POST" {
			docHandlers.HandleCreateCollection(w, r)
			return
		}

		// Index Operations /indexes
		if strings.HasSuffix(r.URL.Path, "/indexes") {
			docHandlers.HandleIndexOperations(w, r)
			return
		}

		// Query Operations /documents/query
		if strings.HasSuffix(r.URL.Path, "/documents/query") && r.Method == "POST" {
			docHandlers.HandleQueryDocuments(w, r)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/documents") {
			if r.Method == "GET" {
				// We need collection name. Path structure: .../collections/{collection}/documents
				// Helper needed to extract collection if valid
				// Current HandleListDocuments extracts it.

				// Wait, default handler routing via strings.HasSuffix is brittle for nested paths.
				// Better to check specific segments or Regex.
				// For MVP, if it ends in /documents, it's list or create.
				// .../documents/{id} is handled below?
			}
		}

		// Rules Operations /collections/{name}/rules
		if strings.HasSuffix(r.URL.Path, "/rules") && r.Method == "PATCH" {
			docHandlers.HandleUpdateRules(w, r)
			return
		}

		// Fallback to specific resource handlers
		if strings.HasSuffix(r.URL.Path, "/collections") {
			if r.Method == "GET" {
				docHandlers.HandleListCollections(w, r)
				return
			} else if r.Method == "POST" {
				docHandlers.HandleCreateCollection(w, r)
				return
			}
		} else if strings.Contains(r.URL.Path, "/collections/") {
			// Document or Collection operations
			// /connections/{name} PATCH -> Update Schema
			// /collections/{name}/documents -> List/Create
			// /collections/{name}/documents/{id} -> Get/Update/Delete

			// We need a better router. But sticking to this switch for now.

			if strings.HasSuffix(r.URL.Path, "/documents") {
				if r.Method == "GET" {
					// Extract collection from path: .../databases/default/collections/{collection}/documents
					path := strings.TrimSuffix(r.URL.Path, "/documents")
					projectID, collection := docHandlers.ParseProjectAndCollectionFromCollectionPath(path)

					// HandleListDocuments args: (w, r, projectID, collection)
					if projectID != "" {
						docHandlers.HandleListDocuments(w, r, projectID, collection)
						return
					}
				} else if r.Method == "POST" {
					docHandlers.HandleCreateDocument(w, r)
					return
				}
			} else {
				// Check for Document ID or Collection ID
				// If path ends with collection name?
				// PATCH /collections/{name}

				// Document ID check:
				// .../documents/{id}
				if strings.Contains(r.URL.Path, "/documents/") {
					if r.Method == "GET" {
						docHandlers.HandleGetDocument(w, r)
					} else if r.Method == "PATCH" {
						docHandlers.HandleUpdateDocument(w, r)
					} else if r.Method == "DELETE" {
						docHandlers.HandleDeleteDocument(w, r)
					}
					return
				}

				// Collection operations
				// PATCH /collections/{name}
				if r.Method == "PATCH" {
					docHandlers.HandleUpdateCollection(w, r)
					return
				}
				// GET /collections/{name}
				if r.Method == "GET" {
					docHandlers.HandleGetCollection(w, r)
					return
				}
				// DELETE /collections/{name}
				if r.Method == "DELETE" {
					docHandlers.HandleDeleteCollection(w, r)
					return
				}
			}
		}

		// Fallback to original switch for now for other methods or unhandled paths
		switch r.Method {
		case "POST":
			// This case is largely handled above for /collections and /documents
			// If it reaches here, it might be an unhandled POST or a specific document POST
			docHandlers.HandleCreateDocument(w, r) // Default to creating a document if not caught by specific paths
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/collections") {
				docHandlers.HandleListCollections(w, r)
			} else {
				docHandlers.HandleGetDocument(w, r)
			}
		case "PATCH":
			docHandlers.HandleUpdateDocument(w, r)
		case "DELETE":
			if strings.Contains(r.URL.Path, "/indexes/") {
				docHandlers.HandleDeleteIndex(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/collections") || strings.Contains(r.URL.Path, "/collections/") && !strings.Contains(r.URL.Path, "/documents/") {
				if strings.Contains(r.URL.Path, "/collections") && !strings.Contains(r.URL.Path, "/documents") {
					docHandlers.HandleDeleteCollection(w, r)
				} else {
					docHandlers.HandleDeleteDocument(w, r)
				}
			} else {
				docHandlers.HandleDeleteDocument(w, r)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Create server (execution limit = write timeout when set)
	writeTimeout := 15 * time.Second
	if limitsConfig.MaxExecutionTimeMs > 0 {
		writeTimeout = time.Duration(limitsConfig.MaxExecutionTimeMs) * time.Millisecond
		log.Printf("Bundoc limits: max_execution_time_ms=%d", limitsConfig.MaxExecutionTimeMs)
	}
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *httpPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: writeTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("🚀 Bundoc HTTP Server starting on %s", server.Addr)
		log.Printf("📊 Health check: http://localhost:8080/health")
		log.Printf("📝 API: http://localhost:8080/v1/projects/{projectId}/databases/(default)/documents/{collection}")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP Server failed: %v", err)
		}
	}()

	// Optional: start internal RPC server for platform (document proxy over TCP)
	var rpcServer *rpc.Server
	if *rpcAddr != "" {
		rpcServer = rpc.NewServer(*rpcAddr, mux)
		if err := rpcServer.Start(); err != nil {
			log.Fatalf("RPC server failed: %v", err)
		}
		defer rpcServer.Stop()
	}

	// Load TLS Config if enabled
	var tlsConfig *tls.Config
	if *tlsCert != "" && *tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to load TLS keys: %v", err)
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	// Start TCP server
	tcpServer := serverPkg.NewTCPServer(addr, mgr, tlsConfig)

	// Initialize Raft if ID is provided
	if *raftID != "" {
		peers := strings.Split(*raftPeers, ",")
		// Filter empty strings if no peers
		if len(peers) == 1 && peers[0] == "" {
			peers = []string{}
		}

		raftCfg := raft.DefaultConfig(*raftID, peers)
		transport := raft.NewTCPTransport()

		// Simple FSM Adapter
		fsm := &BundocFSM{Mgr: mgr}

		raftNode := raft.NewNode(raftCfg, transport, fsm)
		raftNode.Start()
		defer raftNode.Stop()

		tcpServer.SetRaftNode(raftNode)
		log.Printf("⚓️ Raft Node %s started with %d peers", *raftID, len(peers))
	}

	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer tcpServer.Stop()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped")
}
