# BunBase Product Catalog

This document is the current product/service map for the monorepo, aligned to the implementation in code.

## Platform Surface

### `platform` (Control Plane API)
- Owns user sessions, project lifecycle, API keys, function deployment metadata, and developer-facing project APIs.
- Integrates with `bun-auth` for account auth, `tenant-auth` for per-project end-user auth config, `functions` for execution, and `bundoc-server` for document APIs.
- Supports both HTTP proxy integration and lower-latency RPC integration (`PLATFORM_BUNDOC_RPC_ADDR`, `PLATFORM_FUNCTIONS_RPC_ADDR`).

### `platform-web` (Dashboard)
- React + Vite dashboard for auth, project management, function deployment, logs, and database management UI.
- Uses `/v1` API base and cookie auth for console flows.

### `platform/cmd/cli` (`bunbase`)
- CLI for login, project selection, function scaffolding, and function deployment.
- Supports local function dev runner invocation (`bunbase dev`) and deploy to selected project.

## Runtime and Execution

### `functions`
- Serverless execution engine with long-lived worker pools, metadata/log storage, IPC control server, and HTTP gateway.
- Supports Bun worker and QuickJS worker paths in runtime internals.
- Accepts invoke traffic from Platform via HTTP or TCP IPC client mode.

## Data Products

### `bundoc`
- Embedded document database engine with MVCC, WAL, indexing, query execution, security modules, and raft package support.

### `bundoc-server`
- Multi-tenant HTTP and RPC wrapper over Bundoc instances.
- Performs per-project instance routing, collection/document/index CRUD, query, and optional Buncast event publishing.
- Supports optional internal RPC listener and optional raft node startup flags.

### `bunder`
- Redis-like key-value/data-structure store with RESP protocol, HTTP endpoints, persistence, and optional Buncast integration.

### `bunder-manager`
- Lazy project-scoped process manager for Bunder instances.
- Proxies `/kv/{project_id}/...` requests and handles process lifecycle/eviction.

## Messaging and Events

### `buncast`
- In-memory pub/sub service with IPC and HTTP/SSE interfaces.
- Used for deployment and data event fanout patterns.

## Security and Identity

### `bun-auth`
- System account authentication service backed by PostgreSQL.
- Provides register/login/verify endpoints and token issuance.

### `tenant-auth`
- Project-tenant end-user auth service for per-project users and auth config.
- Uses Bundoc (HTTP or RPC) for user/config persistence and optional BunKMS integration for provider secret storage.

### `bun-kms`
- Key management and secret storage service with HTTP and RPC interfaces.
- Provides secret put/get and crypto-key operations for other services.

## SDKs and Shared Libraries

### `bunbase-js`
- TypeScript SDK with auth, database, and functions clients on `/v1` endpoints.

### `pkg`
- Shared Go packages (`bunauth`, `bundocrpc`, `kmsrpc`, `config`, `logger`, `errors`) used by multiple services.

## Infrastructure Composition

### `docker-compose.yml`
- Defines full local stack: Platform, Platform Web, Functions, BunAuth, TenantAuth, Bundoc (data/auth), BunKMS, Buncast, Bunder, Postgres, MinIO, Loki, Prometheus, Grafana, Traefik.

### `deploy/traefik`
- Dynamic routing for API, dashboard, and generated/custom function domain invocation paths.
