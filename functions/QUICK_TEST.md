# Quick Test Guide

## Step 1: Verify Server is Running

```bash
# Test health endpoint
curl http://localhost:8080/health

# Should return: OK
```

## Step 2: Create a Test Function

Run the setup script:

```bash
./scripts/setup-test-function.sh
```

This will:
1. Build the example function
2. Store it in `data/bundles/test-func/v1/bundle.js`
3. Register it in the database

## Step 3: Create Worker Pool (Required)

The pool needs to be created before invocation. You can do this by:

**Option A: Modify main.go** to auto-create pools (recommended for testing)

**Option B: Use a simple Go program** to create the pool:

```go
// create-pool.go
package main

import (
	"os"
	"path/filepath"
	
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

func main() {
	cfg := config.DefaultConfig()
	logr := logger.Default()
	
	// Open metadata store
	meta, _ := metadata.NewStore(cfg.Metadata.DBPath)
	defer meta.Close()
	
	// Get function
	fn, _ := meta.GetFunctionByName("test-func")
	
	// Get version
	versions, _ := meta.GetVersionsByFunctionID(fn.ID)
	version := versions[0]
	
	// Get worker script path
	workerScript, _ := filepath.Abs("worker/worker.ts")
	
	// Create pool
	p := pool.NewPool(
		fn.ID,
		version.Version,
		version.BundlePath,
		&cfg.Worker,
		workerScript,
		map[string]string{}, // env vars
		logr,
	)
	
	// Create router and scheduler
	sched := scheduler.NewScheduler(logr)
	rtr := router.NewRouter(meta, sched, logr)
	
	// Register pool
	rtr.RegisterPool(fn.ID, p)
	
	logr.Info("Pool created for function %s", fn.ID)
	
	// Keep running
	select {}
}
```

Run it: `go run create-pool.go` (in a separate terminal)

## Step 4: Invoke the Function

```bash
curl 'http://localhost:8080/functions/test-func?name=Alice'
```

Expected response:
```json
{
  "message": "Hello, Alice!",
  "timestamp": "2026-01-27T...",
  "method": "GET",
  "path": "/functions/test-func"
}
```

## Troubleshooting

**"Function not found"**
- Check database: `sqlite3 data/metadata.db "SELECT * FROM functions;"`
- Make sure function is registered

**"Function not deployed"**
- Check status: `sqlite3 data/metadata.db "SELECT id, name, status FROM functions;"`
- Should be "deployed"

**"Function has no active pool"**
- Pool needs to be created (see Step 3)
- Check if pool creation program is running

**Worker spawn errors**
- Check Bun is installed: `which bun`
- Check worker script exists: `ls worker/worker.ts`
- Check bundle exists: `ls data/bundles/test-func/v1/bundle.js`
