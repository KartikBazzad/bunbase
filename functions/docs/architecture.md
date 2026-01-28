# BunBase Functions Architecture

This document describes the BunBase Functions system architecture, component interactions, and design decisions.

## Table of Contents

1. [System Overview](#system-overview)
2. [Component Architecture](#component-architecture)
3. [Data Flow](#data-flow)
4. [Worker Model](#worker-model)
5. [IPC Protocol](#ipc-protocol)
6. [Concurrency Model](#concurrency-model)
7. [Storage Architecture](#storage-architecture)
8. [Failure & Recovery](#failure--recovery)
9. [Design Decisions](#design-decisions)
10. [Alternatives Considered](#alternatives-considered)

---

## System Overview

BunBase Functions is a serverless execution subsystem that manages long-lived Bun workers like a database connection pool. The architecture uses **Go as the control plane** and **Bun as the execution plane**, optimized for warm execution and local-first developer experience.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Invocation Sources                        │
│  HTTP │ Scheduled │ Events │ CLI                      │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Functions Gateway (Go)                    │
│  ┌──────────────────────────────────────────────┐      │
│  │  HTTP Server                                │      │
│  │  POST /functions/:name                     │      │
│  │  GET  /functions/:name/logs                │      │
│  │  GET  /functions/:name/metrics             │      │
│  └──────────────────────────────────────────────┘      │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Function Router (Go)                      │
│  ┌──────────────────────────────────────────────┐      │
│  │  Name → Function ID Resolution              │      │
│  │  Active Version Lookup                      │      │
│  │  Deployment Status Check                    │      │
│  └──────────────────────────────────────────────┘      │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Scheduler (Go)                             │
│  ┌──────────────────────────────────────────────┐      │
│  │  Invocation Queue                            │      │
│  │  Worker Assignment                           │      │
│  │  Timeout Management                          │      │
│  │  Concurrency Limits                          │      │
│  └──────────────────────────────────────────────┘      │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Worker Pool (Go)                           │
│  ┌──────────────────────────────────────────────┐      │
│  │  Function Version → Pool Mapping            │      │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐   │      │
│  │  │ Warm []  │  │ Busy [] │  │ Max      │   │      │
│  │  └──────────┘  └──────────┘  └──────────┘   │      │
│  └──────────────────────────────────────────────┘      │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Bun Worker Process                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  Bun Runtime                                 │      │
│  │  ┌────────────────────────────────────┐      │      │
│  │  │  Function Bundle Loader           │      │      │
│  │  │  Handler Execution                │      │      │
│  │  │  IPC Message Handler              │      │      │
│  │  └────────────────────────────────────┘      │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
```

### Key Characteristics

- **Go Control Plane**: Manages lifecycle, routing, scheduling, observability
- **Bun Execution Plane**: Executes JavaScript/TypeScript user code
- **Long-Lived Workers**: One Bun process per function version, handles multiple invocations
- **Worker Pool Pattern**: Similar to database connection pool - warm workers ready, spawn on demand
- **IPC Communication**: JSON over stdin/stdout between Go and Bun
- **Single Process**: All worker pools managed in one Go process

---

## Component Architecture

### 1. Functions Gateway

**Purpose:** HTTP entry point for function invocations.

**Responsibilities:**
- Accept HTTP requests to `/functions/:name`
- Parse request (method, path, headers, query, body)
- Normalize to internal `InvokeRequest`
- Return HTTP response from function result

**Key Components:**
- HTTP server (Go `net/http`)
- Request parsing
- Response formatting
- Error handling

**Thread Safety:** HTTP handlers are thread-safe (Go's HTTP server).

---

### 2. Function Router

**Purpose:** Resolve function names/IDs and route to appropriate pools.

**Responsibilities:**
- Name → Function ID resolution
- Active version lookup
- Deployment status validation
- Function metadata retrieval

**Key Components:**
- Metadata store interface
- Name index
- Version tracking

**Thread Safety:** Read operations are thread-safe (RWMutex).

---

### 3. Scheduler

**Purpose:** Manage invocation queue and worker assignment.

**Responsibilities:**
- Queue invocations per function
- Assign workers to invocations
- Handle timeouts
- Enforce concurrency limits
- Track cold starts

**Key Components:**
- Invocation queue (per function)
- Worker assignment logic
- Timeout timers
- Concurrency controller

**Thread Safety:** Queue operations are thread-safe (mutex-protected).

---

### 4. Worker Pool

**Purpose:** Manage Bun workers for a function version.

**Responsibilities:**
- Maintain warm/busy worker lists
- Spawn new workers on demand
- Terminate idle workers
- Health check workers
- Handle worker crashes

**Key Components:**
- `WorkerPool` struct (per function version)
- `Worker` struct (per Bun process)
- Worker lifecycle management
- IPC communication

**Thread Safety:** Pool operations are thread-safe (RWMutex).

**Structure:**

```go
type WorkerPool struct {
    functionID   string
    version      string
    warm         []*Worker
    busy         []*Worker
    maxWorkers   int
    warmWorkers  int
    mu           sync.RWMutex
    logger       *logger.Logger
}

type Worker struct {
    id           string
    process      *exec.Cmd
    stdin        io.WriteCloser
    stdout       io.ReadCloser
    state        WorkerState
    lastUsed     time.Time
    invocations  int64
    mu           sync.Mutex
}
```

---

### 5. Bun Worker Process

**Purpose:** Execute user function code.

**Responsibilities:**
- Load function bundle
- Initialize Bun runtime
- Handle IPC messages (invoke, ready, error)
- Execute handler function
- Stream logs
- Return responses

**Key Components:**
- `worker.ts` script
- Bundle loader
- Handler executor
- IPC message loop

**Process Lifecycle:**

```
Go spawns → bun worker.ts
        → Load bundle
        → Initialize runtime
        → Open IPC (stdin/stdout)
        → Send READY message
        → Wait for INVOKE messages
        → Execute handler
        → Send RESPONSE
        → (Repeat or terminate)
```

---

### 6. Metadata Store

**Purpose:** Persist function metadata in SQLite.

**Responsibilities:**
- Function definitions
- Version tracking
- Deployment management
- Trigger configuration

**Schema:**

```sql
-- Functions table
CREATE TABLE functions (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    runtime TEXT NOT NULL,
    handler TEXT NOT NULL,
    status TEXT NOT NULL,
    active_version_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Function versions table
CREATE TABLE function_versions (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    version TEXT NOT NULL,
    bundle_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (function_id) REFERENCES functions(id)
);

-- Function deployments table
CREATE TABLE function_deployments (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (function_id) REFERENCES functions(id),
    FOREIGN KEY (version_id) REFERENCES function_versions(id)
);
```

---

### 7. Bundle Storage

**Purpose:** Store function bundles on filesystem.

**Structure:**

```
data/
└── bundles/
    └── {function_id}/
        └── {version}/
            └── bundle.js
```

**Characteristics:**
- Immutable: Versions never change
- Append-only: New versions added, old versions kept
- Fast access: Direct file reads

---

### 8. Log Storage

**Purpose:** Store function execution logs.

**Formats:**
- SQLite: Structured queries, metadata
- JSONL: Streaming, large volumes

**Schema:**

```sql
CREATE TABLE function_logs (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    level TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata TEXT,
    timestamp INTEGER NOT NULL
);
```

---

### 9. IPC Server

**Purpose:** Unix socket server for API server integration.

**Responsibilities:**
- Accept IPC connections
- Handle commands (INVOKE, GET_LOGS, GET_METRICS, REGISTER_FUNCTION, DEPLOY_FUNCTION)
- Route to appropriate handlers
- Return responses

**Protocol:** Binary frames over Unix socket (similar to DocDB).

---

## Data Flow

### 1. Function Invocation Flow

```
HTTP Request
    │
    ▼
Gateway (parse request)
    │
    ▼
Router (resolve function name → ID)
    │
    ▼
Scheduler (enqueue invocation)
    │
    ▼
Worker Pool (acquire worker)
    ├─ Warm worker available → Use it
    ├─ No warm worker → Spawn new Bun process
    └─ Max workers reached → Queue or reject
    │
    ▼
Bun Worker Process
    ├─ Load function bundle (if not loaded)
    ├─ Execute handler(req)
    ├─ Stream logs (if any)
    └─ Return Response
    │
    ▼
Worker Pool (release worker to warm)
    │
    ▼
Gateway (format HTTP response)
    │
    ▼
Client
```

### 2. Worker Spawn Flow

```
Scheduler requests worker
    │
    ▼
Worker Pool checks warm list
    │
    ├─ Warm worker available → Return it
    │
    └─ No warm worker:
        │
        ▼
    Check max workers limit
        │
        ├─ Under limit → Spawn new worker
        │
        └─ At limit → Queue or reject
        │
        ▼
    Spawn Bun process
        │
        ├─ exec.Command("bun", "worker.ts")
        ├─ Set stdin/stdout pipes
        ├─ Set environment variables
        ├─ Set working directory
        └─ Start process
        │
        ▼
    Wait for READY message
        │
        ├─ Timeout → Kill process, error
        │
        └─ READY received → Add to warm list
        │
        ▼
    Return worker
```

### 3. Worker Lifecycle Flow

```
STARTING
    │
    ▼ (READY message received)
READY (warm)
    │
    ▼ (invocation assigned)
BUSY
    │
    ├─► Success → IDLE
    └─► Error → IDLE
    │
    ▼ (idle timeout)
TERMINATED
```

### 4. Cold Start Detection

```
Invocation arrives
    │
    ▼
Check warm workers
    │
    ├─ Warm worker available → Warm execution
    │
    └─ No warm worker:
        │
        ▼
    Spawn new worker
        │
        ▼
    Record spawn time
        │
        ▼
    Wait for READY
        │
        ▼
    Record ready time
        │
        ▼
    Calculate cold start duration
        │
        ▼
    Mark invocation as cold start
```

---

## Worker Model

### Long-Lived Workers

**Key Insight:** One Bun process per function version, handles multiple invocations.

**Why?**
- Bun startup cost amortized across invocations
- Module cache reuse
- Predictable memory usage
- Fast warm execution

**Trade-offs:**
- Memory overhead (idle workers consume memory)
- Stale code (must restart for new version)
- Single-threaded execution (v1)

### Worker States

| State | Meaning | Transitions |
|-------|---------|-------------|
| STARTING | Process spawned, waiting for READY | → READY, → TERMINATED |
| READY | Warm, waiting for invocation | → BUSY |
| BUSY | Executing invocation | → READY, → TERMINATED |
| IDLE | Idle, may be terminated | → READY, → TERMINATED |
| TERMINATED | Process killed | (final state) |

### Worker Pool Strategy

1. **Warm Pool**: Maintain `warmWorkers` count of ready workers
2. **Spawn on Demand**: If warm pool empty and under max limit, spawn new
3. **Idle Timeout**: Terminate workers idle for `idleTimeout`
4. **Crash Recovery**: Detect crashes, remove from pool, spawn replacement

---

## IPC Protocol

### Go ↔ Bun IPC (stdin/stdout)

**Transport:** JSON messages over stdin/stdout with newline framing.

**Message Format:**

```json
{
  "id": "uuid",
  "type": "invoke | response | log | ready | error",
  "payload": {}
}
```

**Message Types:**

1. **READY** (Bun → Go)
   ```json
   {
     "id": "worker-123",
     "type": "ready",
     "payload": {}
   }
   ```

2. **INVOKE** (Go → Bun)
   ```json
   {
     "id": "invoke-456",
     "type": "invoke",
     "payload": {
       "method": "POST",
       "path": "/users",
       "headers": {"Content-Type": "application/json"},
       "query": {},
       "body": "base64-encoded-body",
       "deadline_ms": 5000
     }
   }
   ```

3. **RESPONSE** (Bun → Go)
   ```json
   {
     "id": "invoke-456",
     "type": "response",
     "payload": {
       "status": 200,
       "headers": {"Content-Type": "application/json"},
       "body": "base64-encoded-body"
     }
   }
   ```

4. **LOG** (Bun → Go)
   ```json
   {
     "id": "invoke-456",
     "type": "log",
     "payload": {
       "level": "info",
       "message": "Processing request",
       "metadata": {}
     }
   }
   ```

5. **ERROR** (Bun → Go)
   ```json
   {
     "id": "invoke-456",
     "type": "error",
     "payload": {
       "message": "Handler threw error",
       "stack": "..."
     }
   }
   ```

**Framing:** Newline-delimited JSON (NDJSON) for streaming.

---

### Unix Socket IPC (API Server ↔ Functions Service)

**Transport:** Binary frames over Unix socket (similar to DocDB).

**Commands:**

- `INVOKE` - Invoke function
- `GET_LOGS` - Get function logs
- `GET_METRICS` - Get function metrics
- `REGISTER_FUNCTION` - Register new function
- `DEPLOY_FUNCTION` - Deploy function version

**Protocol:** Length-prefixed binary frames (see `docs/protocol.md`).

---

## Concurrency Model

### Worker Pool Concurrency

- **Per-Pool Lock**: RWMutex protects warm/busy lists
- **Worker Lock**: Mutex protects individual worker state
- **Scheduler Lock**: Mutex protects invocation queue

### Invocation Concurrency

- **Per-Function Limit**: Configurable max concurrent invocations
- **Queue When Full**: Invocations queued if limit reached
- **Timeout**: Queued invocations timeout after deadline

### Worker Spawn Concurrency

- **Serial Spawn**: One worker spawn at a time per pool
- **Max Workers**: Hard limit per function version
- **Backpressure**: Reject if max reached

---

## Storage Architecture

### Bundle Storage

**Location:** `data/bundles/{function_id}/{version}/bundle.js`

**Characteristics:**
- Immutable: Versions never change
- Append-only: New versions added
- Fast access: Direct file reads

### Metadata Storage

**Location:** `data/metadata.db` (SQLite)

**Tables:**
- `functions` - Function definitions
- `function_versions` - Code versions
- `function_deployments` - Active deployments
- `function_triggers` - Invocation triggers

### Log Storage

**Locations:**
- `data/logs.db` (SQLite) - Structured queries
- `data/logs/*.jsonl` (JSONL) - Streaming

**Retention:** Configurable (default: 30 days)

---

## Failure & Recovery

### Worker Crash

**Detection:**
- Process exit detected by Go
- IPC read error
- Health check timeout

**Recovery:**
- Remove from pool
- Spawn replacement (if needed)
- Log error

### Timeout

**Detection:**
- Invocation exceeds `deadline_ms`
- Worker unresponsive

**Recovery:**
- Kill worker process
- Return timeout error
- Spawn replacement (if needed)

### Memory Limit

**Detection:**
- Process memory exceeds limit
- OS OOM killer

**Recovery:**
- Kill worker process
- Log error
- Spawn replacement (if needed)

### Bad Code

**Detection:**
- Bundle load failure
- Handler execution error
- Syntax errors

**Recovery:**
- Mark function unhealthy
- Return error to caller
- Do not spawn replacement

---

## Design Decisions

### 1. Long-Lived Workers

**Decision:** One Bun process per function version, handles multiple invocations.

**Rationale:**
- Amortizes Bun startup cost
- Module cache reuse
- Predictable memory

**Trade-offs:**
- Memory overhead (idle workers)
- Stale code (must restart for new version)

### 2. IPC Transport: stdin/stdout

**Decision:** JSON over stdin/stdout for Go ↔ Bun communication.

**Rationale:**
- Simple, no network overhead
- Works across platforms
- Easy to debug

**Trade-offs:**
- No multiplexing (one worker per process)
- Process-bound communication

### 3. Worker Pool Pattern

**Decision:** Similar to database connection pool.

**Rationale:**
- Proven pattern
- Predictable behavior
- Easy to reason about

**Trade-offs:**
- Fixed limits (no auto-scaling)
- Manual tuning required

### 4. Single-Tenant v1

**Decision:** No strong sandboxing, trust local user.

**Rationale:**
- Simplicity
- Performance
- Local-first focus

**Trade-offs:**
- Security limitations
- Not suitable for untrusted code

---

## Alternatives Considered

### 1. Process-Per-Request (Lambda-style)

**Rejected:** Too slow (cold start on every request).

### 2. WebSocket IPC

**Rejected:** More complex, network overhead, not needed for local execution.

### 3. Shared Memory IPC

**Rejected:** Platform-specific, complex, JSON over pipes is sufficient.

### 4. WASM Runtime

**Rejected:** Not in scope for v1, Bun provides better performance for JS/TS.

---

## Future Considerations

- **Isolates**: V8 isolates for better isolation
- **WASM**: WebAssembly runtime support
- **Edge**: Distributed execution
- **Multi-Tenant**: Strong sandboxing
- **Auto-Scaling**: Dynamic worker limits
