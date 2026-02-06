# Inter-service RPC

Services can talk to Bundoc, Functions, and KMS over **TCP RPC** (length-prefixed frames, JSON payloads) for lower latency when running in the same network (e.g. Docker). HTTP remains supported and is used when RPC is not configured.

## Overview

| Consumer    | Backend   | RPC env (consumer)        | Server env (backend)   | Default in compose |
|------------|-----------|----------------------------|------------------------|--------------------|
| Platform   | Bundoc    | `PLATFORM_BUNDOC_RPC_ADDR` | `BUNDOC_RPC_ADDR`      | Yes (bundoc-data, bundoc-auth) |
| Platform   | Functions | `PLATFORM_FUNCTIONS_RPC_ADDR` | `FUNCTIONS_TCP_ADDR` | Yes |
| Tenant-auth| Bundoc    | `TENANTAUTH_BUNDOC_RPC_ADDR` | `BUNDOC_RPC_ADDR`   | Yes (bundoc-auth) |
| Tenant-auth| KMS       | `TENANTAUTH_BUNKMS_RPC_ADDR` | `BUNKMS_RPC_ADDR`   | Yes (bunkms) |

If the RPC env is **not** set, the consumer falls back to HTTP (Bundoc URL, KMS URL, or Functions HTTP/socket as before).

## Bundoc RPC

- **Server**: `bundoc-server` starts a TCP listener when `BUNDOC_RPC_ADDR` is set (e.g. `:9091`).
- **Protocol**: Length-prefixed frames (4-byte length, then request ID, command byte, payload length, JSON payload). Command `1` = proxy document request (method, project_id, path, body base64).
- **Clients**:
  - **Platform**: `pkg/bundocrpc` → `ProxyRequest(method, projectID, path, body)`.
  - **Tenant-auth**: `tenant-auth/internal/db/rpc_db.go` uses `pkg/bundocrpc` for user/settings CRUD against Bundoc.

## Functions RPC

- **Server**: `functions` listens on `FUNCTIONS_TCP_ADDR` (e.g. `:9090`) using the existing IPC protocol.
- **Client**: Platform uses the functions IPC client with a `tcp://host:port` address when `PLATFORM_FUNCTIONS_RPC_ADDR` is set.

## KMS RPC

- **Server**: `bun-kms` starts a TCP RPC server when `BUNKMS_RPC_ADDR` is set (e.g. `:9092`). Commands: GetSecret (1), PutSecret (2); JSON payloads with `name` and optional `value_b64`.
- **Client**: `pkg/kmsrpc` → `GetSecret(name)`, `PutSecret(name, value)`. Tenant-auth uses it via `tenant-auth/internal/kms/rpc_client.go` when `TENANTAUTH_BUNKMS_RPC_ADDR` is set.

## Docker Compose

- **bundoc-data**: `BUNDOC_RPC_ADDR: ":9091"`, ports `8085:8080`, `9091:9091`.
- **bundoc-auth**: `BUNDOC_RPC_ADDR: ":9091"`, ports `8084:8080` (and optionally `9091:9091` for host access).
- **tenant-auth**: `TENANTAUTH_BUNDOC_RPC_ADDR: bundoc-auth:9091`, `TENANTAUTH_BUNKMS_RPC_ADDR: bunkms:9092`.
- **bunkms**: `BUNKMS_RPC_ADDR: ":9092"`, ports `8087:8080`, `9092:9092`.
- **platform**: `PLATFORM_BUNDOC_RPC_ADDR`, `PLATFORM_FUNCTIONS_RPC_ADDR` set to service names and RPC ports when using RPC.

See `docker-compose.yml` for the exact values.
