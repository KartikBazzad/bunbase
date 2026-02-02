// Bunder-manager is the instance manager for Bunder KV: one Bunder process per project.
// It exposes an HTTP front that accepts /kv/{project_id}/... and proxies to the project's Bunder instance.
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
)

func main() {
	dataPath := flag.String("data", "./data", "Root data directory for project Bunder instances")
	bunderBin := flag.String("bunder-bin", "bunder", "Path to bunder binary")
	addr := flag.String("addr", ":8085", "HTTP listen address for the manager")
	portBase := flag.Int("port-base", 9000, "First port in pool for Bunder instances")
	portCount := flag.Int("port-count", 1000, "Number of ports in pool")
	flag.Parse()

	opts := manager.DefaultManagerOptions(*dataPath)
	opts.BunderBin = *bunderBin
	opts.PortBase = *portBase
	opts.PortCount = *portCount

	m, err := manager.NewInstanceManager(opts)
	if err != nil {
		log.Fatalf("Failed to create instance manager: %v", err)
	}
	defer m.Close()

	proxy := managerhttp.NewProxyHandler(m)
	mux := http.NewServeMux()
	mux.Handle("/kv/", proxy)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	go func() {
		log.Printf("Bunder manager listening on %s", *addr)
		if err := http.ListenAndServe(*addr, mux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	fmt.Fprintf(os.Stderr, "Bunder manager stopped.\n")
}
