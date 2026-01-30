package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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
	schedMaxTotalQueued := flag.Int("sched-max-total-queued", 0, "Global cap on total queued requests across all DBs (0 = disabled; use 400–600 for 20+ DBs)")
	replayBudgetMB := flag.Uint64("replay-budget-mb", 0, "Memory budget for WAL replay in MB (0 = use per-DB limit)")
	debugAddr := flag.String("debug-addr", "", "Enable pprof HTTP server at address (e.g. localhost:6060); empty = disabled")
	walFsyncIntervalMS := flag.Int("wal-fsync-interval-ms", 0, "WAL group commit flush interval in ms (0 = use default 1)")
	walFsyncMaxBatchSize := flag.Int("wal-fsync-max-batch-size", 0, "WAL group commit max records per batch (0 = use default 100)")
	partitionCount := flag.Int("partition-count", 0, "Partitions per database (0 = use default 1; 2–4 for higher write throughput)")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.DataDir = *dataDir
	cfg.WAL.Dir = filepath.Join(cfg.DataDir, "wal")
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
	if *debugAddr != "" {
		cfg.IPC.DebugAddr = *debugAddr
	}
	if *walFsyncIntervalMS > 0 {
		cfg.WAL.Fsync.IntervalMS = *walFsyncIntervalMS
	}
	if *walFsyncMaxBatchSize > 0 {
		cfg.WAL.Fsync.MaxBatchSize = *walFsyncMaxBatchSize
	}
	if *partitionCount > 0 {
		cfg.DB.DefaultPartitionCount = *partitionCount
	}
	if *schedMaxTotalQueued > 0 {
		cfg.Sched.MaxTotalQueued = *schedMaxTotalQueued
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

	if cfg.IPC.DebugAddr != "" {
		runtime.SetMutexProfileFraction(1)
		runtime.SetBlockProfileRate(1)
		go func() {
			logr.Info("pprof enabled at http://%s/debug/pprof/ (mutex and block profiling on)", cfg.IPC.DebugAddr)
			if err := http.ListenAndServe(cfg.IPC.DebugAddr, nil); err != nil {
				logr.Error("pprof server error: %v", err)
			}
		}()
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
