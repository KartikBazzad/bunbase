# Buncast

Buncast is the Publish-Subscribe (Pub/Sub) service for Bunbase. It provides an in-memory event bus so that the Platform API, Functions service, and dashboard can exchange events (e.g. function deployed).

## Build and run

```bash
# From repo root (with go.work)
go build -o buncast ./buncast/cmd/server

# Or from buncast/
cd buncast && go build -o buncast ./cmd/server

# Run (default socket /tmp/buncast.sock, HTTP :8081)
./buncast

# Options
./buncast -socket /tmp/buncast.sock -http :8081 -debug
./buncast -http ""   # Disable HTTP server
```

## Configuration

- **-socket**: Unix socket path for IPC (default: `/tmp/buncast.sock`).
- **-http**: HTTP listen address for health, topics, and SSE subscribe (default: `:8081`). Use `""` to disable.
- **-debug**: Enable debug logging.

See [docs/configuration.md](docs/configuration.md) for full configuration.

## Usage

- **IPC (Go client)**: Use `github.com/kartikbazzad/bunbase/buncast/pkg/client` to Publish, Subscribe, CreateTopic, ListTopics over the Unix socket.
- **HTTP**: `GET /health`, `GET /topics`, `GET /subscribe?topic=NAME` (SSE).

See [docs/api.md](docs/api.md) for IPC commands and HTTP endpoints.

## Documentation

- [Architecture](docs/architecture.md) – internal broker and transports
- [Configuration](docs/configuration.md) – socket path, HTTP port, limits
- [API](docs/api.md) – IPC commands and HTTP endpoints
