# BunBase CLI Guide

The BunBase CLI binary is implemented in `platform/cmd/cli`.

## Build

```bash
cd platform
go build -o bunbase ./cmd/cli
```

## Command Surface

Top-level commands currently available:
- `bunbase login`
- `bunbase projects`
- `bunbase functions`
- `bunbase dev`
- `bunbase whoami`

## Authentication

Login stores session and base URL in `~/.bunbase/cli_config.json`.

```bash
bunbase login --email you@example.com --password 'your-password'
```

Optional flags:
- `--base-url` (default: `http://localhost:3001/api`)

Check current authenticated user:

```bash
bunbase whoami
```

## Projects

List projects:

```bash
bunbase projects list
```

Create a project (also sets it active):

```bash
bunbase projects create "My Project"
```

Use project for deploy commands:

```bash
bunbase projects use <project-id>
```

## Functions

Initialize function template:

```bash
bunbase functions init my-function --template ts
```

Supported templates:
- `ts`
- `js`

Deploy function bundle/file:

```bash
bunbase functions deploy \
  --file dist/index.js \
  --name my-function \
  --runtime bun \
  --handler default \
  --version v1
```

Notes:
- `--name` defaults to source filename (without extension) when omitted.
- Active project from `bunbase projects use` is required.

## Local Dev Runner

Run local function dev workflow through `functions-dev` wrapper:

```bash
bunbase dev --entry src/index.ts --name my-function --runtime bun --port 8787
```

Useful flags:
- `--entry` (auto-detects `dist/index.js`, `src/index.ts`, `src/index.js` if omitted)
- `--name` (defaults to current directory name)
- `--runtime` (`bun` or `quickjs-ng`)
- `--handler` (default: `default`)
- `--port` (default: `8787`)
- `--runner` (default binary: `functions-dev`)

## Troubleshooting

- `not logged in; run bunbase login first`
  - Run `bunbase login --email ... --password ...`
- `no active project`
  - Run `bunbase projects list` then `bunbase projects use <project-id>`
- `deploy failed`
  - Verify API server is reachable and function file exists.
