# BunBase Service Roadmap

Last updated: February 7, 2026.

This roadmap is implementation-driven: milestones are organized by service and grounded in current code state.

## Planning Horizon
- `Now` (0-4 weeks): correctness, consistency, and contract hardening.
- `Next` (1-2 quarters): reliability, operability, and product completeness.
- `Later` (2+ quarters): scale-out capabilities and advanced platform features.

## Platform Core

## `platform`

### Now
- Consolidate duplicated `/api` and `/v1` route definitions into a smaller canonical handler map.
- Formalize per-endpoint auth requirements (session, bearer token, project key) into a single matrix and middleware contract tests.
- Add integration tests for deploy and invoke paths across HTTP and RPC downstream modes.

### Next
- Introduce structured request tracing IDs propagated to `functions`, `bundoc-server`, and auth services.
- Harden project-scoped authorization edge cases (mixed owner/member/key access).
- Add versioned API compatibility tests for critical endpoints used by `platform-web`, CLI, and `bunbase-js`.

### Later
- Control-plane multi-node readiness (connection management, lock/lease semantics, migrations governance).

## `platform-web`

### Now
- Align all client paths and env defaults with documented API path conventions.
- Normalize loading/error UX for auth, projects, functions, and database views.
- Add smoke tests for major user journeys (login -> create project -> deploy -> invoke).

### Next
- Improve operational UX: deploy progress, function health signals, and log exploration ergonomics.
- Add settings hardening flows for API key rotation and auth-provider secret lifecycle.

### Later
- Multi-project workspaces and richer role-based operator tooling.

## `platform/cmd/cli`

### Now
- Standardize command naming and docs (`login`/`logout`/`whoami` and `functions` subcommands).
- Add machine-readable output mode (`--json`) for CI pipelines.

### Next
- Add lifecycle parity commands (`functions list/delete/invoke/logs`).
- Add profile support for multiple control-plane environments.

### Later
- Release packaging and signed distribution pipeline.

## Compute and Runtime

## `functions`

### Now
- Publish explicit runtime capability matrix (Bun vs QuickJS) and enforce runtime validation in deploy/invoke paths.
- Add failure-mode tests for worker startup timeout, bundle load errors, and malformed responses.
- Improve function-level observability (per-invocation IDs, status classes, latency buckets).

### Next
- Introduce stronger sandbox controls (filesystem/network guards and capability allowlists).
- Add queue/backpressure controls for high-concurrency invoke traffic.
- Expand structured logs export/query interfaces for platform consumers.

### Later
- Distributed scheduling and multi-node worker execution.

## Data Plane

## `bundoc`

### Now
- Clean up stale API example docs and align package paths/examples with actual module structure.
- Add deterministic regression tests for query/index behavior under update/delete-heavy workloads.

### Next
- Expand advanced query operator support and execution-plan observability.
- Build snapshot/log compaction strategy for raft-backed use cases.

### Later
- Sharding/partitioning model for horizontal data distribution.

## `bundoc-server`

### Now
- Replace path-suffix routing branching with explicit route table and parser utilities.
- Add route contract tests for all collection/document/index/query endpoints.
- Clarify HTTP vs RPC behavior in server docs with exact examples.

### Next
- Add stronger realtime event contracts (shape/versioning) when publishing through Buncast.
- Introduce operational metrics for instance manager churn and request latency.

### Later
- Clustered control strategy for multi-node tenancy hosting.

## `bunder`

### Now
- Validate RESP and HTTP parity on core command set with compatibility tests.
- Clarify persistence mode guarantees and restart semantics in docs.

### Next
- Expand command coverage and finalize pub/sub/SSE behavior.
- Add performance dashboards and stability test profiles.

### Later
- Horizontal scaling and slot/partition awareness.

## `bunder-manager`

### Now
- Add robustness tests for process spawn failures, port exhaustion, and concurrent first-request races.
- Document and enforce instance eviction policy guarantees.

### Next
- Add instance health remediation and automatic replacement logic.

### Later
- Orchestrator integration for managed process lifecycle.

## Messaging

## `buncast`

### Now
- Add Prometheus metrics for topic/subscriber/message counts.
- Add integration tests for long-lived SSE subscriptions and reconnect behavior.

### Next
- Define canonical event envelope schema and versioning policy.

### Later
- Optional durable stream/replay mode for selected topics.

## Identity and Security

## `bun-auth`

### Now
- Replace development HS256 static secret with environment-controlled key strategy and rotation plan.
- Add endpoint-level tests for duplicate registration, invalid credentials, and verify edge cases.

### Next
- Introduce refresh/session lifecycle policies and revocation semantics.

### Later
- External identity provider federation support.

## `tenant-auth`

### Now
- Harden token/key management and document production defaults.
- Add integration coverage for Bundoc/KMS HTTP and RPC fallback behavior.

### Next
- Expand auth provider configuration model and auditability.

### Later
- Fine-grained tenant auth policy engine.

## `bun-kms`

### Now
- Formalize production deployment guide for master key handling and auth configuration.
- Add integration tests for HTTP and RPC parity on secret operations.

### Next
- Strengthen audit log querying and key lifecycle tooling.

### Later
- HSM-backed key material workflows.

## SDK and Shared Libraries

## `bunbase-js`

### Now
- Align method surface with actual platform route contracts and remove stale endpoint assumptions.
- Add test coverage for auth/database/functions clients and SSE behavior.

### Next
- Add better typed schema ergonomics and runtime validation helpers.
- Publish versioned compatibility table against Platform API versions.

### Later
- Additional platform modules (tokens, project config helpers, admin flows).

## `pkg`

### Now
- Add tests and usage docs for `bunauth`, `bundocrpc`, and `kmsrpc` shared clients.

### Next
- Define shared protocol contracts to reduce duplicated request/response shapes across services.

### Later
- Extract stable protocol package for cross-service internal APIs.

## Cross-Service Milestones

## Milestone A: Contract Stability (Now)
- Deliverables:
- Route and auth contract matrix for Platform.
- API path alignment across Platform, Platform Web, CLI, and SDK.
- Basic integration tests for deploy/invoke/data proxy flows.

## Milestone B: Reliability and Visibility (Next)
- Deliverables:
- End-to-end request tracing.
- Common metrics dashboards for control/data/runtime services.
- Hardened fallback and failure semantics for RPC/HTTP modes.

## Milestone C: Scale and Isolation (Later)
- Deliverables:
- Stronger runtime isolation.
- Multi-node-ready control and runtime services.
- Data partitioning and messaging durability options.
