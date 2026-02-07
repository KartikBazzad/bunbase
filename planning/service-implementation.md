# BunBase Service Implementation Baseline

Last updated: February 7, 2026.

This document captures current implementation state by service based on code in this repository.

## Status Definitions
- `Implemented`: exists in code and wired in primary runtime path.
- `Partial`: exists but with known scope gaps or non-final behavior.
- `Planned`: captured in docs/plans but not materially implemented.

## Platform and Interfaces

## `platform`
- `Implemented` PostgreSQL-backed API server initialization and migration startup.
- `Implemented` Account flows via `bun-auth` client integration.
- `Implemented` Project CRUD, API key regeneration, and ownership/member checks.
- `Implemented` Function deployment/list/delete/invoke endpoints.
- `Implemented` Database proxy endpoints for project/user and key-scoped paths.
- `Implemented` Optional bundoc/functions RPC usage controlled by environment.
- `Partial` Route surface is broad and duplicated between `/api` and `/v1`; requires long-term consolidation.

## `platform-web`
- `Implemented` React/Vite application with auth, projects, functions, and database views.
- `Implemented` API client defaults to `/v1` base and cookie credentials.
- `Implemented` Container build sets `VITE_API_URL=/api`, relying on matching server route aliases.
- `Partial` UX hardening (error states/loading behavior consistency) is uneven across flows.

## `platform/cmd/cli`
- `Implemented` Login (flag-driven), project list/create/use, whoami.
- `Implemented` Function init template generation and deploy command.
- `Implemented` Dev-runner wrapper command.
- `Partial` Command naming in user docs and command tree is not fully aligned (`auth login` vs `login` usage in code paths).

## Compute and Runtime

## `functions`
- `Implemented` IPC server, HTTP gateway, scheduler/router wiring, metadata initialization.
- `Implemented` Bun worker lifecycle and invoke flow.
- `Implemented` QuickJS worker integration path in runtime and deployment tooling.
- `Implemented` Metadata/log/metrics SQLite stores.
- `Partial` Security model is process-level with trusted-host assumptions; not hard-isolated.
- `Partial` Runtime capability model exists but needs clearer externally documented guarantees.

## Data Services

## `bundoc`
- `Implemented` MVCC transactions, WAL durability, collection/document CRUD.
- `Implemented` Secondary indexing and query planner/execution pieces.
- `Implemented` Raft package support in engine module.
- `Partial` Documentation includes outdated import paths/examples in some files.

## `bundoc-server`
- `Implemented` Multi-tenant HTTP path handling under `/v1/projects/...`.
- `Implemented` Project instance manager and data path isolation.
- `Implemented` Optional internal TCP RPC server for proxy request execution.
- `Implemented` Optional raft node and TCP server startup.
- `Partial` HTTP routing logic is path-string driven and complex; should be normalized for maintainability.

## `bunder`
- `Implemented` RESP/HTTP server, persistence components, TTL, core data structures.
- `Implemented` Load-test command and package tests.
- `Partial` SSE/pubsub behavior and production hardening are not fully complete.

## `bunder-manager`
- `Implemented` Lazy per-project process spawn and request proxying.
- `Implemented` Port pool and data directory mapping.
- `Partial` Operational behaviors (restart policy, stronger failure isolation) need expansion.

## Messaging

## `buncast`
- `Implemented` In-memory broker, IPC server/client, HTTP/SSE endpoints.
- `Implemented` Platform deploy-event publication integration.
- `Partial` Prometheus metrics and durable/replay semantics are not in the current core.

## Security and Identity

## `bun-auth`
- `Implemented` PostgreSQL user/session schema migration bootstrap.
- `Implemented` Register/login/verify HTTP endpoints.
- `Partial` Token strategy currently uses HS256 with static development secret in handler path; production-grade key strategy not completed.

## `tenant-auth`
- `Implemented` Project user register/login/verify flows.
- `Implemented` Project auth config get/update and user listing routes.
- `Implemented` Bundoc HTTP/RPC client modes.
- `Implemented` BunKMS HTTP/RPC integration for provider secret references.
- `Partial` JWT model is HS256 secret-driven and requires hardening/rotation strategy.

## `bun-kms`
- `Implemented` Service, docs, CLI/loadtest tooling, secret/key operation surfaces.
- `Implemented` RPC endpoint support and docker-compose wiring.
- `Partial` Production auth and policy defaults depend on deployment-time configuration choices.

## SDK and Shared Libraries

## `bunbase-js`
- `Implemented` `createClient`, auth/database/functions client modules.
- `Implemented` Key-based request model and SSE subscription helpers.
- `Partial` Some method coverage and endpoint parity are incomplete (for example, delete function path in SDK does not match platform route surface).

## `pkg`
- `Implemented` Shared config/logger/errors and auth/rpc clients (`bunauth`, `bundocrpc`, `kmsrpc`).
- `Partial` Common contract typing across services is still distributed rather than centralized.

## Delivery and Infrastructure

## `docker-compose.yml`
- `Implemented` Integrated local stack for API, web, auth, functions, data, KMS, messaging, and observability services.
- `Implemented` Health checks and service dependency graph.
- `Partial` Documentation around published ports/entrypoints is inconsistent in older docs and needs convergence on compose as source of truth.

## Key Document Gaps Addressed in This Update
- Corrected top-level documentation index references away from stale `docdb` paths.
- Added unified product catalog (`docs/products/README.md`).
- Added consolidated service requirements (`requirements/services.md`).
- Added detailed service roadmap (`planning/service-roadmap.md`).
