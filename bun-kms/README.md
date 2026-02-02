# BunKMS (Key Management Service)

BunKMS is the centralized "Root of Trust" for the BunBase ecosystem. It provides secure key management, encryption-as-a-service, secret storage, and signing/verification.

## Features

- **Key Management**: Create, rotate, and revoke cryptographic keys (AES-256, RSA-2048, ECDSA-P256).
- **Crypto Operations**: Encrypt/decrypt data without exposing keys (envelope encryption).
- **Signing**: Sign and verify with RSA and ECDSA keys (SHA-256 digest).
- **Secrets**: Store and retrieve secrets encrypted with the master key.
- **Persistence**: Optional Bunder-backed storage with AES-GCM encryption at rest.
- **Audit Logging**: Append-only log of key and secret operations.
- **Auth**: Optional JWT authentication and role-based access.
- **Health & Metrics**: `/health`, `/ready`, Prometheus `/metrics`.

## Getting Started

### Run server (in-memory)

```bash
export BUNKMS_MASTER_KEY=base64:$(echo -n "0123456789abcdef0123456789abcdef" | base64)
go run ./cmd/server
```

Server listens on `:8080`. Create a key and encrypt:

```bash
curl -s -X POST http://localhost:8080/v1/keys -H "Content-Type: application/json" -d '{"name":"mykey","type":"aes-256"}'
curl -s -X POST http://localhost:8080/v1/encrypt/mykey -H "Content-Type: application/json" -d '{"plaintext":"hello"}'
```

### Run with persistence (Bunder)

```bash
export BUNKMS_MASTER_KEY=base64:$(echo -n "0123456789abcdef0123456789abcdef" | base64)
export BUNKMS_DATA_PATH=./data
export BUNKMS_AUDIT_LOG=./data/audit.log
go run ./cmd/server
```

### CLI

```bash
go build -o bunkms-cli ./cmd/cli
./bunkms-cli key create mykey aes-256
./bunkms-cli encrypt mykey "secret message"
./bunkms-cli secret put db-password "s3cr3t"
./bunkms-cli secret get db-password
```

Use `-url` and `-token` (or `BUNKMS_TOKEN`) when the server requires JWT.

### Load / stress tests

Start the server, then run the load test CLI:

```bash
# In one terminal: start server
export BUNKMS_MASTER_KEY=base64:$(echo -n "0123456789abcdef0123456789abcdef" | base64)
go run ./cmd/server

# In another: run load test (10s, 20 clients, mixed encrypt/decrypt)
go run ./cmd/loadtest -url http://localhost:8080 -duration 10s -clients 20

# Encrypt-only, 5s, 10 clients
go run ./cmd/loadtest -url http://localhost:8080 -workload encrypt -duration 5s -clients 10

# Secrets workload
go run ./cmd/loadtest -url http://localhost:8080 -workload secrets -duration 5s
```

Workloads: `encrypt`, `decrypt`, `mixed` (encrypt+decrypt), `secrets` (put+get), `keys` (create+get). Results show total ops, errors, ops/sec, and latency percentiles (P50, P95, P99).

In-process tests (no server required): `go test ./internal/loadtest/... -v`

## Documentation

- [Architecture](docs/architecture.md)
- [API Reference](docs/api.md)
- [Deployment](docs/deployment.md)
- [Security](docs/security.md)

## Build and Test

```bash
make build
make test
```

## Configuration

See [Deployment](docs/deployment.md) for environment variables. Configuration is also available via `internal/config` (env-based).

## Architecture

See [planning/bun_kms.md](../planning/bun_kms.md) for the implementation plan and [docs/architecture.md](docs/architecture.md) for the current design.
