# Function Development Guide

This guide covers how to write, test, and deploy functions for BunBase Functions.

## Function Handler API

All functions must export a default async handler:

```typescript
export default async function handler(req: Request): Promise<Response> {
  // Your function logic here
  return Response.json({ message: "Hello!" });
}
```

### Handler Signature

```typescript
async function handler(req: Request): Promise<Response>
```

- **Input**: Standard Web API `Request` object
- **Output**: Standard Web API `Response` object
- **Async**: Handler must be async (can use `await`)

### Request Object

The `Request` object provides:

```typescript
req.method        // HTTP method: "GET", "POST", etc.
req.url           // Full URL string
req.headers       // Headers object
req.body          // ReadableStream for request body
req.bodyUsed      // Boolean indicating if body was read
```

**Reading the body:**

```typescript
// JSON body
const data = await req.json();

// Text body
const text = await req.text();

// ArrayBuffer
const buffer = await req.arrayBuffer();

// FormData
const formData = await req.formData();
```

**URL and query parameters:**

```typescript
const url = new URL(req.url);
const name = url.searchParams.get("name");
const path = url.pathname;
```

### Response Object

Create responses using the `Response` constructor:

```typescript
// JSON response
return Response.json({ message: "Success" });

// JSON with status
return Response.json({ error: "Not found" }, { status: 404 });

// Text response
return new Response("Hello World", {
  headers: { "Content-Type": "text/plain" }
});

// Custom headers
return Response.json(data, {
  headers: {
    "X-Custom-Header": "value",
    "Cache-Control": "no-cache"
  }
});

// Redirect
return Response.redirect("https://example.com", 302);
```

## Environment Variables

Access environment variables via `process.env`:

```typescript
const apiKey = process.env.API_KEY;
const dbUrl = process.env.DATABASE_URL;
```

Set environment variables when deploying (see [Deployment Guide](deployment.md)).

## Invocation Context

Each invocation receives:

- **Invocation ID**: Available via `X-Invocation-Id` header
- **Request metadata**: Method, path, headers, query params
- **Environment variables**: Set at deployment time

```typescript
export default async function handler(req: Request): Promise<Response> {
  const invocationId = req.headers.get("X-Invocation-Id");
  console.log(`Invocation ${invocationId} started`);
  
  // Your logic here
  
  return Response.json({ invocationId });
}
```

## Best Practices

### 1. Error Handling

Always handle errors gracefully:

```typescript
export default async function handler(req: Request): Promise<Response> {
  try {
    const data = await req.json();
    // Process data
    return Response.json({ success: true });
  } catch (error) {
    return Response.json(
      { error: error.message },
      { status: 400 }
    );
  }
}
```

### 2. Input Validation

Validate inputs before processing:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const id = url.searchParams.get("id");
  
  if (!id || !/^\d+$/.test(id)) {
    return Response.json(
      { error: "Invalid ID" },
      { status: 400 }
    );
  }
  
  // Process valid ID
  return Response.json({ id });
}
```

### 3. Async Operations

Use `await` for async operations:

```typescript
export default async function handler(req: Request): Promise<Response> {
  // Good: await async operations
  const data = await fetch("https://api.example.com/data");
  const json = await data.json();
  
  // Bad: don't forget await
  // const data = fetch("https://api.example.com/data"); // Missing await!
  
  return Response.json(json);
}
```

### 4. Timeout Awareness

Functions have a default timeout (30s). For long-running operations:

```typescript
export default async function handler(req: Request): Promise<Response> {
  // Use AbortController for cancellable operations
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 25000); // 25s safety margin
  
  try {
    const data = await fetch("https://slow-api.com", {
      signal: controller.signal
    });
    clearTimeout(timeout);
    return Response.json(await data.json());
  } catch (error) {
    clearTimeout(timeout);
    if (error.name === "AbortError") {
      return Response.json(
        { error: "Request timeout" },
        { status: 504 }
      );
    }
    throw error;
  }
}
```

### 5. Logging

Use `console.log`, `console.error`, etc. for logging:

```typescript
export default async function handler(req: Request): Promise<Response> {
  console.log("Function invoked", { method: req.method });
  
  try {
    // Process request
    console.log("Processing successful");
    return Response.json({ success: true });
  } catch (error) {
    console.error("Processing failed", error);
    return Response.json(
      { error: "Internal error" },
      { status: 500 }
    );
  }
}
```

Logs are automatically captured and associated with the invocation.

## Testing Functions Locally

### Direct Testing

Test your function directly with Bun:

```typescript
// test-function.ts
import handler from "./my-function.ts";

const req = new Request("http://localhost/?name=Alice");
const res = await handler(req);
console.log(await res.json());
```

Run: `bun test-function.ts`

### Integration Testing

Use the setup script to test with the full stack:

```bash
# Build your function
bun build my-function.ts --outdir ./data/bundles/test-func/v1 --target bun --outfile bundle.js

# Setup test function
./scripts/setup-test-function.sh

# Test via HTTP
curl -X POST "http://localhost:8080/functions/test-func?name=Alice"
```

## Function Examples

### Simple JSON API

```typescript
export default async function handler(req: Request): Promise<Response> {
  if (req.method !== "POST") {
    return Response.json(
      { error: "Method not allowed" },
      { status: 405 }
    );
  }
  
  const data = await req.json();
  return Response.json({
    received: data,
    timestamp: new Date().toISOString()
  });
}
```

### Query Parameter Handler

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const action = url.searchParams.get("action");
  
  switch (action) {
    case "greet":
      const name = url.searchParams.get("name") || "World";
      return Response.json({ message: `Hello, ${name}!` });
    
    case "time":
      return Response.json({ time: new Date().toISOString() });
    
    default:
      return Response.json(
        { error: "Invalid action" },
        { status: 400 }
      );
  }
}
```

### External API Proxy

```typescript
export default async function handler(req: Request): Promise<Response> {
  const apiKey = process.env.API_KEY;
  if (!apiKey) {
    return Response.json(
      { error: "API key not configured" },
      { status: 500 }
    );
  }
  
  const url = new URL(req.url);
  const targetUrl = url.searchParams.get("url");
  
  if (!targetUrl) {
    return Response.json(
      { error: "Missing url parameter" },
      { status: 400 }
    );
  }
  
  const response = await fetch(targetUrl, {
    headers: {
      "Authorization": `Bearer ${apiKey}`
    }
  });
  
  return Response.json({
    status: response.status,
    data: await response.json()
  });
}
```

## Limitations (v1)

Functions run in a Bun process with these limitations:

- **No child processes**: Cannot spawn subprocesses
- **No file system access**: Limited to temp directory (future)
- **Network access**: Open (will be restricted in future)
- **Memory limits**: Configurable per function
- **Timeout**: Default 30s, configurable
- **Single-threaded**: One invocation per worker at a time

## Next Steps

- Review [Examples](../examples/) for more patterns
- Read [Deployment Guide](deployment.md) for production deployment
- Check [API Reference](api-reference.md) for advanced features
