## BunBase Development Workflow

This document describes how to work with the BunBase monorepo in day-to-day development.

### Prerequisites

- Go 1.21+ installed (`go` on PATH).
- Bun runtime installed (`bun` on PATH).
- SQLite3 installed.

### Top-Level Commands

From the repository root, the `Makefile` provides common build and run targets:

- `make functions` – Build the Functions service binary in `functions/functions`.
- `make platform` – Build the Platform API server binary in `platform/platform-server`.
- `make platform-web` – Install dependencies and start the Platform Web dev server with Bun.
- `make dev` – Print recommended commands for running the full stack in separate terminals.

### Running the Full Stack Locally

Use separate terminals (or panes) for each component:

2. **Functions Service**

   ```bash
   cd functions
   go build -o ./functions ./cmd/functions
   ./functions --data-dir ./data --socket /tmp/functions.sock
   ```

3. **Platform API**

   ```bash
   cd platform
   go build -o ./platform-server ./cmd/server
   ./platform-server \
     --db-path ./data/platform.db \
     --port 3001 \
     --functions-socket /tmp/functions.sock \
     --bundle-path ../functions/data/bundles \
     --cors-origin http://localhost:5173
   ```

4. **Platform Web**
   ```bash
   cd platform-web
   bun install
   bun run dev
   ```

### Bun for JS/TS Projects

All JS/TS projects in this repo (currently `platform-web/`, and future `packages/*`) should be developed using Bun:

- Install dependencies with `bun install`.
- Run scripts with `bun run <script-name>`.

For example, in `platform-web/`:

- `bun run dev` – Start Vite dev server.
- `bun run build` – Type-check and build production assets.
- `bun run preview` – Preview the production build.

### Where to Add New Docs and Plans

- Use `requirements/` for product requirements and user flows.
- Use `planning/` for engineering designs and RFCs.
- Use `docs/` for cross-cutting architecture and workflows like this one.
