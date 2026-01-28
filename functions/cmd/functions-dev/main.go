package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/gateway"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

// functions-dev: local-only QuickJS-based dev runner.
//
// Example:
//   functions-dev --entry dist/index.js --name hello-world --port 8787
//
// Serves:
//   http://127.0.0.1:8787/functions/hello-world

func main() {
	entry := flag.String("entry", "", "Path to function bundle or entry file (default: dist/index.js or src/index.ts|js)")
	name := flag.String("name", "", "Function name (defaults to directory name)")
	runtime := flag.String("runtime", "bun", "Runtime (bun or quickjs-ng)")
	handler := flag.String("handler", "default", "Handler name")
	port := flag.Int("port", 8787, "HTTP port for dev server")
	flag.Parse()

	// Derive entry path if not provided.
	if *entry == "" {
		candidates := []string{
			"dist/index.js",
			"src/index.ts",
			"src/index.js",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				*entry = c
				break
			}
		}
		if *entry == "" {
			fmt.Fprintln(os.Stderr, "dev runner: could not find an entry file (tried dist/index.js, src/index.ts, src/index.js)")
			os.Exit(1)
		}
	}

	// Normalize entry to absolute path.
	absEntry, err := filepath.Abs(*entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dev runner: failed to resolve entry path: %v\n", err)
		os.Exit(1)
	}

	// Derive name from flag or directory.
	fnName := *name
	if fnName == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "dev runner: failed to get working directory: %v\n", err)
			os.Exit(1)
		}
		fnName = filepath.Base(wd)
	}

	// Prepare dev-scoped config and paths.
	cfg := config.DefaultConfig()
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dev runner: failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	devDir := filepath.Join(wd, ".bunbase", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "dev runner: failed to create dev dir: %v\n", err)
		os.Exit(1)
	}

	cfg.DataDir = devDir
	cfg.Metadata.DBPath = filepath.Join(devDir, "functions.db")
	cfg.Gateway.HTTPPort = *port
	cfg.Gateway.EnableHTTP = true

	log := logger.Default()
	log.SetLevel(logger.LevelDebug)

	log.Info("Starting BunBase Functions Dev...")
	log.Info("Entry: %s", absEntry)
	log.Info("Name: %s", fnName)
	log.Info("Data dir: %s", cfg.DataDir)
	log.Info("HTTP port: %d", cfg.Gateway.HTTPPort)

	// Initialize metadata store (dev DB).
	store, err := metadata.NewStore(cfg.Metadata.DBPath)
	if err != nil {
		log.Error("Failed to initialize metadata store: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize scheduler and router.
	sched := scheduler.NewScheduler(log)
	rtr := router.NewRouter(store, sched, log)

	// Register dev function.
	fnID := "dev-" + fnName
	caps := capabilities.DefaultProfile("") // no project context for dev
	fn, err := store.RegisterFunction(fnID, fnName, *runtime, *handler, caps)
	if err != nil {
		log.Error("Failed to register dev function: %v", err)
		os.Exit(1)
	}

	// Create dev version.
	versionID := uuid.New().String()
	version, err := store.CreateVersion(versionID, fn.ID, "dev", absEntry)
	if err != nil {
		log.Error("Failed to create dev version: %v", err)
		os.Exit(1)
	}

	// Mark function as deployed and set active version.
	if err := store.DeployFunction(uuid.New().String(), fn.ID, version.ID); err != nil {
		log.Error("Failed to deploy dev function: %v", err)
		os.Exit(1)
	}

	// Create pools for this deployed function (reuse main service logic locally).
	if err := createDevPool(store, rtr, cfg, log); err != nil {
		log.Warn("Failed to create dev pool: %v", err)
	}

	// Start HTTP gateway.
	gw := gateway.NewGateway(rtr, sched, &cfg.Gateway, log)
	go func() {
		if err := gw.Start(); err != nil {
			log.Error("Dev HTTP gateway error: %v", err)
		}
	}()

	log.Info("bunbase dev running at http://127.0.0.1:%d/functions/%s", cfg.Gateway.HTTPPort, fnName)

	// Wait for SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutting down dev runner...")
	if err := gw.Stop(); err != nil {
		log.Warn("Error stopping dev gateway: %v", err)
	}
	sched.Stop()
	log.Info("Dev runner stopped")
}

// createDevPool is a small, local variant of createPoolsForDeployedFunctions
// from the main functions binary. It assumes a single deployed function.
func createDevPool(meta *metadata.Store, rtr *router.Router, cfg *config.Config, logr *logger.Logger) error {
	functions, err := meta.ListFunctions()
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}
	if len(functions) == 0 {
		return fmt.Errorf("no functions registered in dev metadata")
	}

	for _, fn := range functions {
		if fn.Status != metadata.FunctionStatusDeployed || fn.ActiveVersionID == "" {
			continue
		}

		version, err := meta.GetVersionByID(fn.ActiveVersionID)
		if err != nil {
			logr.Warn("Failed to get version %s for function %s: %v", fn.ActiveVersionID, fn.ID, err)
			continue
		}

		if _, err := os.Stat(version.BundlePath); err != nil {
			logr.Warn("Bundle not found for function %s: %s (error: %v)", fn.ID, version.BundlePath, err)
			continue
		}

		if _, err := os.ReadFile(version.BundlePath); err != nil {
			logr.Warn("Bundle not readable for function %s: %s (error: %v)", fn.ID, version.BundlePath, err)
			continue
		}

		poolCfg := cfg.Worker
		if fn.Runtime != "" {
			poolCfg.Runtime = fn.Runtime
		}
		if fn.Capabilities != nil {
			poolCfg.Capabilities = fn.Capabilities
		}

		// For QuickJS, QuickJSPath is used directly; worker script is unused.
		runtimeWorkerScript := ""

		p := pool.NewPool(
			fn.ID,
			version.Version,
			version.BundlePath,
			&poolCfg,
			runtimeWorkerScript,
			map[string]string{},
			logr,
		)

		rtr.RegisterPool(fn.ID, p)
		logr.Info("Created dev pool for function %s (version %s)", fn.ID, version.Version)
	}

	return nil
}

