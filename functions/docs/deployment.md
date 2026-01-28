# Function Deployment Guide

This document describes how functions are built, registered, deployed, and managed in BunBase Functions.

## Table of Contents

1. [Function Lifecycle](#function-lifecycle)
2. [Build Process](#build-process)
3. [Registration](#registration)
4. [Deployment](#deployment)
5. [Version Management](#version-management)
6. [Rollback](#rollback)
7. [Worker Warmup](#worker-warmup)

---

## Function Lifecycle

Functions progress through the following states:

```
REGISTERED → BUILT → DEPLOYED → WARM → BUSY → IDLE → TERMINATED
```

### State Descriptions

| State | Description | Transitions |
|-------|-------------|-------------|
| **REGISTERED** | Function metadata created, no code yet | → BUILT |
| **BUILT** | Code bundled and stored, not deployed | → DEPLOYED |
| **DEPLOYED** | Active version set, ready for invocations | → WARM (on first invocation) |
| **WARM** | Worker process ready and waiting | → BUSY (on invocation) |
| **BUSY** | Worker executing an invocation | → WARM (on completion) |
| **IDLE** | Worker idle, may be terminated after timeout | → WARM or TERMINATED |
| **TERMINATED** | Worker process killed | (final state) |

---

## Build Process

Build happens **outside** the functions service. The API server or CLI handles bundling.

### Build Steps

1. **Bundle Function Code**
   ```bash
   bun build ./function.ts --outdir ./dist --target bun
   # or
   esbuild ./function.ts --bundle --platform=node --outfile=./dist/bundle.js
   ```

2. **Store Bundle**
   ```
   data/bundles/{function_id}/{version}/bundle.js
   ```

3. **Register Version**
   - Create entry in `function_versions` table
   - Link to function ID
   - Store bundle path

### Build Requirements

- **Handler Export**: Function must export default handler
  ```typescript
  export default async function handler(req: Request): Promise<Response>
  ```

- **Valid JavaScript/TypeScript**: Must compile without errors

- **No External Dependencies**: All dependencies must be bundled (or use Bun's built-in modules)

---

## Registration

### Register Function

**IPC Command:** `REGISTER_FUNCTION`

**Payload:**
```json
{
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler"
}
```

**Response:**
```json
{
  "function_id": "func-123",
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler",
  "status": "registered"
}
```

**What Happens:**
1. Create entry in `functions` table
2. Set status to `registered`
3. Return function ID

---

## Deployment

### Deploy Function Version

**IPC Command:** `DEPLOY_FUNCTION`

**Payload:**
```json
{
  "function_id": "func-123",
  "version": "v1",
  "bundle_path": "/path/to/bundle.js"
}
```

**Response:**
```json
{
  "deployment_id": "deploy-789",
  "function_id": "func-123",
  "version": "v1",
  "status": "deployed"
}
```

**What Happens:**
1. Validate bundle exists at `bundle_path`
2. Create entry in `function_versions` table (if not exists)
3. Create entry in `function_deployments` table
4. Set function `active_version_id`
5. Set function status to `deployed`
6. Optionally warm up workers (see [Worker Warmup](#worker-warmup))

### Deployment Validation

Before deployment, the service validates:
- Bundle file exists
- Bundle is valid JavaScript
- Handler function exists (basic check)
- No syntax errors (if possible)

**Note:** Full validation happens on first invocation (when bundle is loaded).

---

## Version Management

### Versioning Strategy

Versions are **immutable** and **append-only**:
- Once created, a version never changes
- New versions are added, old versions kept
- Old versions can be deployed (rollback)

### Version Naming

Versions are strings (e.g., `"v1"`, `"v2"`, `"1.0.0"`, `"2026-01-27"`).

**Recommendation:** Use semantic versioning or timestamps.

### Version Storage

```
data/bundles/{function_id}/
├── v1/
│   └── bundle.js
├── v2/
│   └── bundle.js
└── v3/
    └── bundle.js
```

### Listing Versions

**IPC Command:** `GET_FUNCTION_VERSIONS` (future)

**Response:**
```json
{
  "versions": [
    {
      "id": "version-1",
      "version": "v1",
      "created_at": "2026-01-27T10:00:00Z",
      "bundle_path": "/path/to/v1/bundle.js"
    },
    {
      "id": "version-2",
      "version": "v2",
      "created_at": "2026-01-27T11:00:00Z",
      "bundle_path": "/path/to/v2/bundle.js"
    }
  ]
}
```

---

## Rollback

### Rollback to Previous Version

**Steps:**
1. List function versions (get previous version ID)
2. Deploy previous version
3. Workers for old version terminate
4. Workers for new version spawn (on next invocation)

**Example:**
```json
// Deploy v2 (current)
{
  "function_id": "func-123",
  "version": "v2",
  "bundle_path": "/path/to/v2/bundle.js"
}

// Rollback to v1
{
  "function_id": "func-123",
  "version": "v1",
  "bundle_path": "/path/to/v1/bundle.js"
}
```

**What Happens:**
- Old workers (v2) continue serving until idle timeout
- New invocations use v1 (new workers spawn)
- Old workers terminate after idle timeout

---

## Worker Warmup

### Automatic Warmup

Workers spawn **on-demand** (when first invocation arrives):
- No warm workers → spawn new worker
- Worker loads bundle → sends READY
- Worker ready → execute invocation

### Manual Warmup (Future)

**IPC Command:** `WARMUP_FUNCTION`

**Payload:**
```json
{
  "function_id": "func-123",
  "count": 2
}
```

**What Happens:**
- Spawn `count` workers
- Wait for READY messages
- Add to warm pool

**Use Cases:**
- Pre-warm before traffic spike
- Ensure fast first invocation
- Testing worker spawn

### Warmup Strategy

**Current (v1):**
- No automatic warmup
- Spawn on first invocation
- Maintain warm pool size (`warmWorkers` config)

**Future:**
- Scheduled warmup (before known traffic)
- Predictive warmup (based on patterns)
- Keep-alive for critical functions

---

## Deployment Best Practices

### 1. Version Before Deploy

Always create a new version before deploying:
- Keeps old version available for rollback
- Immutable versions prevent accidental changes
- Clear audit trail

### 2. Test Before Deploy

Validate bundle before deployment:
- Syntax check
- Handler existence check
- Basic smoke test (if possible)

### 3. Gradual Rollout (Future)

Deploy to subset of workers:
- Canary deployment
- A/B testing
- Traffic splitting

### 4. Monitor After Deploy

Watch metrics after deployment:
- Error rate
- Execution time
- Cold start rate

### 5. Keep Old Versions

Don't delete old versions immediately:
- Enables quick rollback
- Historical reference
- Debugging

---

## Deployment Flow Diagram

```
Build Function Code
    │
    ▼
Bundle (bun build / esbuild)
    │
    ▼
Store Bundle (filesystem)
    │
    ▼
Register Function (if new)
    │
    ▼
Create Version Entry
    │
    ▼
Deploy Version
    │
    ├─► Set active_version_id
    ├─► Set status = deployed
    └─► (Optional) Warmup workers
    │
    ▼
Ready for Invocations
    │
    ▼
First Invocation Arrives
    │
    ├─► No warm workers → Spawn worker
    ├─► Worker loads bundle
    ├─► Worker sends READY
    └─► Execute invocation
```

---

## Environment Variables

Functions can access environment variables set via API:

**Set Environment Variable:**
```json
POST /api/functions/{function_id}/env
{
  "key": "API_KEY",
  "value": "secret-value",
  "is_secret": true
}
```

**Access in Function:**
```typescript
export default async function handler(req: Request): Promise<Response> {
  const apiKey = process.env.API_KEY;
  // Use apiKey...
}
```

**Note:** Environment variables are injected when worker spawns (not on each invocation).

---

## Function Configuration

Functions can have configuration:

```json
{
  "function_id": "func-123",
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler",
  "memory_mb": 256,
  "timeout_seconds": 30,
  "max_concurrent_executions": 10,
  "warm_workers": 2
}
```

**Configuration Options:**
- `memory_mb`: Memory limit per worker
- `timeout_seconds`: Execution timeout
- `max_concurrent_executions`: Concurrency limit
- `warm_workers`: Number of warm workers to maintain

---

## Cleanup

### Delete Function

**IPC Command:** `DELETE_FUNCTION`

**Payload:**
```json
{
  "function_id": "func-123"
}
```

**What Happens:**
1. Terminate all workers
2. Mark function as deleted (soft delete)
3. Keep bundles (for audit)
4. Archive logs (optional)

### Delete Version

**IPC Command:** `DELETE_VERSION`

**Payload:**
```json
{
  "function_id": "func-123",
  "version": "v1"
}
```

**What Happens:**
1. Check if version is active (reject if active)
2. Delete bundle file
3. Delete version entry
4. Delete deployment entries

**Note:** Cannot delete active version (must deploy different version first).
