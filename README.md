# BunBase

**The Open Source Serverless Platform for the Bun Era.**

BunBase is a self-hostable, high-performance serverless platform built on top of Bun. It provides a complete suite of backend services including Authentication, Database (Document Store), Storage, and Functions, all orchestrated via a unified CLI and Console.

## Core Services

-   **BunAuth**: Centralized Identity & Access Management (Postgres + JWT).
-   **Bundoc**: Real-time Document Store (JSON-RPC).
-   **Functions**: Serverless Function Execution (Bun/QuickJS Runtimes).
-   **BunKMS**: Key Management Service & Encryption.
-   **Platform API**: Project Management & Control Plane.

## Infrastructure Stack

-   **Runtime**: Bun (JavaScript/TypeScript), Go (System Services).
-   **Database**: PostgreSQL (System Data), Bundoc (User Data).
-   **Storage**: MinIO (S3 Compatible).
-   **Monitoring**: Prometheus & Grafana.
-   **Gateway**: Traefik.

## Getting Started

### Prerequisites
-   Docker & Docker Compose
-   Go 1.22+
-   Bun 1.1+

### Local Development
1.  **Clone**: `git clone https://github.com/kartikbazzad/bunbase`
2.  **Start Stack**: `docker-compose up -d`
3.  **CLI**: `go install ./cmd/bunbase`

## Documentation
-   [Documentation Index](docs/README.md)
-   [Product Catalog](docs/products/README.md)
-   [Service Requirements](requirements/services.md)
-   [Service Implementation](planning/service-implementation.md)
-   [Service Roadmap](planning/service-roadmap.md)
