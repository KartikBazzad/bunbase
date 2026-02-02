# Platform Integration Plan (Docker, Auth, Postgres, Monitoring)

## Goal Description
Integrate all BunBase services using Docker Compose with a robust production stack: PostgreSQL (Data), MinIO (Storage), Redis/Bunder (Cache), Prometheus/Grafana (Monitoring), and Traefik (Gateway).

## User Review Required
> [!IMPORTANT]
> **Stack Update**: Moving from SQLite/Local-FS to **PosegreSQL** and **MinIO**.
> **Monitoring**: Adding **Prometheus** and **Grafana**.
> **Auth**: New `bun-auth` service backed by Postgres.

## Proposed Changes

### 1. New Service: `bun-auth` (System)
- **Path**: `bun-auth/`
- **Tech**: Go, PostgreSQL.
- **Specification**: `planning/bun_auth_service.md`

### 2. New Service: `tenant-auth` (Project)
- **Path**: `tenant-auth/` (or part of Platform?)
- **Specification**: `planning/project_auth_requirements.md`

### 3. Service Containerization
Create `Dockerfile` for each service:
- `platform` (Migrate to Postgres)
- `functions`
- `bundoc-server`
- `bunder-manager`
- `bun-kms`
- `buncast`
- `bun-auth` [NEW]

### 3. Orchestration
#### [NEW] [docker-compose.yml](file:///Users/kartikbazzad/Desktop/projects/bunbase/docker-compose.yml)
- **Infrastructure**: Postgres, MinIO, Bunder (Redis), Prometheus, Grafana.
- **Services**: All BunBase apps.
- **Gateway**: Traefik (:80).

### 4. Application Logic Updates

#### [MODIFY] [Platform Service]
- **DB**: Migrate GORM/SQL from SQLite to Postgres (`planning/postgres_schemas.md`).
- **Auth**: Use `bun-auth` RPC.

#### [MODIFY] [Bundoc Server]
- **Encryption**: Integrate `bun-kms` (`planning/kms_bundoc_integration.md`).
- **Monitoring**: Add Prometheus metrics (`planning/monitoring_requirements.md`).

### 5. Client SDKs & CLI (Future Phase)
- **bunbase-js**: Browser/React Native Support (`planning/sdk_requirements.md`).
- **bunbase-admin**: Node.js/Server Support.
- **bunbase-cli**: Developer Tooling (`planning/cli_requirements.md`).

## Roadmap (`planning/roadmap_phases.md`)
1.  **Foundation**: Docker, Auth, Postgres.
2.  **Core Refactor**: Platform, Functions Isolation, KMS.
3.  **Clients**: SDKs & CLI.
4.  **Observability**: Monitoring.

## Verification Plan
1.  **Build**: `docker-compose build`.
2.  **Up**: `docker-compose up -d`.
3.  **Health**: Check all services via `http://localhost:PORT/health`.
4.  **Metrics**: Verify targets in `http://localhost:9090/targets`.
