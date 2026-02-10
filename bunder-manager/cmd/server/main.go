// Bunder-manager is the instance manager for Bunder KV: one embedded Bunder instance per project.
// It exposes an HTTP front that accepts /kv/{project_id}/... and routes to the project's embedded Bunder instance.
// It also exposes an RPC server for internal use by platform.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	managerhttp "github.com/kartikbazzad/bunbase/bunder-manager/internal/http"
	"github.com/kartikbazzad/bunbase/bunder-manager/internal/manager"
	"github.com/kartikbazzad/bunbase/bunder-manager/internal/pubsub"
	"github.com/kartikbazzad/bunbase/bunder-manager/internal/rpc"
)

func main() {
	dataPath := flag.String("data", "./data", "Root data directory for project Bunder instances")
	addr := flag.String("addr", ":8080", "HTTP listen address for the manager")
	rpcAddr := flag.String("rpc-addr", "", "TCP address for internal RPC server (e.g. :9091). If empty, RPC server is disabled.")
	flag.Parse()

	// Override from environment variable if set
	if val := os.Getenv("BUNDER_RPC_ADDR"); val != "" {
		*rpcAddr = val
	}

	opts := manager.DefaultManagerOptions(*dataPath)

	m, err := manager.NewInstanceManager(opts)
	if err != nil {
		log.Fatalf("Failed to create instance manager: %v", err)
	}
	defer m.Close()

	buncastSocket := os.Getenv("BUNDER_BUNCAST_SOCKET")
	var publisher *pubsub.Publisher
	if buncastSocket != "" {
		publisher = pubsub.NewPublisher(buncastSocket)
		log.Printf("KV realtime: publishing to Buncast at %s", buncastSocket)
	}

	proxy := managerhttp.NewProxyHandler(m, publisher)
	mux := http.NewServeMux()
	mux.Handle("/kv/", proxy)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	go func() {
		log.Printf("Bunder manager HTTP listening on %s", *addr)
		if err := http.ListenAndServe(*addr, mux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	// Optional: start internal RPC server for platform (KV proxy over TCP)
	var rpcServer *rpc.Server
	if *rpcAddr != "" {
		rpcServer = rpc.NewServer(*rpcAddr, m, mux)
		if err := rpcServer.Start(); err != nil {
			log.Fatalf("RPC server failed: %v", err)
		}
		defer rpcServer.Stop()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	fmt.Fprintf(os.Stderr, "Bunder manager stopped.\n")
}
