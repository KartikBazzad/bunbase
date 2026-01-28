# Testing BunBase Functions

## Quick Test

### 1. Check if server is running

```bash
# Check HTTP health endpoint
curl http://localhost:8080/health

# Should return: OK
```

### 2. Check Unix socket

```bash
# Check if socket exists
ls -la /tmp/functions.sock

# Should show the socket file
```

## Manual Testing (Current State)

Since the IPC handlers for `REGISTER_FUNCTION` and `DEPLOY_FUNCTION` are not fully implemented yet, you can test by manually inserting data into the database:

### Step 1: Create a test function bundle

Create a simple test function:

```typescript
// test-function.ts
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
  });
}
```

Build it:
```bash
bun build test-function.ts --outdir ./test-bundle --target bun
# This creates ./test-bundle/test-function.js
```

### Step 2: Store the bundle

```bash
mkdir -p data/bundles/test-func/v1
cp test-bundle/test-function.js data/bundles/test-func/v1/bundle.js
```

### Step 3: Manually insert into database

Use SQLite CLI to insert function metadata:

```bash
sqlite3 data/metadata.db <<EOF
-- Insert function
INSERT INTO functions (id, name, runtime, handler, status, created_at, updated_at)
VALUES ('test-func', 'test-func', 'bun', 'handler', 'registered', strftime('%s', 'now'), strftime('%s', 'now'));

-- Insert version
INSERT INTO function_versions (id, function_id, version, bundle_path, created_at)
VALUES ('version-1', 'test-func', 'v1', '$(pwd)/data/bundles/test-func/v1/bundle.js', strftime('%s', 'now'));

-- Deploy function
INSERT INTO function_deployments (id, function_id, version_id, status, created_at)
VALUES ('deploy-1', 'test-func', 'version-1', 'active', strftime('%s', 'now'));

-- Update function to deployed
UPDATE functions 
SET status = 'deployed', active_version_id = 'version-1', updated_at = strftime('%s', 'now')
WHERE id = 'test-func';
EOF
```

### Step 4: Create a pool and register it

You'll need to create a pool manually. For now, you can use the Go client or modify main.go to auto-create pools for deployed functions.

### Step 5: Invoke via HTTP

```bash
curl -X POST http://localhost:8080/functions/test-func?name=Alice

# Should return JSON response from the function
```

## Using the Go Client

Create a test program:

```go
// test-client.go
package main

import (
	"fmt"
	"github.com/kartikbazzad/bunbase/functions/pkg/client"
)

func main() {
	cli := client.New("/tmp/functions.sock")
	if err := cli.Connect(); err != nil {
		panic(err)
	}
	defer cli.Close()

	result, err := cli.Invoke(&client.InvokeRequest{
		FunctionID: "test-func",
		Method:     "GET",
		Path:       "/",
		Headers:    map[string]string{},
		Query:      map[string]string{"name": "Alice"},
		Body:       nil,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Status: %d\n", result.Status)
	fmt.Printf("Body: %s\n", string(result.Body))
}
```

Run it:
```bash
go run test-client.go
```

## Expected Behavior

1. **Health check** should return `OK`
2. **Function invocation** should:
   - Spawn a Bun worker (cold start)
   - Load the function bundle
   - Execute the handler
   - Return the response
   - Keep worker warm for next invocation

## Debugging

Check logs:
```bash
# Server logs will show:
# - Worker spawns
# - Invocations
# - Errors
```

Check worker process:
```bash
ps aux | grep bun
# Should see Bun processes running worker.ts
```

Check database:
```bash
sqlite3 data/metadata.db "SELECT * FROM functions;"
sqlite3 data/metadata.db "SELECT * FROM function_versions;"
sqlite3 data/metadata.db "SELECT * FROM function_deployments;"
```
