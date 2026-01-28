# QuickJS-NG Function Deployment Guide

This guide shows how to deploy functions using the QuickJS-NG runtime.

## Quick Start

### 1. Deploy a Function

```bash
cd functions
./scripts/deploy-quickjs-function.sh <function-name> <function-file> [version] [profile]
```

**Example:**
```bash
./scripts/deploy-quickjs-function.sh hello-world examples/hello-world.ts v1 strict
```

### 2. Start Functions Service

```bash
./functions --data-dir ./data --socket /tmp/functions.sock
```

The service will automatically load deployed functions on startup.

### 3. Test the Function

```bash
# Via HTTP gateway
curl 'http://localhost:8080/functions/hello-world?name=Alice'

# Expected response:
# {"message":"Hello, Alice!","timestamp":"2026-01-28T...","method":"GET","path":"/"}
```

## Deployment Script Options

### Basic Usage

```bash
./scripts/deploy-quickjs-function.sh hello-world examples/hello-world.ts
```

### With Custom Version

```bash
./scripts/deploy-quickjs-function.sh my-function ./my-function.ts v2
```

### With Security Profile

```bash
# Strict (default) - no filesystem, no network, no eval
./scripts/deploy-quickjs-function.sh api-handler ./api.ts v1 strict

# Permissive - all capabilities enabled
./scripts/deploy-quickjs-function.sh trusted-function ./trusted.ts v1 permissive
```

### With Custom Data Directory

```bash
DATA_DIR=/custom/path ./scripts/deploy-quickjs-function.sh hello examples/hello-world.ts
```

## Function Requirements

Functions must export a default handler that accepts `Request` and returns `Promise<Response>`:

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

## Security Profiles

### Strict Profile (Default)

- ✅ No filesystem access
- ✅ No network access
- ✅ No child process spawning
- ✅ No eval/Function constructor
- ✅ Memory limit: 100MB
- ✅ CPU limit: 30 seconds
- ✅ File descriptors: 10

**Use for:** Untrusted user code, multi-tenant environments

### Permissive Profile

- ✅ Full filesystem access
- ✅ Full network access
- ✅ Child process spawning allowed
- ✅ Eval/Function constructor allowed
- ✅ Memory limit: 512MB
- ✅ CPU limit: 5 minutes
- ✅ File descriptors: 100

**Use for:** Trusted code, development/testing

## What the Script Does

1. **Creates Bundle Directory**: `data/bundles/{function-id}/{version}/`
2. **Builds Bundle**: Uses `bun` or `esbuild` to bundle your function
3. **Registers Function**: Creates entry in metadata database with QuickJS-NG runtime
4. **Creates Version**: Links bundle to function version
5. **Deploys Function**: Sets function status to "deployed"

## Bundle Structure

```
data/bundles/
└── func-hello-world/
    └── v1/
        └── bundle.js
```

## Verification

After deployment, verify it worked:

```bash
# Check database
sqlite3 data/metadata.db "SELECT name, runtime, status FROM functions WHERE name = 'hello-world';"

# Check bundle exists
ls -lh data/bundles/func-hello-world/v1/bundle.js

# Check function is deployed
sqlite3 data/metadata.db "SELECT status FROM functions WHERE name = 'hello-world';"
```

## Testing Deployment

Use the test script:

```bash
./scripts/test-quickjs-deployment.sh
```

This will:
1. Deploy the example QuickJS function
2. Verify it's in the database
3. Check the bundle exists
4. Show next steps

## Troubleshooting

### Bundle Build Fails

**Problem:** "No bundler available"

**Solution:** Install `bun` or `esbuild`:
```bash
# Install bun
curl -fsSL https://bun.sh/install | bash

# Or install esbuild
npm install -g esbuild
```

### Function Not Loading

**Problem:** Function deployed but not executing

**Solution:**
1. Check QuickJS worker binary exists: `ls cmd/quickjs-worker/quickjs-worker`
2. Check function runtime in DB: `sqlite3 data/metadata.db "SELECT runtime FROM functions WHERE name = 'your-function';"`
3. Restart functions service to reload pools

### QuickJS Worker Not Found

**Problem:** "QuickJS worker binary not found"

**Solution:** Build the QuickJS worker:
```bash
cd cmd/quickjs-worker
make
cd ../..
```

## Examples

### Example 1: Simple Hello World

```bash
./scripts/deploy-quickjs-function.sh hello examples/hello-world.ts
```

### Example 2: API Handler with Permissive Profile

```bash
./scripts/deploy-quickjs-function.sh api-handler ./api.ts v1 permissive
```

### Example 3: Multiple Versions

```bash
# Deploy v1
./scripts/deploy-quickjs-function.sh my-function ./function.ts v1

# Deploy v2 (keeps v1 available for rollback)
./scripts/deploy-quickjs-function.sh my-function ./function.ts v2
```

## Next Steps

After deployment:

1. **Start Functions Service**: `./functions --data-dir ./data`
2. **Test Function**: `curl 'http://localhost:8080/functions/your-function'`
3. **Monitor Logs**: Check service logs for execution details
4. **View Metrics**: Check `data/metrics.db` for execution statistics

## Differences from Bun Runtime

- **Smaller Memory Footprint**: 5-10MB vs 50-200MB
- **Faster Cold Starts**: 50-100ms vs 200-500ms
- **Better Security**: Capability-based sandboxing
- **No Node.js APIs**: Uses Web APIs only (Request/Response)
- **Limited npm Packages**: Only bundled dependencies work
