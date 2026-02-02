# Bunder Manager

Instance manager for Bunder KV: one Bunder process per project. Used behind Traefik so that each project gets isolated KV storage.

## Overview

- Accepts HTTP requests at `/kv/{project_id}/...` and proxies to the Bunder instance for that project.
- Spawns a new Bunder process on first request for a project (lazy provisioning).
- Evicts idle instances after a configurable TTL (hot/cold pool).
- Data is stored under `./data/projects/{project_id}/`.

## Quick Start

```bash
# Build bunder binary first (used as child process)
cd ../bunder && go build -o bunder ./cmd/server && cd ../bunder-manager

# Run manager (default: listen :8085, spawn bunder from PATH)
go run ./cmd/server

# With options
go run ./cmd/server -addr :8085 -data ./data -bunder-bin ../bunder/bunder -port-base 9000 -port-count 100
```

## Configuration

- **-addr**: HTTP listen address (default `:8085`)
- **-data**: Root data directory; project data under `data/projects/{project_id}/`
- **-bunder-bin**: Path to bunder binary (default `bunder` from PATH)
- **-port-base**: First port in pool for Bunder HTTP (default 9000)
- **-port-count**: Number of ports in pool (default 1000)

## API

- `GET /health` – health check
- `GET/PUT/DELETE /kv/{project_id}/kv/{key}` – proxied to project's Bunder
- `GET /kv/{project_id}/keys` – proxied to project's Bunder
- `GET /kv/{project_id}/health` – proxied to project's Bunder

Traefik routes `https://api.example.com/kv/{project_id}/...` to this manager.
