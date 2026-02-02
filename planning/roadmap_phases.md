# BunBase Project Roadmap

This roadmap organizes the implementation of all planned features into 5 logical phases.

## Phase 1: Foundation (Infrastructure)
**Goal**: Establish the core runtime environment and database.

1.  **Shared Libraries (`pkg/`)**:
    -   Setup `go.work` workspace.
    -   Create `pkg/logger`, `pkg/config`, `pkg/errors`.
2.  **Docker Orchestration**:
    -   Create `docker-compose.yml`.
    -   Configure `postgres`, `minio`, `bunder` (Redis), `prometheus`, `grafana`.
    -   Setup `traefik` gateway.
2.  **BunAuth Service**:
    -   Implement Go service with Postgres backend.
    -   Implement JWT issuance (RS256).
    -   Implement RPC endpoints (`Login`, `Verify`).

## Phase 2: Core Services Refactor
**Goal**: Migrate existing services to the new infrastructure.

1.  **Platform Service**:
    -   Migrate DB from SQLite to Postgres.
    -   Integrate with `bun-auth` for user verification.
2.  **Project Auth (`tenant-auth`)**:
    -   Implement Tenant Authentication (Email/Password, JWT).
    -   Integrate with BunKMS for token signing.
3.  **Functions Service**:
    -   Implement **Runtime Isolation** (Preload script).
    -   Isolate `bun` runtime from file system/network.
4.  **Bundoc**:
    -   Implement **BunKMS Integration** (Fetch DEK from KMS).
    -   Implement Envelope Encryption.

## Phase 3: Client Access (SDKs, CLI & Console)
**Goal**: Enable developers to build apps on BunBase.

1.  **Client SDK (`bunbase-js`)**:
    -   Implement `auth` module (Login/Logout).
    -   Implement `data` module (Document Store API).
    -   Implement `functions` module (HTTP caller).
2.  **Admin SDK (`bunbase-admin`)**:
    -   Implement privileged access methods.
    -   Server-to-Server token generation.
3.  **CLI (`bunbase`)**:
    -   Implement `login`, `deploy`, `logs`, `env`.
    -   Support local config (`bunbase.toml`).
4.  **Web Console (`platform-web`)**:
    -   Rewritten in React/Vite/Tailwind.
    -   Dashboard, Data Browser, Function Logs.
5.  **Documentation (`docs/`)**:
    -   Fumadocs (Next.js) site.
    -   Auto-generated API References.

## Phase 4: Observability & Production Readiness
**Goal**: Ensure system is monitorable and stable.

1.  **Monitoring**:
    -   Instrument all Go services with Prometheus metrics.
    -   Create Grafana dashboards.
2.  **Testing**:
    -   End-to-End integration tests (Login -> Deploy Function -> Invoke).

## Phase 5: Future Enhancements (Post-MVP)
1.  **Buncast**: Real-time updates for Bundoc (Live Connect).
2.  **Billing**: Integrate Stripe for usage-based billing.
3.  **Global Edge**: Deploy workers to multiple regions (requires complex orchestrator).
