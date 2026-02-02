# BunKMS Deployment

## Environment Variables

| Variable                | Default    | Description                                       |
| ----------------------- | ---------- | ------------------------------------------------- |
| BUNKMS_ADDR             | :8080      | HTTP listen address                               |
| BUNKMS_MASTER_KEY       | (required) | 32-byte key: `base64:<b64>`, hex, or raw 32 chars |
| BUNKMS_DATA_PATH        | (empty)    | If set, use Bunder persistence at this path       |
| BUNKMS_AUDIT_LOG        | (empty)    | If set, append audit events to this file          |
| BUNKMS_JWT_SECRET       | (empty)    | If set, require Bearer JWT for /v1/\*             |
| BUNKMS_BUFFER_POOL_SIZE | 10000      | Bunder buffer pool size                           |
| BUNKMS_SHARDS           | 256        | Bunder shard count                                |

## Binary

```bash
go build -o bunkms ./cmd/server
BUNKMS_MASTER_KEY=base64:$(echo -n "32-byte-master-key-here!!!!!!!!" | base64) ./bunkms
```

## Docker

From repo root (so Bunder is in context):

```bash
docker build -f bun-kms/Dockerfile -t bunkms:latest .
docker run -p 8080:8080 -e BUNKMS_MASTER_KEY=base64:... -e BUNKMS_DATA_PATH=/data -v bunkms-data:/data bunkms:latest
```

## Docker Compose

From `bun-kms/`:

```bash
docker-compose up -d
```

Uses a volume for `/data` (Bunder + audit log). Set `BUNKMS_MASTER_KEY` in the compose file or `.env`.

## Graceful Shutdown

The server handles SIGTERM/SIGINT: it stops accepting new requests, waits for in-flight requests (up to 10s), then closes the store and audit log.

## Health Checks

- Liveness: `GET /health` (200 when master key loaded; 503 if storage unavailable when configured).
- Readiness: `GET /ready` (200 when storage is ok or not used).

Use these in Kubernetes or load balancers.
