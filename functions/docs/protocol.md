# BunBase Functions IPC Protocol

This document specifies the IPC protocols used by BunBase Functions:

1. **Go ↔ Bun IPC** (stdin/stdout JSON)
2. **Unix Socket IPC** (API Server ↔ Functions Service)

---

## 1. Go ↔ Bun IPC Protocol

### Transport

JSON messages over stdin/stdout with newline-delimited framing (NDJSON).

### Message Format

All messages are JSON objects with the following structure:

```json
{
  "id": "string",
  "type": "string",
  "payload": {}
}
```

**Fields:**
- `id`: Unique message identifier (UUID or worker ID)
- `type`: Message type (see below)
- `payload`: Message-specific payload

### Message Types

#### READY (Bun → Go)

Sent by Bun worker when it's ready to accept invocations.

```json
{
  "id": "worker-123",
  "type": "ready",
  "payload": {}
}
```

**When:** After worker loads bundle and initializes runtime.

**Response:** None (Go marks worker as READY).

---

#### INVOKE (Go → Bun)

Sent by Go to request function execution.

```json
{
  "id": "invoke-456",
  "type": "invoke",
  "payload": {
    "method": "POST",
    "path": "/users",
    "headers": {
      "Content-Type": "application/json",
      "X-Invocation-Id": "invoke-456"
    },
    "query": {
      "page": "1"
    },
    "body": "base64-encoded-body",
    "deadline_ms": 5000
  }
}
```

**Payload Fields:**
- `method`: HTTP method (GET, POST, PUT, DELETE, etc.)
- `path`: Request path
- `headers`: HTTP headers (object)
- `query`: Query parameters (object)
- `body`: Request body (base64-encoded string, optional)
- `deadline_ms`: Execution deadline in milliseconds

**Response:** Bun must send RESPONSE or ERROR message with matching `id`.

---

#### RESPONSE (Bun → Go)

Sent by Bun worker after successful handler execution.

```json
{
  "id": "invoke-456",
  "type": "response",
  "payload": {
    "status": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "body": "base64-encoded-body"
  }
}
```

**Payload Fields:**
- `status`: HTTP status code (number)
- `headers`: Response headers (object)
- `body`: Response body (base64-encoded string)

**When:** Handler returns `Response` object successfully.

---

#### LOG (Bun → Go)

Sent by Bun worker for log messages (console.log, console.error, etc.).

```json
{
  "id": "invoke-456",
  "type": "log",
  "payload": {
    "level": "info",
    "message": "Processing request",
    "metadata": {
      "timestamp": "2026-01-27T10:00:00Z"
    }
  }
}
```

**Payload Fields:**
- `level`: Log level (info, warn, error, debug)
- `message`: Log message (string)
- `metadata`: Additional metadata (object, optional)

**When:** Any console.* call in user code.

**Note:** Multiple LOG messages can be sent for a single invocation.

---

#### ERROR (Bun → Go)

Sent by Bun worker when handler execution fails.

```json
{
  "id": "invoke-456",
  "type": "error",
  "payload": {
    "message": "Handler threw error",
    "stack": "Error: ...\n    at handler (bundle.js:10:5)\n    ...",
    "code": "HANDLER_ERROR"
  }
}
```

**Payload Fields:**
- `message`: Error message (string)
- `stack`: Stack trace (string, optional)
- `code`: Error code (string, optional)

**When:** Handler throws exception or returns invalid response.

---

### Framing

Messages are newline-delimited JSON (NDJSON):

```
{"id":"worker-123","type":"ready","payload":{}}\n
{"id":"invoke-456","type":"invoke","payload":{...}}\n
{"id":"invoke-456","type":"response","payload":{...}}\n
```

**Why NDJSON:**
- Simple parsing (one JSON object per line)
- Streaming support (read line-by-line)
- Easy to debug (human-readable)

### Error Handling

**Invalid Message:**
- Bun: Ignore malformed JSON, continue
- Go: Log error, mark worker unhealthy

**Missing Response:**
- Go: Timeout after `deadline_ms`, kill worker

**Process Crash:**
- Go: Detect exit, remove from pool, spawn replacement

---

## 2. Unix Socket IPC Protocol

### Transport

Binary frames over Unix domain socket (similar to DocDB IPC).

### Frame Format

```
┌─────────────────────────────────────────┐
│  Length (4 bytes, little-endian)      │
├─────────────────────────────────────────┤
│  RequestID (8 bytes, little-endian)   │
├─────────────────────────────────────────┤
│  Command (1 byte)                      │
├─────────────────────────────────────────┤
│  Payload Length (4 bytes, little-endian)│
├─────────────────────────────────────────┤
│  Payload (variable length)            │
└─────────────────────────────────────────┘
```

**Fields:**
- `Length`: Total frame length (including header)
- `RequestID`: Unique request identifier
- `Command`: Command type (see below)
- `Payload Length`: Length of payload data
- `Payload`: Command-specific payload (JSON)

### Commands

#### INVOKE (0x01)

Invoke a function.

**Request Payload:**
```json
{
  "function_id": "func-123",
  "method": "POST",
  "path": "/users",
  "headers": {},
  "query": {},
  "body": "base64-encoded-body"
}
```

**Response Payload:**
```json
{
  "success": true,
  "status": 200,
  "headers": {},
  "body": "base64-encoded-body",
  "execution_time_ms": 45,
  "execution_id": "exec-456"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Function not found",
  "code": "FUNCTION_NOT_FOUND"
}
```

---

#### GET_LOGS (0x02)

Get function execution logs.

**Request Payload:**
```json
{
  "function_id": "func-123",
  "limit": 100,
  "offset": 0,
  "level": "error",
  "execution_id": "exec-456",
  "start_date": "2026-01-27T00:00:00Z",
  "end_date": "2026-01-27T23:59:59Z"
}
```

**Response Payload:**
```json
{
  "logs": [
    {
      "id": "log-789",
      "execution_id": "exec-456",
      "level": "info",
      "message": "Processing request",
      "metadata": {},
      "timestamp": "2026-01-27T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

#### GET_METRICS (0x03)

Get function metrics.

**Request Payload:**
```json
{
  "function_id": "func-123",
  "period": "day"
}
```

**Response Payload:**
```json
{
  "invocations": 1000,
  "errors": 5,
  "average_duration_ms": 45,
  "cold_starts": 10,
  "last_invoked": "2026-01-27T10:00:00Z",
  "period": "day"
}
```

---

#### REGISTER_FUNCTION (0x04)

Register a new function.

**Request Payload:**
```json
{
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler"
}
```

**Response Payload:**
```json
{
  "function_id": "func-123",
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler",
  "status": "registered"
}
```

---

#### DEPLOY_FUNCTION (0x05)

Deploy a function version.

**Request Payload:**
```json
{
  "function_id": "func-123",
  "version": "v1",
  "bundle_path": "/path/to/bundle.js"
}
```

**Response Payload:**
```json
{
  "deployment_id": "deploy-789",
  "function_id": "func-123",
  "version": "v1",
  "status": "deployed"
}
```

---

### Response Format

All responses follow this structure:

```
┌─────────────────────────────────────────┐
│  Length (4 bytes, little-endian)      │
├─────────────────────────────────────────┤
│  RequestID (8 bytes, little-endian)   │
├─────────────────────────────────────────┤
│  Status (1 byte)                       │
├─────────────────────────────────────────┤
│  Payload Length (4 bytes, little-endian)│
├─────────────────────────────────────────┤
│  Payload (variable length, JSON)      │
└─────────────────────────────────────────┘
```

**Status Values:**
- `0x00`: Success
- `0x01`: Error

**Error Payload:**
```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

---

## 3. Message Flow Examples

### Example 1: Successful Invocation

```
Go → Bun: {"id":"invoke-1","type":"invoke","payload":{...}}
Bun → Go: {"id":"invoke-1","type":"log","payload":{"level":"info","message":"Starting"}}
Bun → Go: {"id":"invoke-1","type":"response","payload":{"status":200,"headers":{},"body":"..."}}
```

### Example 2: Handler Error

```
Go → Bun: {"id":"invoke-2","type":"invoke","payload":{...}}
Bun → Go: {"id":"invoke-2","type":"error","payload":{"message":"Handler error","stack":"..."}}
```

### Example 3: Worker Startup

```
Bun → Go: {"id":"worker-1","type":"ready","payload":{}}
```

---

## 4. Implementation Notes

### Go Side

- Use `encoding/json` for JSON encoding/decoding
- Use `bufio.Scanner` for line-by-line reading (NDJSON)
- Use `exec.Cmd` with `StdinPipe()` and `StdoutPipe()`
- Handle process crashes (check `cmd.ProcessState`)

### Bun Side

- Use `Bun.stdin` for reading (if available) or `process.stdin`
- Use `console.log` / `console.error` → send LOG messages
- Wrap handler in try/catch → send ERROR on exception
- Send READY after bundle load

### Error Recovery

- **Invalid JSON**: Log and continue (don't crash)
- **Missing Response**: Go times out, kills worker
- **Process Crash**: Go detects, removes from pool, spawns replacement

---

## 5. Protocol Versioning

**Current Version:** 1.0

**Versioning Strategy:**
- Add new message types (backward compatible)
- Extend payloads (backward compatible)
- Breaking changes require version negotiation (future)

**Version Header (Future):**
```json
{
  "version": "1.0",
  "id": "...",
  "type": "...",
  "payload": {}
}
```
