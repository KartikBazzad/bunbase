# Bundoc Server

Multi-tenant database server with Firebase-compatible REST API.

## Quick Start

```bash
# Build
go build -o bundoc-server

# Run
./bundoc-server

# Server starts on http://localhost:8080
```

## API Endpoints

### Health Check

```bash
GET /health
```

### Create Document

```bash
curl -X POST http://localhost:8080/v1/projects/my-project/databases/(default)/documents/users \
  -H "Content-Type: application/json" \
  -d '{"_id": "user1", "name": "Alice", "email": "alice@example.com"}'
```

### Get Document

```bash
curl http://localhost:8080/v1/projects/my-project/databases/(default)/documents/users/user1
```

### Update Document

```bash
curl -X PATCH http://localhost:8080/v1/projects/my-project/databases/(default)/documents/users/user1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Updated", "age": 31}'
```

### Delete Document

```bash
curl -X DELETE http://localhost:8080/v1/projects/my-project/databases/(default)/documents/users/user1
```

## Features

✅ Multi-tenant (one bundoc instance per project)  
✅ Lock-free hot/cold database management  
✅ Firebase-compatible REST API  
✅ CORS enabled  
✅ Graceful shutdown  
✅ Health monitoring  
✅ ACID transactions per request

## Architecture

```
Client Request
    ↓
HTTP Server (:8080)
    ↓
Document Handlers
    ↓
Instance Manager (sync.Map)
    ↓
Hot DB Cache ←→ Bundoc Instances
```

## Configuration

Data stored in: `./data/projects/{projectId}/`

Instance Manager:

- Max hot instances: 100
- Idle TTL: 10 minutes
- Eviction interval: 1 minute

### Resource limits (DoS protection)

In cloud mode you can cap per-project usage via environment variables. Set on the bundoc-server process; `0` or unset = unlimited.

| Variable | Description |
| -------- | ----------- |
| `BUNDOC_LIMITS_MAX_CONNECTIONS_PER_PROJECT` | Max concurrent requests per project (returns 429 when exceeded) |
| `BUNDOC_LIMITS_MAX_EXECUTION_MS` | Max request duration in ms (server write timeout) |
| `BUNDOC_LIMITS_MAX_SCAN_DOCS` | Cap on list/query result limit per request (e.g. 1000) |
| `BUNDOC_LIMITS_MAX_DATABASE_SIZE_BYTES` | Max on-disk size per project in bytes (returns 507 when exceeded on create/update) |

## Development

```bash
# Run tests
go test -v -race ./...

# Build
go build

# Run
./bundoc-server
```

## Project Isolation

Each project ID gets its own isolated bundoc database instance:

- `my-project` → `./data/projects/my-project/`
- `another-app` → `./data/projects/another-app/`

Fully isolated - no data leakage between projects!
