## BunBase Monorepo Structure

This document defines how the BunBase repository is organized and how new components should be added.

### Top-Level Layout

- `docdb/` – Go DocDB database engine and shell, plus a TypeScript client.
- `functions/` – Go-based control plane for running Bun workers that execute JS/TS functions.
- `platform/` – Go Platform API (auth, projects, function deployment, integration with Functions).
- `platform-web/` – React + Vite + Tailwind dashboard for the BunBase Platform.
- `buncast/` – Go Pub/Sub service (event bus for cross-service events).
- `requirements/` – Product and feature requirements (this is where `platform.md` lives).
- `planning/` – Plans, RFCs, and design documents (this file, shared libraries plan, etc.).
- `docs/` – Cross-cutting architecture, development workflow, and onboarding docs.

Service-specific docs remain in their respective directories (for example, `docdb/docs`, `functions/docs`).

### Goals

- Treat this repository as a **single monorepo** while allowing each service to remain independently buildable and testable.
- Provide a **single entrypoint** for common build/run tasks via the root `Makefile`.
- Standardize on **Bun** for all JS/TS tooling (frontend and future shared packages).
- Avoid premature code moves; introduce shared libraries only when clear duplication appears.

### Shared Code Strategy

Short term:

- Keep all Go and JS/TS code in their existing service directories.
- Use shared documentation (`requirements/`, `planning/`, `docs/`) to coordinate work across services.

Medium term:

- Introduce a top-level `packages/` directory for shared TypeScript/JavaScript libraries (e.g. DocDB TS client, shared types, UI kit).
- Introduce a small shared Go module (for example `internal/shared/`) if meaningful duplication emerges across services.

See `planning/shared-libraries.md` for details once created.

### Adding New Components

When adding a new app or service:

1. Place its code in a new top-level directory (for example, `worker-agent/`, `admin-web/`).
2. Add its build/run commands to the root `Makefile`.
3. Document its purpose and how it fits into the platform in:
   - A `README.md` inside the new directory.
   - The architecture overview in `docs/architecture.md` (link to the new README).
4. If it introduces cross-cutting behavior, add a short design/RFC in `planning/`.

### Documentation Conventions

- Use `requirements/` for product-facing requirements and user flows.
- Use `planning/` for engineering designs, RFCs, and implementation plans.
- Use `docs/` for system-wide architecture and developer onboarding.
- Keep service-local operational details in each service’s own `docs/` or `README.md`.
