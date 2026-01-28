# Getting Started with BunBase Functions

This guide will walk you through setting up and using BunBase Functions from scratch.

## Prerequisites

Before you begin, ensure you have:

- **Go 1.21+** installed ([download](https://go.dev/dl/))
- **Bun runtime** installed ([installation guide](https://bun.sh/docs/installation))
- **SQLite3** (usually pre-installed on macOS/Linux)
- Basic familiarity with TypeScript/JavaScript

Verify your setup:

```bash
go version    # Should show 1.21 or higher
bun --version # Should show Bun version
sqlite3 --version
```

## Installation

### Build from Source

```bash
cd functions
go build -o functions ./cmd/functions
```

This creates a `functions` binary in the current directory.

## Running the Server

Start the Functions service:

```bash
./functions --data-dir ./data --socket /tmp/functions.sock --log-level info
```

Options:
- `--data-dir`: Directory for storing bundles, metadata, and logs (default: `./data`)
- `--socket`: Unix domain socket path for IPC (default: `/tmp/functions.sock`)
- `--log-level`: Logging level: `debug`, `info`, `warn`, `error` (default: `info`)
- `--http-port`: HTTP gateway port (default: `8080`)

The server will:
- Create necessary directories
- Initialize SQLite metadata database
- Start HTTP gateway on port 8080
- Start IPC server on the Unix socket

Verify it's running:

```bash
curl http://localhost:8080/health
# Should return: OK
```

## Your First Function

### 1. Create a Function

Create a simple TypeScript function:

```typescript
// examples/my-first-function.ts
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
    method: req.method,
    path: url.pathname,
  });
}
```

### 2. Build the Function

Bundle your function:

```bash
bun build examples/my-first-function.ts \
  --outdir ./data/bundles/my-function/v1 \
  --target bun \
  --outfile bundle.js
```

This creates `./data/bundles/my-function/v1/bundle.js`.

### 3. Register and Deploy

Use the setup script or manually register:

```bash
# Using the setup script (recommended)
./scripts/setup-test-function.sh

# Or manually via SQLite (see TESTING.md)
```

### 4. Invoke Your Function

```bash
curl -X POST "http://localhost:8080/functions/my-function?name=Alice"
```

Expected response:

```json
{
  "message": "Hello, Alice!",
  "timestamp": "2026-01-27T22:00:00.000Z",
  "method": "POST",
  "path": "/functions/my-function"
}
```

## Understanding the Flow

1. **HTTP Request** → Gateway receives request
2. **Routing** → Router resolves function name to worker pool
3. **Scheduling** → Scheduler acquires a worker (warm or spawns new)
4. **Execution** → Worker executes your function handler
5. **Response** → Result flows back through the stack

## Next Steps

- Read the [Function Development Guide](function-development.md) to learn best practices
- Explore [Examples](../examples/) for more complex use cases
- Check the [API Reference](api-reference.md) for detailed API documentation
- Review [Configuration Guide](configuration.md) for advanced settings

## Common Issues

### Function Not Found

If you get "Function not found":
- Verify the function is registered: `sqlite3 data/metadata.db "SELECT * FROM functions;"`
- Check the function status is `deployed`
- Ensure the bundle file exists at the path in the database

### Worker Won't Start

If workers fail to spawn:
- Check Bun is installed: `bun --version`
- Verify the bundle path is correct and accessible
- Check server logs for detailed error messages
- Ensure the bundle exports a default handler function

### Timeout Errors

If invocations timeout:
- Check function execution time (default timeout: 30s)
- Increase timeout in gateway configuration if needed
- Review function code for blocking operations

See [Troubleshooting Guide](troubleshooting.md) for more solutions.
