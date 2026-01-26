# BunBase CLI

Command-line interface for managing BunBase projects and deploying functions.

## Installation

```bash
npm install -g @bunbase/cli
# or
bun add -g @bunbase/cli
```

## Authentication

### Login

```bash
# Interactive login (email/password)
bunbase login

# Non-interactive login
bunbase login --email user@example.com --password mypassword

# API key login (backward compatible)
bunbase login --api-key <your-api-key> --base-url https://api.bunbase.com --project-id <project-id>
```

### Logout

```bash
bunbase logout
```

## Functions Commands

### Initialize Function

```bash
# Initialize HTTP function
bunbase functions init my-api --runtime nodejs20 --type http --path /api/users

# Initialize callable function
bunbase functions init send-email --runtime nodejs20 --type callable

# Initialize Python function
bunbase functions init process-data --runtime python3.11 --type http
```

This creates a function template in `functions/<name>/` with:
- Handler code (`index.ts`, `index.py`, etc.)
- `package.json` or `requirements.txt` (depending on runtime)
- `README.md` with documentation
- Updates `functions.json` or `bunbase.config.ts` with function configuration

### List Functions

```bash
bunbase functions list
bunbase functions list --json
```

### Deploy Functions

```bash
# Deploy all functions from config
bunbase functions deploy

# Deploy specific function
bunbase functions deploy my-function

# Deploy from directory
bunbase functions deploy --dir ./functions
```

### Create Function

```bash
bunbase functions create my-function \
  --runtime nodejs20 \
  --handler index.handler \
  --type http \
  --path /api/users \
  --methods GET,POST \
  --code ./functions/my-function/index.ts
```

### Invoke Function

```bash
# Invoke with inline data
bunbase functions invoke my-function --data '{"key":"value"}'

# Invoke with file
bunbase functions invoke my-function --data-file input.json
```

### View Logs

```bash
# View logs
bunbase functions logs my-function

# Stream logs
bunbase functions logs my-function --tail

# With pagination
bunbase functions logs my-function --limit 50 --offset 0
```

### Delete Function

```bash
bunbase functions delete my-function --force
```

## Configuration

### bunbase.config.ts

Create a `bunbase.config.ts` file in your project root:

```typescript
export default {
  functions: {
    "my-api": {
      runtime: "nodejs20",
      handler: "index.handler",
      type: "http",
      path: "/api/users",
      methods: ["GET", "POST"],
      memory: 512,
      timeout: 30,
    },
    "send-email": {
      runtime: "nodejs20",
      handler: "index.handler",
      type: "callable",
      memory: 256,
      timeout: 10,
    },
  },
};
```

### functions.json

Alternatively, use a JSON configuration file:

```json
{
  "functions": {
    "my-api": {
      "runtime": "nodejs20",
      "handler": "index.handler",
      "type": "http",
      "path": "/api/users",
      "methods": ["GET", "POST"],
      "memory": 512,
      "timeout": 30
    }
  }
}
```

## License

MIT
