# Traefik configuration for BunBase

Static entrypoint and dynamic routing config for the BunBase gateway. Traefik is the single entry point; each Bunbase service runs as its own process.

## Backend ports (default)

| Service        | Port | Path prefix  |
| -------------- | ---- | ------------ |
| Platform       | 3001 | /api         |
| Bunder Manager | 8085 | /kv          |
| Bundoc-server  | 8080 | /v1/projects |
| Buncast        | 8081 | /events      |
| Functions      | 8082 | /invoke      |

If Bundoc and Functions both use 8080 by default, run one on a different port (e.g. Functions on 8082) and set the URL in `dynamic.yml` accordingly.

## Running Traefik

### With dynamic config file

```bash
# From repo root
traefik --entrypoints.web.address=:8080 --providers.file.filename=./deploy/traefik/dynamic.yml --api.dashboard=true
```

Gateway listens on `:8080`. Clients use `http://localhost:8080` as the gateway URL (set Platform `--gateway-url http://localhost:8080`).

### With config directory

```bash
traefik --entrypoints.web.address=:8080 --providers.file.directory=./deploy/traefik/ --providers.file.watch=true
```

## Files

- **dynamic.yml** – HTTP routers and services (path → backend). Edit server URLs for your environment (e.g. Docker service names instead of 127.0.0.1).
- **README.md** – This file.

## Optional: middlewares

Add auth or rate limiting by defining `http.middlewares` in `dynamic.yml` and attaching them to routers via `routers.*.middlewares`.
