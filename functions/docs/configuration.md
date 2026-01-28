# Configuration Guide

This document describes all configuration options for BunBase Functions.

## Table of Contents

1. [Configuration File](#configuration-file)
2. [Command-Line Flags](#command-line-flags)
3. [Environment Variables](#environment-variables)
4. [Function-Level Configuration](#function-level-configuration)
5. [Recommended Settings](#recommended-settings)

---

## Configuration File

Configuration can be provided via a JSON or YAML file (future) or command-line flags.

### Default Configuration

```go
type Config struct {
    DataDir    string
    SocketPath string
    Worker     WorkerConfig
    Gateway    GatewayConfig
    Metadata   MetadataConfig
    Logs       LogsConfig
}

type WorkerConfig struct {
    MaxWorkersPerFunction  int
    WarmWorkersPerFunction int
    IdleTimeout           time.Duration
    StartupTimeout        time.Duration
    ExecutionTimeout      time.Duration
    MemoryLimitMB         int
    BunPath               string
}

type GatewayConfig struct {
    HTTPPort   int
    EnableHTTP bool
}

type MetadataConfig struct {
    DBPath string
}

type LogsConfig struct {
    DBPath    string
    JSONLPath string
    Retention time.Duration
}
```

### Example Configuration File

```json
{
  "data_dir": "./data",
  "socket_path": "/tmp/functions.sock",
  "worker": {
    "max_workers_per_function": 10,
    "warm_workers_per_function": 2,
    "idle_timeout_seconds": 300,
    "startup_timeout_seconds": 10,
    "execution_timeout_seconds": 30,
    "memory_limit_mb": 256,
    "bun_path": "bun"
  },
  "gateway": {
    "http_port": 8080,
    "enable_http": true
  },
  "metadata": {
    "db_path": "./data/metadata.db"
  },
  "logs": {
    "db_path": "./data/logs.db",
    "jsonl_path": "./data/logs",
    "retention_days": 30
  }
}
```

---

## Command-Line Flags

### Server Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--data-dir` | string | `./data` | Directory for data files |
| `--socket` | string | `/tmp/functions.sock` | Unix socket path |
| `--config` | string | `` | Path to config file (optional) |
| `--http-port` | int | `8080` | HTTP gateway port |
| `--enable-http` | bool | `true` | Enable HTTP gateway |
| `--log-level` | string | `info` | Log level (debug, info, warn, error) |

### Worker Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max-workers` | int | `10` | Max workers per function |
| `--warm-workers` | int | `2` | Warm workers per function |
| `--idle-timeout` | duration | `5m` | Idle worker timeout |
| `--startup-timeout` | duration | `10s` | Worker startup timeout |
| `--execution-timeout` | duration | `30s` | Execution timeout |
| `--memory-limit` | int | `256` | Memory limit per worker (MB) |
| `--bun-path` | string | `bun` | Path to Bun executable |

### Storage Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--metadata-db` | string | `./data/metadata.db` | Metadata database path |
| `--logs-db` | string | `./data/logs.db` | Logs database path |
| `--logs-jsonl` | string | `./data/logs` | JSONL logs directory |
| `--logs-retention` | duration | `720h` | Log retention period |

---

## Environment Variables

All configuration can be overridden via environment variables:

| Environment Variable | Config Field | Example |
|---------------------|--------------|---------|
| `FUNCTIONS_DATA_DIR` | `data_dir` | `./data` |
| `FUNCTIONS_SOCKET_PATH` | `socket_path` | `/tmp/functions.sock` |
| `FUNCTIONS_HTTP_PORT` | `gateway.http_port` | `8080` |
| `FUNCTIONS_MAX_WORKERS` | `worker.max_workers_per_function` | `10` |
| `FUNCTIONS_WARM_WORKERS` | `worker.warm_workers_per_function` | `2` |
| `FUNCTIONS_IDLE_TIMEOUT` | `worker.idle_timeout` | `5m` |
| `FUNCTIONS_MEMORY_LIMIT` | `worker.memory_limit_mb` | `256` |
| `FUNCTIONS_BUN_PATH` | `worker.bun_path` | `bun` |
| `FUNCTIONS_LOG_LEVEL` | `log_level` | `info` |

**Precedence:** Command-line flags > Environment variables > Config file > Defaults

---

## Function-Level Configuration

Functions can have per-function configuration:

```json
{
  "function_id": "func-123",
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler",
  "memory_mb": 512,
  "timeout_seconds": 60,
  "max_concurrent_executions": 20,
  "warm_workers": 3
}
```

### Function Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `memory_mb` | int | `256` | Memory limit per worker (MB) |
| `timeout_seconds` | int | `30` | Execution timeout (seconds) |
| `max_concurrent_executions` | int | `10` | Max concurrent invocations |
| `warm_workers` | int | `2` | Number of warm workers to maintain |

**Note:** Function-level config overrides global config for that function.

---

## Recommended Settings

### Development

```json
{
  "worker": {
    "max_workers_per_function": 5,
    "warm_workers_per_function": 1,
    "idle_timeout_seconds": 60,
    "execution_timeout_seconds": 30,
    "memory_limit_mb": 128
  },
  "logs": {
    "retention_days": 7
  }
}
```

**Rationale:**
- Lower limits for resource efficiency
- Shorter retention for disk space
- Faster idle timeout for cleanup

### Production

```json
{
  "worker": {
    "max_workers_per_function": 20,
    "warm_workers_per_function": 3,
    "idle_timeout_seconds": 600,
    "execution_timeout_seconds": 30,
    "memory_limit_mb": 512
  },
  "logs": {
    "retention_days": 30
  }
}
```

**Rationale:**
- Higher limits for throughput
- More warm workers for fast execution
- Longer idle timeout to reduce cold starts
- Longer retention for debugging

### High-Throughput

```json
{
  "worker": {
    "max_workers_per_function": 50,
    "warm_workers_per_function": 10,
    "idle_timeout_seconds": 1800,
    "execution_timeout_seconds": 30,
    "memory_limit_mb": 1024
  }
}
```

**Rationale:**
- Very high limits for high traffic
- Many warm workers for instant execution
- Very long idle timeout to minimize cold starts

---

## Configuration Validation

The service validates configuration on startup:

### Validation Rules

1. **Data Directory**
   - Must exist or be creatable
   - Must be writable

2. **Socket Path**
   - Must be valid Unix socket path
   - Parent directory must exist

3. **Worker Limits**
   - `max_workers_per_function` > 0
   - `warm_workers_per_function` <= `max_workers_per_function`
   - `idle_timeout` > 0
   - `startup_timeout` > 0
   - `execution_timeout` > 0
   - `memory_limit_mb` > 0

4. **Gateway**
   - `http_port` must be valid port (1-65535)
   - `enable_http` must be bool

5. **Storage**
   - `metadata_db` path must be writable
   - `logs_db` path must be writable
   - `logs_jsonl` directory must exist or be creatable

6. **Bun Path**
   - `bun_path` must be executable
   - Must be in PATH or absolute path

**On Validation Failure:**
- Log error with details
- Exit with non-zero code
- Do not start server

---

## Runtime Configuration Changes

**Current (v1):** Configuration is read-only at runtime.

**Future:** Support hot-reload for certain settings:
- Worker limits (with graceful scaling)
- Log retention (affects cleanup only)
- Timeouts (affects new invocations only)

**Not Hot-Reloadable:**
- Data directory
- Socket path
- Bun path (requires restart)

---

## Configuration Examples

### Minimal Configuration

```bash
./functions --data-dir ./data --socket /tmp/functions.sock
```

Uses all defaults.

### Custom Worker Settings

```bash
./functions \
  --data-dir ./data \
  --socket /tmp/functions.sock \
  --max-workers 20 \
  --warm-workers 5 \
  --idle-timeout 10m \
  --memory-limit 512
```

### Production Configuration

```bash
./functions \
  --data-dir /var/lib/functions \
  --socket /var/run/functions.sock \
  --http-port 8080 \
  --max-workers 50 \
  --warm-workers 10 \
  --idle-timeout 30m \
  --memory-limit 1024 \
  --logs-retention 720h \
  --log-level info
```

### Development Configuration

```bash
./functions \
  --data-dir ./data \
  --socket /tmp/functions.sock \
  --http-port 3000 \
  --max-workers 5 \
  --warm-workers 1 \
  --idle-timeout 1m \
  --memory-limit 128 \
  --logs-retention 168h \
  --log-level debug
```

---

## Troubleshooting

### Configuration Not Applied

**Symptoms:** Changes don't take effect.

**Solutions:**
- Check configuration file syntax
- Verify environment variables are set
- Check command-line flag precedence
- Restart server (config is read-only at runtime)

### Invalid Configuration

**Symptoms:** Server fails to start.

**Solutions:**
- Check validation errors in logs
- Verify paths exist and are writable
- Check numeric values are positive
- Verify Bun path is executable

### Performance Issues

**Symptoms:** Slow execution, high memory usage.

**Solutions:**
- Increase `max_workers_per_function` for throughput
- Increase `warm_workers_per_function` for fast execution
- Increase `idle_timeout` to reduce cold starts
- Adjust `memory_limit_mb` based on function needs
