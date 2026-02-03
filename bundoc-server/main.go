package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flag"
	"strings"

	"github.com/kartikbazzad/bunbase/bundoc-server/internal/handlers"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	serverPkg "github.com/kartikbazzad/bunbase/bundoc-server/internal/server"
	"github.com/kartikbazzad/bunbase/bundoc/raft"
)

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
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)

	// Create instance manager
	managerOpts := manager.DefaultManagerOptions("./data/projects")
	mgr, err := manager.NewInstanceManager(managerOpts)
	if err != nil {
		log.Fatalf("Failed to create instance manager: %v", err)
	}
	defer mgr.Close()

	// Create handlers
	docHandlers := handlers.NewDocumentHandlers(mgr)

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

		// Route based on method and path
		switch r.Method {
		case "POST":
			if strings.HasSuffix(r.URL.Path, "/collections") {
				docHandlers.HandleCreateCollection(w, r)
			} else {
				docHandlers.HandleCreateDocument(w, r)
			}
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/collections") {
				docHandlers.HandleListCollections(w, r)
			} else {
				docHandlers.HandleGetDocument(w, r)
			}
		case "PATCH":
			docHandlers.HandleUpdateDocument(w, r)
		case "DELETE":
			docHandlers.HandleDeleteDocument(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *httpPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	// Start HTTP server in goroutine
	go func() {
		log.Printf("🚀 Bundoc HTTP Server starting on %s", server.Addr)
		log.Printf("📊 Health check: http://localhost:8080/health")
		log.Printf("📝 API: http://localhost:8080/v1/projects/{projectId}/databases/(default)/documents/{collection}")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP Server failed: %v", err)
		}
	}()

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
