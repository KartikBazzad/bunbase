# BunBase CLI Requirements

The **BunBase CLI** (`bunbase`) is the primary developer tool for interacting with the platform. It allows developers to develop, test, and deploy their applications.

## Technical Stack
-   **Language**: Go
-   **Libraries**: `spf13/cobra` (Commands), `charmbracelet/bubbletea` (TUI/Spinners).
-   **Distribution**: Homebrew, `go install`, Binary release.

## Core Commands

### 1. Authentication
Interacts with `bun-auth` and `platform`.
-   `bunbase login`: Opens browser to authenticate. Saves JWT to `~/.bunbase/credentials.json`.
-   `bunbase logout`: Clears local credentials.
-   `bunbase whoami`: Shows current logged-in user and project context.

### 2. Project Management
-   `bunbase init`: Creates `bunbase.toml` in current directory. Interactive wizard (Project Name, Region).
-   `bunbase link`: Links current directory to an existing remote project.

### 3. Functions
-   `bunbase deploy`: uploads the `functions/` directory to the Platform.
    -   Zips directory.
    -   POSTs to `/api/v1/projects/{id}/deploy`.
    -   Streams build/deployment logs.
-   `bunbase logs`: Tails logs for the current project's functions (connects to `buncast` or Platform API).
    -   `bunbase logs -f`: Follow mode.

### 4. Configuration & Secrets
-   `bunbase env set KEY=VALUE`: Sets encrypted environment variables for functions (stored in BunKMS/Platform DB).
-   `bunbase env get`: Lists keys (values masked).
-   `bunbase env pull`: Downloads secrets to `.env.local` for local development.

### 5. Local Development (Future)
-   `bunbase dev`: Starts a local emulation of the BunBase stack (running local Bun worker, local MinIO/Postgres via Docker).

## Configuration File (`bunbase.toml`)
```toml
project_id = "proj_12345"
region = "us-east-1"

[functions]
source = "./functions"
# runtime = "bun" (default)
```

## Implementation Strategy
1.  **Repo**: `github.com/kartikbazzad/bunbase/cli` (Monorepo `cmd/bunbase` or separate package).
2.  **Versioning**: SemVer, aligned with Platform API versions.
