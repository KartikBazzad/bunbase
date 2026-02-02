// BunBase Functions Sandbox Preload
// This script runs before the user code or worker script.

const BLOCKED_MESSAGE = "Blocked by BunBase Sandbox";

// 1. Block File System Access via Bun API
// We wrap the APIs to allow necessary internal operations (like logging) but block user access.

const originalFile = Bun.file;
Object.defineProperty(Bun, "file", {
  value: function(path, options) {
    // Allow essential system devices
    if (path === "/dev/tty" || path === "/dev/stdout" || 
        path === "/dev/stderr" || path === "/dev/null") {
        return originalFile.call(Bun, path, options);
    }
    
    // Allow numeric file descriptors (often used for stdout/stderr)
    if (typeof path === "number") {
         return originalFile.call(Bun, path, options);
    }

    // Debugging what is being accessed
    throw new Error(`Blocked by BunBase Sandbox (file): ${String(path)}`);
  },
  configurable: false,
  writable: false,
});

const originalWrite = Bun.write;
Object.defineProperty(Bun, "write", {
  value: function(destination, input) {
    // Allow writing to stdout (1) and stderr (2)
    if (destination === 1 || destination === 2 || 
        destination === process.stdout || destination === process.stderr) {
      return originalWrite.call(Bun, destination, input);
    }
    throw new Error(`Blocked by BunBase Sandbox (write)`);
  },
  configurable: false,
  writable: false,
});

const originalMmap = Bun.mmap;
Object.defineProperty(Bun, "mmap", {
    value: function(...args) { throw new Error("Blocked by BunBase Sandbox (mmap)"); },
    configurable: false,
    writable: false,
});

// 2. Block Subprocesses
// We can simply block them as there is legitimate reason for a function to spawn processes
const originalSpawn = Bun.spawn;
Object.defineProperty(Bun, "spawn", {
  value: function(...args) { throw new Error("Blocked by BunBase Sandbox (spawn)"); },
  configurable: false,
  writable: false,
});

const originalSpawnSync = Bun.spawnSync;
Object.defineProperty(Bun, "spawnSync", {
    value: function(...args) { throw new Error("Blocked by BunBase Sandbox (spawnSync)"); },
    configurable: false,
    writable: false,
});

// 3. Network Restrictions (Fetch)
// We can intercept fetch to strictly allow listing.
// For now, we'll allow all but log it (or block localhost if needed).
// Ideally, we'd inject an allowlist via env vars.

const originalFetch = globalThis.fetch;
globalThis.fetch = async function (input, init) {
  const url = new URL(input instanceof Request ? input.url : input.toString());
  
  // Example: Block metadata services or internal endpoints
  if (url.hostname === "169.254.169.254" || url.hostname === "localhost" || url.hostname === "127.0.0.1") {
      // In a real sandbox, we might block localhost. 
      // But functions might need to call local sidecars.
      // For now, let's allow but log (if we could logging without breaking stdout).
      // console.error(`[SANDBOX] Fetching internal URL: ${url}`);
  }

  return originalFetch(input, init);
};

// 4. Block 'node:fs' and 'node:child_process'?
// Bun's module resolution might still allow importing them.
// We can try to mock them in the module cache if Bun allows.
// (Advanced: requires loader hooks or careful patching).

console.error("[SANDBOX] Initialized");
