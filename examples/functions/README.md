# Function Examples

This directory contains example functions that demonstrate various use cases for BunBase serverless functions.

## Available Examples

### 1. hello-world.ts
Simple greeting function that accepts a name parameter.

**Usage:**
```bash
GET /functions/hello-world?name=Alice
```

**Response:**
```json
{
  "message": "Hello, Alice!",
  "timestamp": "2026-01-26T...",
  "method": "GET",
  "path": "/"
}
```

### 2. echo.ts
Echoes back request data including method, URL, query params, headers, and body.

**Usage:**
```bash
POST /functions/echo
Content-Type: application/json

{
  "test": "data"
}
```

### 3. json-processor.ts
Processes JSON data and returns metadata about it.

**Usage:**
```bash
POST /functions/json-processor
Content-Type: application/json

[1, 2, 3, 4, 5]
```

### 4. calculator.ts
Simple calculator API supporting basic operations.

**Usage:**
```bash
GET /functions/calculator?a=10&b=5&op=multiply
# or
POST /functions/calculator
{
  "a": 10,
  "b": 5,
  "op": "divide"
}
```

**Supported operations:** add, subtract, multiply, divide, power

### 5. env-demo.ts
Demonstrates accessing environment variables.

**Usage:**
```bash
GET /functions/env-demo
```

Set environment variables first:
```bash
POST /api/functions/{functionId}/env
{
  "key": "CUSTOM_VAR",
  "value": "my-secret-value"
}
```

## Creating and Deploying Functions

### Using the API

1. **Create a function:**
```bash
curl -X POST http://localhost:3000/api/functions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-world",
    "runtime": "bun",
    "handler": "handler",
    "code": "export async function handler(req: Request): Promise<Response> { return Response.json({ message: \"Hello!\" }); }"
  }'
```

2. **Deploy the function:**
```bash
curl -X POST http://localhost:3000/api/functions/{functionId}/deploy \
  -H "Authorization: Bearer YOUR_API_KEY"
```

3. **Invoke the function:**
```bash
curl http://localhost:3000/api/functions/hello-world \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Using the CLI (if available)

```bash
# Create function
bunbase functions create hello-world \
  --runtime bun \
  --handler handler \
  --code ./examples/functions/hello-world.ts

# Deploy
bunbase functions deploy hello-world

# Invoke
bunbase functions invoke hello-world
```

## Testing Functions

You can test functions using curl, the API, or by making HTTP requests directly to the function endpoints once deployed.
