## Shared Libraries Plan

This document outlines how we plan to share code across services in the BunBase monorepo.

### Motivation

Several pieces of functionality are naturally shared across components:

- TypeScript clients and type definitions used by `platform-web/` and other tools.
- Go client code and utilities shared between `platform/` and `functions/`.
- Potential shared UI primitives for future web UIs.

The goal is to centralize these into clearly versioned libraries without over-complicating the current codebase.

### TypeScript / JavaScript Libraries

We will eventually introduce a top-level `packages/` directory for JS/TS shared code. Candidate packages:

- `packages/docdb-client-ts/`
  - Extracted from `docdb/tsclient`.
  - Published (or locally referenced) as a reusable TypeScript client for DocDB.
  - Consumed by `platform-web/` and any other JS/TS consumers.

- `packages/platform-types/`
  - Shared TypeScript types for API payloads (auth, projects, functions).
  - Consumed by `platform-web/` and any CLIs or tools that talk to the Platform API.

All JS/TS packages will:

- Use Bun for scripts (`bun install`, `bun run build`, etc.).
- Be developed and tested in this monorepo.

### Go Shared Code

**Current Setup**: We use **Go workspaces** (`go.work` at repo root) to enable cross-module imports between `docdb`, `functions`, and `platform`.

**How it works:**

- Each service remains its own Go module
- `go.work` makes them part of a single workspace
- Services can import each other using module paths (e.g., `platform` can import `github.com/kartikbazzad/bunbase/functions/pkg/client`)

**See**: `docs/sharing-code-between-services.md` for the complete guide on how to share code.

**Future shared packages:**

- `packages/shared-go/` – Common helpers used by multiple Go services (logging wrappers, error helpers, configuration parsing, etc.).
- Shared client libraries:
  - Functions client is currently in `functions/pkg/client` (can be imported by `platform`).
  - Platform-specific helpers may live in `platform/pkg` and can be reused as needed.

Any shared Go module should:

- Stay internal to this repository (no external import path stability guarantees yet).
- Have a narrow and well-documented surface area.

### When to Create a Shared Library

Use a shared library when:

- The same code or types are copy-pasted into more than one service or app.
- Changes to the logic or types would require edits in multiple places.
- The shared code has a clear domain (e.g. DocDB client, platform API models).

Avoid a shared library when:

- The code is only used in a single service.
- The abstraction is unclear or still evolving rapidly.

### Migration Strategy

Short term:

- Keep existing code where it is (`docdb/tsclient`, `functions/pkg/client`, etc.).
- Reference this document when deciding whether a new piece of code should be shared.

Medium term:

- Introduce `packages/` once there is a clear candidate (e.g. DocDB TS client).
- Move the minimum necessary code into the new package.
- Update consumers (`platform-web/`, CLIs) to import from the shared package.

Long term:

- Stabilize package APIs.
- Consider publishing packages to a registry once the surface area is stable.
