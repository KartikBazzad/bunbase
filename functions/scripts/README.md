# Deployment Scripts

This directory contains scripts for deploying functions to the BunBase Functions service.

## QuickJS-NG Deployment

### Bash Script

Deploy a function with QuickJS-NG runtime using the bash script:

```bash
./scripts/deploy-quickjs-function.sh <function-name> <function-file> [version] [capability-profile]
```

**Examples:**

```bash
# Deploy with default settings (strict security profile)
./scripts/deploy-quickjs-function.sh hello-world examples/hello-world.ts

# Deploy with custom version
./scripts/deploy-quickjs-function.sh my-function ./my-function.ts v2

# Deploy with permissive security profile
./scripts/deploy-quickjs-function.sh api-handler ./api.ts v1 permissive
```

**Capability Profiles:**

- `strict` (default): No filesystem, no network, no child processes, no eval
- `permissive`: All capabilities enabled (for trusted code)

**What it does:**

1. Creates bundle directory structure
2. Builds function bundle using bun or esbuild
3. Registers function in metadata database with QuickJS-NG runtime
4. Creates function version
5. Deploys the function

**Requirements:**

- `bun` or `esbuild` for bundling
- Functions service database (created automatically)
- QuickJS worker binary built (`cmd/quickjs-worker/quickjs-worker`)

### Go Deployment Tool

A Go-based deployment tool is also available (work in progress):

```bash
go run scripts/deploy-quickjs-function.go \
  -name hello-world \
  -file examples/hello-world.ts \
  -version v1 \
  -profile strict \
  -data-dir ./data
```

## Testing Deployed Functions

After deployment, test your function:

```bash
# Via HTTP (if gateway is enabled)
curl 'http://localhost:8080/functions/hello-world?name=Alice'

# Via IPC client (see pkg/client)
```

## Function Requirements

Functions must export a default handler:

```typescript
export default async function handler(req: Request): Promise<Response> {
  return Response.json({ message: "Hello from QuickJS!" });
}
```

## Security

QuickJS-NG functions run with capability-based security:

- **Strict Profile**: Isolated execution, no external access
- **Permissive Profile**: Full access (use with caution)

Capabilities can be customized per function in the database.
