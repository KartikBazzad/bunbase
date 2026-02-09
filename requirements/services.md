# BunBase Service Requirements

This is the consolidated product requirements document for all first-party BunBase services in this repository.

## Requirement Levels

- `P0`: Required for a production-usable core platform
- `P1`: High-value near-term requirement
- `P2`: Important but deferrable

## Cross-Platform Requirements

### Functional
- `P0` Unified project model across Platform, Functions, Bundoc, and auth services.
- `P0` Stable API contracts for dashboard, CLI, and SDK clients.
- `P0` End-to-end deploy and invoke flow for project-scoped functions.
- `P1` Project-scoped data APIs with key-based and user-auth access modes.
- `P1` Event publication for major lifecycle actions (deployments, data changes).

### Non-Functional
- `P0` Consistent health endpoints and startup failure visibility.
- `P0` Service-level logs suitable for incident triage.
- `P1` Prometheus-ready metrics on core control/data planes.
- `P1` Backward-compatible API evolution policy for `/v1`.
- `P2` Automated architecture and API docs generation.

## Service Requirements

## `platform` (Control Plane API)

### Deployment modes
- **Cloud** (`PLATFORM_DEPLOYMENT_MODE=cloud` or unset): Any user can sign up and create projects. Authorization is enforced via Casbin (instance-level and project-level).
- **Self-hosted** (`PLATFORM_DEPLOYMENT_MODE=self_hosted`): One-time setup creates a root admin; only that admin can create projects. Signup is disabled after setup. Same Casbin model; policies restrict create_project to instance admins. Future: team invites via email.

### Functional
- `P0` User account lifecycle: register, login, logout, current-session introspection.
- `P0` Project CRUD and membership/ownership authorization checks (Casbin).
- `P0` Self-hosted: bootstrap endpoint `POST /api/setup`, instance status `GET /api/instance/status`, and signup/create-project gating by deployment mode.
- `P0` Function deployment orchestration into `functions` service.
- `P0` Project API key generation and regeneration.
- `P1` Developer database proxy APIs (collection, document, query, index, rules).
- `P1` Token management APIs for non-cookie auth clients.
- `P1` Function logs retrieval and filtering by project/function.
- `P1` Custom domain/function host routing to project-scoped invokes.

### Non-Functional
- `P0` PostgreSQL-backed persistence with startup migrations.
- `P0` Configurable CORS and environment-driven service integration.
- `P1` RPC fallback behavior when downstream RPC addresses are unset.
- `P1` Request correlation between Platform and downstream services.

## `platform-web` (Dashboard)

### Functional
- `P0` Cookie-session login/signup/logout.
- `P0` Project list/create/detail/delete flows.
- `P0` Function deploy/list/delete and invoke UX.
- `P1` Database browser/editor for collections, documents, indexes, and rules.
- `P1` Project auth configuration and project user management UI.
- `P1` Function log viewing with practical filters.

### Non-Functional
- `P0` Correct API base/path behavior in local and containerized environments.
- `P1` Loading/error states for all async user paths.
- `P1` Build output suitable for static hosting behind Traefik/Nginx.

## `platform/cmd/cli` (`bunbase`)

### Functional
- `P0` Non-interactive login via flags and stored session persistence.
- `P0` Project list/create/use commands.
- `P0` Function scaffold (`functions init`) and deploy (`functions deploy`).
- `P1` Function list/get/delete parity with API.
- `P1` Explicit profile/environment support for multiple control-plane targets.

### Non-Functional
- `P0` Stable local config file format and secure file permissions.
- `P1` Predictable machine-readable output mode (`--json`).

## `functions` (Execution Engine)

### Functional
- `P0` Register/deploy/invoke lifecycle via IPC and HTTP gateway.
- `P0` Worker pool management with warm/cold transitions and timeouts.
- `P0` Per-invocation request/response semantics compatible with Web `Request`/`Response`.
- `P1` Runtime selection policy (`bun`, `quickjs-ng`) with clear capability constraints.
- `P1` Execution logs and metrics persistence/query.
- `P1` Admin SDK context injection (`project_id`, `api_key`, `gateway_url`).

### Non-Functional
- `P0` Bounded execution resources (timeout, memory, worker count).
- `P0` Safe process lifecycle handling and graceful shutdown.
- `P1` Startup/runtime diagnostics that identify worker script/runtime mismatch.

## `bundoc` (Database Engine)

### Functional
- `P0` ACID CRUD with MVCC transaction semantics.
- `P0` Secondary indexes and query execution for common operators.
- `P1` Schema/rules integration hooks for server-facing enforcement.
- `P1` **Cross-collection references** (schema extension `x-bundoc-ref`, strict FK at write time, `on_delete`: restrict / set_null / cascade). Implemented.
- `P1` Recovery and metadata durability across restarts.
- `P2` Advanced query operators and partitioning primitives.

### Non-Functional
- `P0` WAL durability and recoverability guarantees.
- `P0` Performance envelope for mixed read/write workloads.
- `P1` Deterministic behavior under concurrent transaction load.

## `bundoc-server` (Multi-Tenant Data API)

### Functional
- `P0` Project-scoped collection/document/index CRUD over HTTP.
- `P0` Query endpoint with project isolation.
- `P0` Instance manager with lazy activation and idle eviction.
- `P1` Internal RPC proxy API for lower-latency service integration.
- `P1` Optional realtime publication hooks to Buncast.

### Non-Functional
- `P0` Strict project isolation in storage paths and request routing.
- `P1` Stable path parsing and routing correctness for all supported endpoint forms.
- `P1` Graceful shutdown behavior for open instances.

## `bunder` (KV/Data Structure Store)

### Functional
- `P0` Core RESP command support and HTTP API parity for key-value operations.
- `P1` Data structure command coverage (list/set/hash) and TTL semantics.
- `P1` Persistence modes with predictable restart recovery.
- `P2` Full pub/sub semantics beyond placeholder endpoints.

### Non-Functional
- `P0` High-concurrency correctness in sharded storage paths.
- `P1` Operational metrics and health endpoints for control-plane integration.

## `bunder-manager`

### Functional
- `P0` Project-scoped process allocation on first request.
- `P0` Request proxying to correct project instance.
- `P1` Idle eviction and bounded port-pool management.
- `P1` Process restart handling and unhealthy child detection.

### Non-Functional
- `P0` No cross-project request/data leakage.
- `P1` Fast cold-start provisioning with backpressure protections.

## `buncast`

### Functional
- `P0` Topic create/delete/list and publish/subscribe operations over IPC.
- `P0` HTTP endpoints for health/topics and SSE subscriptions.
- `P1` Standardized event payload shape guidance for producing services.
- `P1` Consumer resilience semantics for disconnected subscribers.

### Non-Functional
- `P0` In-order delivery per subscriber/topic for in-memory mode.
- `P1` Message throughput telemetry (publish/deliver/subscriber counts).

## `bun-auth`

### Functional
- `P0` User registration and credential login against PostgreSQL.
- `P0` Token verification endpoint with user profile return.
- `P1` Session and refresh token lifecycle aligned with Platform expectations.
- `P1` Key management model that supports algorithm rotation.

### Non-Functional
- `P0` Password hashing with modern secure defaults.
- `P0` Deterministic behavior under duplicate-email and bad-credential flows.
- `P1` Removal of hardcoded signing secrets in production paths.

## `tenant-auth`

### Functional
- `P0` Project-scoped end-user register/login/verify.
- `P0` Project auth config read/update endpoints.
- `P1` Provider secret indirection using BunKMS references.
- `P1` Admin user-list APIs for project operators.

### Non-Functional
- `P0` Project isolation in user/config lookups.
- `P1` Clean fallback between HTTP and RPC clients for Bundoc/KMS.

## `bun-kms`

### Functional
- `P0` Secret put/get APIs and secure persistence behavior.
- `P1` Key generation/rotation and crypto operation APIs.
- `P1` Audit log capture for secret/key operations.
- `P1` RPC parity with HTTP for dependent services.

### Non-Functional
- `P0` Encryption-at-rest and master-key boot requirements.
- `P1` Clear failure semantics when misconfigured keys or auth are present.

## `bunbase-js`

### Functional
- `P0` Client initialization with project API key and `/v1` paths.
- `P0` Database collection/document CRUD and function invoke wrappers.
- `P1` Realtime subscriptions for collection/query update streams.
- `P1` Auth utility methods for app-level onboarding flows.
- `P2` Typed schema registry ergonomics and runtime validation helpers.

### Non-Functional
- `P0` Browser-safe transport behavior.
- `P1` SDK/API version compatibility policy and deprecation notes.

## `pkg` (Shared Go Libraries)

### Functional
- `P0` Stable shared client packages used by multiple services.
- `P1` Shared request/response envelope typing where protocols overlap.

### Non-Functional
- `P0` Backward compatibility for packages consumed in the monorepo.
- `P1` Test coverage for wire-protocol clients (`bundocrpc`, `kmsrpc`, `bunauth`).

## Acceptance Criteria Template (All Services)

Each requirement promoted to implementation must include:
- explicit API/CLI/interface contract,
- automated tests covering success and failure paths,
- operational validation (health/log/metric visibility),
- migration and rollback notes when persistence or protocols change.
