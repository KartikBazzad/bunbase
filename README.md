# BunBase

**The Open Source Serverless Platform for the Bun Era.**

BunBase is a self-hostable, high-performance serverless platform. Backend services (auth, database, storage, platform API) are written in **Go**. **Bun** is used only for **serverless function execution**—your app code runs on Bun (or QuickJS). You get a full suite of Authentication, Document Store, Storage, and Functions, orchestrated via a unified CLI and Console.

## Core Services

- **BunAuth**: Centralized Identity & Access Management (Postgres + JWT).
- **Bundoc**: Real-time Document Store (JSON-RPC).
- **Functions**: Serverless Function Execution (Bun / QuickJS runtimes—Bun only here).
- **BunKMS**: Key Management Service & Encryption.
- **Platform API**: Project Management & Control Plane.

## Infrastructure Stack

- **Runtime**: Go (all system services); Bun / QuickJS (serverless functions only).
- **Database**: PostgreSQL (System Data), Bundoc (User Data).
- **Storage**: MinIO (S3 Compatible).
- **Monitoring**: Prometheus & Grafana.
- **Gateway**: Traefik.

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.22+
- Bun 1.1+

### Local Development

1.  **Clone**: `git clone https://github.com/kartikbazzad/bunbase`
2.  **Start Stack**: `docker compose up -d` (or `./scripts/deploy-cloud.sh` for Cloud mode with health wait)
3.  **CLI**: `go install ./cmd/bunbase`

### Deploy as Cloud

To run in **Cloud mode** (any user can sign up and create projects), start the stack as above—Cloud is the default. See [Deploy BunBase in Cloud mode](docs/deploy-cloud.md) for the full steps and dashboard URL.

## Documentation

- [Documentation Index](docs/README.md)
- [Product Catalog](docs/products/README.md)
- [Service Requirements](requirements/services.md)
- [Service Implementation](planning/service-implementation.md)
- [Service Roadmap](planning/service-roadmap.md)

NOTE:- VIBE CODED platform.
