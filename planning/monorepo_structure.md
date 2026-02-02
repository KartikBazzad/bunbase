# Monorepo & Shared Libraries Strategy

To maintain code quality and DRY principles across our multiple Go services (`bun-auth`, `platform`, `functions`, `bun-kms`, `bundoc`), we will centralize common code in a root `pkg/` directory.

## Directory Structure

```
bunbase/
├── cmd/                 # CLI entrypoints
│   └── bunbase/         # Main Developer CLI
├── pkg/                 # SHARED LIBRARIES
│   ├── logger/          # Structured logging (Slog wrapper)
│   ├── config/          # Configuration loading (Koanf/Viper)
│   ├── errors/          # Standardized error types (gRPC/HTTP mapping)
│   ├── middle/          # Common HTTP middlewares (Auth, Tracing)
│   ├── models/          # Shared Data Models (if any)
│   └── proto/           # RPC/Protobuf definitions (if using gRPC)
├── bun-auth/            # Auth Service
├── platform/            # Platform API
├── functions/           # Functions Service
├── bun-kms/             # KMS Service
├── bundoc-server/       # Data Service
├── docker-compose.yml   # Orchestration
├── go.work              # Go Workspace (manages all modules)
└── README.md            # Entry point
```

## Shared Modules (`pkg/`)

### 1. `pkg/logger`
-   Standardized JSON logging.
-   Trace ID injection.
-   Log levels matching environment (Debug/Info/Error).

### 2. `pkg/config`
-   Standardized env var loading behavior.
-   Support for `.env` files.

### 3. `pkg/errors`
-   `AppError` struct with Code, Message, HttpStatusCode.
-   Helper functions: `NewNotFound`, `NewBadRequest`.

### 4. `pkg/rpc`
-   Client consumers for internal service-to-service communication.
-   e.g. `bunauth.Client` to call `bun-auth` service.

## Go Workspace (`go.work`)
We will use Go Workspaces to allow all services to import `github.com/kartikbazzad/bunbase/pkg` relative to the root, without `replace` directives in `go.mod` files during development.

```go
go 1.22

use (
    ./bun-auth
    ./platform
    ./functions
    ./pkg
    ./cmd/bunbase
)
```
