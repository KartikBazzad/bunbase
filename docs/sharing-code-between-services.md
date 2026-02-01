## Sharing Code Between Services

This guide explains how to share code between the Go services (`functions`, `platform`) in the BunBase monorepo.

### Current State

Each service is a **separate Go module**:

- `github.com/kartikbazzad/bunbase/functions`
- `github.com/kartikbazzad/bunbase/platform`

**Problem**: Code duplication exists. For example:

- `platform/pkg/functions/client.go` duplicates `functions/pkg/client/client.go`
- Both implement the same IPC protocol client

### Solution: Go Workspaces + Shared Packages

We'll use **Go workspaces** (introduced in Go 1.18) to enable local imports between modules, plus a `packages/` directory for truly shared code.

#### Step 1: Create `go.work` File

Create a `go.work` file at the repository root:

```go
go 1.21

use (
	./functions
	./platform
)
```

This tells Go to treat all three modules as part of a single workspace, allowing them to import each other using their module paths.

#### Step 2: Use Cross-Module Imports

Once `go.work` exists, you can import code from other modules:

**In `platform/internal/services/function.go`:**

```go
import (
	"github.com/kartikbazzad/bunbase/functions/pkg/client"
)

// Use the shared client instead of duplicating
func NewFunctionService(...) {
	client := client.New(functionsSocketPath)
	// ...
}
```

**Benefits:**

- No code duplication
- Single source of truth
- Changes propagate automatically

#### Step 3: Create Shared Packages (When Needed)

For code that's truly shared across multiple services (not just one importing another), create a shared package:

**Structure:**

```
packages/
└── shared-go/
    ├── go.mod          # module github.com/kartikbazzad/bunbase/shared-go
    ├── pkg/
    │   ├── logger/     # Shared logging utilities
    │   ├── errors/     # Shared error types
    │   └── config/     # Shared config parsing
    └── internal/
        └── utils/      # Internal shared utilities
```

**Add to `go.work`:**

```go
use (
	.
	./functions
	./platform
	./packages/shared-go
)
```

**Import in services:**

```go
import "github.com/kartikbazzad/bunbase/shared-go/pkg/logger"
```

### Migration Example: Functions Client

Here's how to eliminate the duplication of the Functions client:

1. **Keep the canonical version** in `functions/pkg/client/`
2. **Remove** `platform/pkg/functions/client.go`
3. **Update** `platform/go.mod` to import from functions:
   ```go
   require github.com/kartikbazzad/bunbase/functions v0.0.0
   ```
4. **Add replace directive** (for local development):
   ```go
   replace github.com/kartikbazzad/bunbase/functions => ../functions
   ```
5. **Update imports** in `platform/internal/services/function.go`:
   ```go
   import "github.com/kartikbazzad/bunbase/functions/pkg/client"
   ```

### Rules of Thumb

**Import from another service when:**

- One service needs to call another (e.g., Platform → Functions client)
- The code belongs to a specific service's domain

**Create a shared package when:**

- Code is used by 3+ services
- Code has no clear "owner" service
- Code is pure utilities (logging, errors, config)

**Keep code local when:**

- Only used within one service
- Still evolving rapidly
- Service-specific implementation details

### TypeScript/JavaScript Sharing

For JS/TS code (like `platform-web`), use a `packages/` directory with Bun workspaces:

**Structure:**

```
packages/
└── platform-types/     # Shared API types
    ├── package.json
    └── src/
```

**In `platform-web/package.json`:**

```json
{
  "dependencies": {
    "@bunbase/platform-types": "workspace:*"
  }
}
```

**Root `package.json` (or `bunfig.toml`):**

```json
{
  "workspaces": ["packages/*", "platform-web"]
}
```

### Quick Start

1. Create `go.work` at repo root with all Go modules
2. Run `go work sync` to ensure modules are in sync
3. Start using cross-module imports
4. Create `packages/` directory when you need truly shared code

### See Also

- `planning/shared-libraries.md` - Long-term strategy
- `planning/monorepo-structure.md` - Overall monorepo conventions
