# Functions Runtime Isolation

## Goal
Restrict access to sensitive system modules (Filesystem, Network, Shell) in the **Bun** runtime environment, aligning it with the isolation level of QuickJS.

## Problem
The current `bun_worker.go` spawns a standard Bun process. User code running in this process has full access to:
-   `fs` (Read/Write any file user permissions allow)
-   `child_process` (Spawn arbitrary commands)
-   `Bun.spawn`, `Bun.write`, etc.
-   `bun:ffi` (Native code execution)

## Proposed Solution: Userland Isolation (Preload Script)
Since we cannot easily recompile Bun to strip these features, we will use a **Preload Script** (`preload.ts`) that runs *before* user code to patch the global scope and module system.

### 1. The Preload Script (`worker/preload.ts`)
This script will:
1.  **Patch Globals**: Overwrite `Bun.file`, `Bun.write`, `Bun.spawn` with functions that throw `SecurityError`.
2.  **Mock Modules**: Use Bun's plugin system or module mocking to redirect imports of `fs`, `node:fs`, `child_process`, `bun:ffi` to a "poisoned" module that throws errors on access.

### 2. Implementation Logic

```typescript
// worker/preload.ts

const BLOCKED_MODULES = [
  "fs", "node:fs", "node:fs/promises",
  "child_process", "node:child_process",
  "bun:ffi",
  "bun:sqlite",
];

// 1. Block Bun Globals
globalThis.Bun.file = () => { throw new Error("Security: File access denied"); };
globalThis.Bun.write = () => { throw new Error("Security: File write denied"); };
globalThis.Bun.spawn = () => { throw new Error("Security: Process spawning denied"); };
globalThis.Bun.spawnSync = () => { throw new Error("Security: Process spawning denied"); };

// 2. Block Node Builtins (via Plugin or runtime mocking)
// Note: Bun doesn't strictly support "blocking" builtins easily without a loader.
// Strategy: We might need to wrap the user code execution in a sandbox-like scope
// or rely on `bun build` to tree-shake/mock them if we were building, but we are running.

// ALTERNATIVE STRATEGY for Modules:
// We can't easily intercept "import fs from 'fs'" in pure runtime without a loader hook.
// However, we CAN poison the global `require` if it exists, and `process.binding`.
```

### Refined Strategy: Runtime Wrapper
Since purely blocking `import` in a running process is hard, we will:
1.  **Disable Auto-loading**: Run Bun with `--no-install`.
2.  **Loader Hook**: Use `Bun.plugin` to intercept imports.

```typescript
import { plugin } from "bun";

plugin({
  name: "security-policy",
  setup(build) {
    build.onResolve({ filter: /^(fs|node:fs|child_process|bun:ffi)/ }, args => {
      return { path: args.path, namespace: "blocked" };
    });
    
    build.onLoad({ filter: /.*/, namespace: "blocked" }, args => {
      return {
        contents: `throw new Error("Security: Module '${args.path}' is blocked.");`,
        loader: "js"
      };
    });
  }
});
```
*Note: Bun plugins currently affect **bundling**, but `bun run` also supports some runtime hooks. We need to verify if `Bun.plugin` works for runtime imports in `bun run`.*

## Risky Modules List
| Module | Risk | Action |
| `fs`, `node:fs` | File System Access | **BLOCK** |
| `child_process` | Shell Execution | **BLOCK** |
| `bun:ffi` | Native Code | **BLOCK** |
| `worker_threads` | Resource Exhaustion | **limit** (or block) |
| `dgram`, `net` | Raw Sockets | **Block** (Allow `fetch` only) |
| `os` | System Info | **Allow** (Low risk, maybe mock) |

## Changes Required

### 1. `functions/worker/preload.ts`
Create this file to implement the global patching and plugin injection.

### 2. `functions/internal/worker/bun_worker.go`
Update `Spawn` method to inject the preload script.
```go
// cmd.Args likely needs to include "-r" (require/preload)
cmd := exec.CommandContext(ctx, bunPath, "run", "--preload", preloadPath, scriptPath)
```

### 3. Verification
Create a test function trying to:
```javascript
import fs from 'fs';
export default function() {
  fs.readFileSync('/etc/passwd'); // Should throw
}
```
