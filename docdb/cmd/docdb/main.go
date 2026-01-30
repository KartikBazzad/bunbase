package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/logger"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config file (optional)")
	dataDir := flag.String("data-dir", "./data", "Directory for database files")
	socketPath := flag.String("socket", "/tmp/docdb.sock", "Unix socket path")
	debugMode := flag.Bool("debug", false, "Enable debug mode (request flow logging with requestID)")
	unsafeMultiWriter := flag.Bool("unsafe-multi-writer", false, "Allow multiple scheduler workers (higher throughput; use with -sched-workers)")
	schedWorkers := flag.Int("sched-workers", 0, "Number of scheduler workers (0 = use default; requires -unsafe-multi-writer to be > 1)")
	schedMaxWorkers := flag.Int("sched-max-workers", 0, "Max scheduler workers for cap (0 = use default)")
	replayBudgetMB := flag.Uint64("replay-budget-mb", 0, "Memory budget for WAL replay in MB (0 = use per-DB limit)")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.DataDir = *dataDir
	cfg.IPC.SocketPath = *socketPath
	cfg.IPC.DebugMode = *debugMode
	if *replayBudgetMB > 0 {
		cfg.Memory.ReplayBudgetMB = *replayBudgetMB
	}

	if *unsafeMultiWriter {
		cfg.Sched.UnsafeMultiWriter = true
		if *schedWorkers > 0 {
			cfg.Sched.WorkerCount = *schedWorkers
		}
		if *schedMaxWorkers > 0 {
			cfg.Sched.MaxWorkers = *schedMaxWorkers
		}
		// If UnsafeMultiWriter but no explicit workers, use a sensible multi-worker default
		if cfg.Sched.WorkerCount <= 1 && cfg.Sched.MaxWorkers <= 1 {
			cfg.Sched.WorkerCount = 4
			cfg.Sched.MaxWorkers = 16
		}
	}

	if cfgPath != nil && *cfgPath != "" {
		fmt.Printf("Config file not yet implemented, using defaults\n")
	}

	logr := logger.Default()
	logr.Info("Starting DocDB...")
	logr.Info("Data directory: %s", cfg.DataDir)
	logr.Info("Socket: %s", cfg.IPC.SocketPath)
	if cfg.IPC.DebugMode {
		logr.Info("Debug mode: enabled (request flow logging)")
	}

	server, err := ipc.NewServer(cfg, logr)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logr.Info("Shutting down...")

	if err := server.Stop(); err != nil {
		logr.Error("Error during shutdown: %v", err)
	}

	logr.Info("DocDB stopped")
	os.Exit(0)
}
