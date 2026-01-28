# BunBase Functions

> ⚠️ **Status:** In development. A high-performance serverless execution subsystem for BunBase, built with **Go as the control plane** and **Bun as the JavaScript runtime**.

BunBase Functions is a serverless execution subsystem that manages long-lived Bun workers like a database connection pool. It provides fast, warm execution for JavaScript/TypeScript functions with minimal cold start overhead.

## Features

- **Long-Lived Workers**: One Bun process per function version, handles multiple invocations (amortizes startup cost)
- **Worker Pool Management**: Warm workers ready for instant execution, spawn on demand, terminate idle
- **Multiple Invocation Sources**: HTTP requests, scheduled cron jobs, internal events, CLI invocations
- **Fast Execution**: Warm invocations < 50ms overhead, cold starts < 500ms
- **IPC Interface**: Unix domain socket for API server integration
- **Observability**: Structured logging, metrics aggregation, execution tracing
- **Function Lifecycle**: States from REGISTERED → BUILT → DEPLOYED → WARM → BUSY → IDLE → TERMINATED
- **Bounded Resources**: Memory limits, execution timeouts, concurrency limits

## Non-Goals (Explicitly Out of Scope)

This project will NOT support (v1):
- Multi-tenant isolation (single-tenant only)
- Strong sandboxing (trust local user)
- WASM runtimes (Bun only)
- Edge deployment (local-first)
- Auto-scaling (fixed worker limits)
- Distributed execution (single-node)

If any of these appear, **the project scope has failed**.

## Architecture

```
Client / Event / Cron
        │
        ▼
┌────────────────────┐
│  Functions Gateway │  (Go)
└─────────┬──────────┘
          ▼
┌────────────────────┐
│   Function Router  │  (Go)
└─────────┬──────────┘
          ▼
┌────────────────────┐
│   Scheduler        │  (Go)
└─────────┬──────────┘
          ▼
┌────────────────────┐
│   Worker Pool      │  (Go)
└─────────┬──────────┘
          ▼
┌────────────────────┐
│   Bun Worker       │  (Bun)
└─────────┬──────────┘
          ▼
┌────────────────────┐
│ User Function Code │
└────────────────────┘
```

## Core Concepts

| Concept | Meaning |
|---------|---------|
| **Function** | User-defined JavaScript/TypeScript code with a handler |
| **Function Version** | Immutable snapshot of function code at a point in time |
| **Deployment** | Active version of a function that can receive invocations |
| **Worker Pool** | Collection of Bun processes for a function version |
| **Warm Worker** | Ready Bun process waiting for invocations |
| **Busy Worker** | Bun process currently executing an invocation |
| **Cold Start** | Spawning a new Bun worker when none are warm |
| **Invocation** | Single execution of a function handler |

## Function Lifecycle States

```
REGISTERED → BUILT → DEPLOYED → WARM → BUSY → IDLE → TERMINATED
```

- **REGISTERED**: Function metadata created, no code yet
- **BUILT**: Code bundled and stored, not deployed
- **DEPLOYED**: Active version set, ready for invocations
- **WARM**: Worker process ready and waiting
- **BUSY**: Worker executing an invocation
- **IDLE**: Worker idle, may be terminated after timeout
- **TERMINATED**: Worker process killed

## Quick Start

### Prerequisites

- Go 1.21+
- Bun runtime installed (`bun` in PATH)
- SQLite3 (for metadata storage)

### Build

```bash
cd functions
go build -o functions ./cmd/functions
```

### Run Server

```bash
./functions --data-dir ./data --socket /tmp/functions.sock
```

### Use Go Client

```go
package main

import (
    "fmt"
    "github.com/kartikbazzad/bunbase/functions/pkg/client"
)

func main() {
    cli := client.New("/tmp/functions.sock")
    
    // Register a function
    funcID, err := cli.RegisterFunction("hello-world", "bun", "handler")
    if err != nil {
        panic(err)
    }
    
    // Deploy function version
    versionID, err := cli.DeployFunction(funcID, "v1", "/path/to/bundle.js")
    if err != nil {
        panic(err)
    }
    
    // Invoke function
    result, err := cli.Invoke(funcID, &client.InvokeRequest{
        Method: "POST",
        Path: "/",
        Headers: map[string]string{},
        Body: []byte(`{"name":"Alice"}`),
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Println(string(result.Body)) // Function response
}
```

## Function Handler API

Functions must export a default handler:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
  });
}
```

### Provided Context

- `Request` object (Web API standard)
- Environment variables (via `process.env`)
- Invocation ID (via `X-Invocation-Id` header)

### Not Provided (v1)

- Child processes
- Network outside allowlist (v1: open)
- File system outside temp directory
- Strong isolation

## Invocation Sources

| Source | Example | Normalized To |
|--------|---------|---------------|
| HTTP | REST API, Webhooks | `InvokeRequest` |
| Scheduled | Cron jobs | `InvokeRequest` |
| Internal Events | DB triggers (future) | `InvokeRequest` |
| CLI | `bunbase fn invoke` | `InvokeRequest` |

All invocation types normalize into a single internal `InvokeRequest` structure.

## Storage

| Data | Location | Format |
|------|----------|--------|
| Function Bundles | Filesystem | `data/bundles/{function_id}/{version}/bundle.js` |
| Metadata | SQLite | `data/metadata.db` |
| Logs | SQLite + JSONL | `data/logs.db`, `data/logs/*.jsonl` |
| State | Memory | Worker pools, scheduler queues |

## Performance Targets

| Metric | Target |
|--------|--------|
| Cold Start | < 500ms |
| Warm Execution | < 50ms overhead |
| Concurrent Invocations | 100+ per function |
| Worker Startup | < 200ms |
| Memory per Worker | 50-200MB (configurable) |

## Testing

Run tests:

```bash
# Run all tests
go test ./...

# Run specific test suites
go test ./tests/integration
go test ./tests/failure
go test ./tests/concurrency

# Run benchmarks
go test -bench=. ./tests/benchmarks
```

## Documentation

### Getting Started
- [Getting Started Guide](docs/getting-started.md) - Step-by-step setup and first function
- [Function Development Guide](docs/function-development.md) - How to write and test functions
- [Examples](docs/examples.md) - Example functions and patterns

### Reference
- [Architecture Guide](docs/architecture.md) - Detailed system architecture and component interactions
- [IPC Protocol](docs/protocol.md) - Go ↔ Bun IPC and Unix socket protocol specifications
- [API Reference](docs/api-reference.md) - Complete API documentation
- [Configuration Guide](docs/configuration.md) - All configuration options and settings
- [Deployment Guide](docs/deployment.md) - Function deployment and lifecycle management

### Operations
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions
- [Testing Guide](TESTING.md) - Testing functions and the system

## License

MIT License - see LICENSE file for details.
