// Bunder server is the main entrypoint for the Bunder Redis-like KV database.
// It parses flags (data path, TCP/HTTP addresses, shards, etc.), starts the TCP and HTTP
// listeners, and runs until SIGINT/SIGTERM; then it closes the server and KV store.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kartikbazzad/bunbase/bunder/internal/config"
	"github.com/kartikbazzad/bunbase/bunder/internal/server"
)

func main() {
	cfg := config.Default()
	if err := cfg.ParseFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	srv, err := server.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Bunder server listening on %s (data: %s)\n", cfg.ListenAddr, cfg.DataPath)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	if err := srv.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Bunder stopped.")
}
