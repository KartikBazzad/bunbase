# QuickJS-NG Worker

This is the C wrapper that embeds QuickJS-NG and libuv to execute JavaScript functions in a sandboxed environment.

## Building

### Prerequisites

1. **QuickJS-NG**: Clone and build QuickJS-NG
   ```bash
   cd ..
   git clone https://github.com/quickjs-ng/quickjs.git quickjs-ng
   cd quickjs-ng
   make
   ```

2. **libuv**: Install libuv development package
   ```bash
   # Ubuntu/Debian
   sudo apt-get install libuv1-dev
   
   # macOS
   brew install libuv
   ```

### Build

```bash
cd cmd/quickjs-worker
make
```

This will create the `quickjs-worker` binary.

## Usage

The worker is spawned by the Go control plane with the following environment variables:

- `WORKER_ID`: Unique worker identifier
- `BUNDLE_PATH`: Path to JavaScript bundle file
- `CAPABILITIES`: JSON string with security capabilities
- `MAX_MEMORY`: Maximum memory in bytes
- `MAX_FDS`: Maximum file descriptors
- `ALLOW_FILESYSTEM`: Enable filesystem access
- `ALLOW_NETWORK`: Enable network access
- `ALLOW_CHILD_PROCESS`: Enable child process spawning
- `ALLOW_EVAL`: Enable eval() and Function() constructor

## Protocol

The worker communicates via NDJSON (newline-delimited JSON) over stdin/stdout:

1. Worker sends `{"id":"<worker_id>","type":"ready","payload":{}}` when ready
2. Control plane sends `{"id":"<invoke_id>","type":"invoke","payload":{...}}` to invoke function
3. Worker sends `{"id":"<invoke_id>","type":"response","payload":{...}}` or `{"id":"<invoke_id>","type":"error","payload":{...}}`

## Security

The worker enforces:
- Resource limits (memory, file descriptors)
- Capability-based access control
- No eval/Function by default
- Process-level isolation
