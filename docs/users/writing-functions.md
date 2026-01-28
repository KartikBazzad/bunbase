# Writing Functions for BunBase

This guide covers how to write effective JavaScript/TypeScript functions for BunBase.

## Function Structure

Every BunBase function must export a default async handler:

```typescript
export default async function handler(req: Request): Promise<Response> {
  // Your function logic here
  return Response.json({ message: "Hello!" });
}
```

### Handler Signature

- **Input**: `Request` object (Web API standard)
- **Output**: `Promise<Response>` (Web API standard)

This follows the same pattern as Cloudflare Workers, Deno Deploy, and other modern serverless platforms.

## Basic Examples

### Hello World

```typescript
export default async function handler(req: Request): Promise<Response> {
  return Response.json({ message: "Hello, World!" });
}
```

### Query Parameters

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";

  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
  });
}
```

### Request Body

```typescript
export default async function handler(req: Request): Promise<Response> {
  const body = await req.json();
  const { name, age } = body;

  return Response.json({
    message: `Hello, ${name}! You are ${age} years old.`,
  });
}
```

### HTTP Methods

```typescript
export default async function handler(req: Request): Promise<Response> {
  const method = req.method;
  const url = new URL(req.url);

  switch (method) {
    case "GET":
      return Response.json({ message: "GET request" });

    case "POST":
      const body = await req.json();
      return Response.json({ received: body });

    case "PUT":
      return Response.json({ message: "PUT request" });

    case "DELETE":
      return Response.json({ message: "DELETE request" });

    default:
      return Response.json({ error: "Method not allowed" }, { status: 405 });
  }
}
```

## Working with Headers

### Reading Headers

```typescript
export default async function handler(req: Request): Promise<Response> {
  const authHeader = req.headers.get("Authorization");
  const contentType = req.headers.get("Content-Type");

  return Response.json({
    auth: authHeader,
    contentType: contentType,
  });
}
```

### Setting Response Headers

```typescript
export default async function handler(req: Request): Promise<Response> {
  return Response.json(
    { message: "Hello!" },
    {
      headers: {
        "X-Custom-Header": "value",
        "Cache-Control": "max-age=3600",
      },
    },
  );
}
```

## Error Handling

### Returning Errors

```typescript
export default async function handler(req: Request): Promise<Response> {
  try {
    const data = await processRequest(req);
    return Response.json(data);
  } catch (error) {
    return Response.json({ error: error.message }, { status: 500 });
  }
}
```

### Validation

```typescript
export default async function handler(req: Request): Promise<Response> {
  const body = await req.json();

  if (!body.email) {
    return Response.json({ error: "Email is required" }, { status: 400 });
  }

  // Process valid request
  return Response.json({ success: true });
}
```

## Using Environment Variables

Environment variables are available via `process.env`:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const apiKey = process.env.API_KEY;
  const dbUrl = process.env.DATABASE_URL;

  // Use environment variables
  return Response.json({
    hasApiKey: !!apiKey,
    dbConfigured: !!dbUrl,
  });
}
```

Set environment variables in the dashboard or via API when deploying.

## Working with External APIs

```typescript
export default async function handler(req: Request): Promise<Response> {
  const apiKey = process.env.EXTERNAL_API_KEY;

  const response = await fetch("https://api.example.com/data", {
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
  });

  const data = await response.json();
  return Response.json(data);
}
```

## Using npm Packages

BunBase functions run on Bun, which supports npm packages. Install dependencies before bundling:

```bash
# In your function directory
bun install
```

Then bundle with dependencies:

```bash
bun build src/handler.ts --outfile bundle.js --target bun
```

### Example with Dependencies

```typescript
import { z } from "zod";

const schema = z.object({
  name: z.string(),
  email: z.string().email(),
});

export default async function handler(req: Request): Promise<Response> {
  const body = await req.json();
  const result = schema.safeParse(body);

  if (!result.success) {
    return Response.json({ error: result.error.errors }, { status: 400 });
  }

  return Response.json({ valid: true, data: result.data });
}
```

## Best Practices

### 1. Keep Functions Focused

Each function should do one thing well:

```typescript
// ✅ Good: Single responsibility
export default async function handler(req: Request): Promise<Response> {
  const { userId } = await req.json();
  const user = await getUser(userId);
  return Response.json(user);
}

// ❌ Bad: Multiple responsibilities
export default async function handler(req: Request): Promise<Response> {
  // Gets user, processes payment, sends email, updates database...
}
```

### 2. Handle Errors Gracefully

```typescript
export default async function handler(req: Request): Promise<Response> {
  try {
    const result = await processRequest(req);
    return Response.json(result);
  } catch (error) {
    console.error("Function error:", error);
    return Response.json({ error: "Internal server error" }, { status: 500 });
  }
}
```

### 3. Validate Input

```typescript
export default async function handler(req: Request): Promise<Response> {
  const body = await req.json();

  if (typeof body.name !== "string" || body.name.length === 0) {
    return Response.json({ error: "Invalid name" }, { status: 400 });
  }

  // Process valid input
  return Response.json({ success: true });
}
```

### 4. Use Appropriate HTTP Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error

### 5. Logging

Use `console.log()` for debugging (visible in function logs):

```typescript
export default async function handler(req: Request): Promise<Response> {
  console.log("Function invoked:", req.method, req.url);

  const result = await processRequest(req);

  console.log("Processing complete:", result.id);

  return Response.json(result);
}
```

### 6. Timeout Awareness

Functions have a 30-second timeout. For long-running operations, consider:

- Breaking work into smaller chunks
- Using background jobs (future feature)
- Optimizing algorithms

## Advanced Patterns

### REST API Handler

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const path = url.pathname;
  const method = req.method;

  // Route: /api/users/:id
  const userMatch = path.match(/^\/api\/users\/(.+)$/);

  if (userMatch && method === "GET") {
    const userId = userMatch[1];
    const user = await getUser(userId);
    return Response.json(user);
  }

  // Route: /api/users
  if (path === "/api/users" && method === "POST") {
    const body = await req.json();
    const user = await createUser(body);
    return Response.json(user, { status: 201 });
  }

  return Response.json({ error: "Not found" }, { status: 404 });
}
```

### Middleware Pattern

```typescript
async function withAuth(req: Request, handler: Function): Promise<Response> {
  const authHeader = req.headers.get("Authorization");
  if (!authHeader) {
    return Response.json({ error: "Unauthorized" }, { status: 401 });
  }

  return handler(req);
}

export default async function handler(req: Request): Promise<Response> {
  return withAuth(req, async (req: Request) => {
    // Your protected logic
    return Response.json({ message: "Protected resource" });
  });
}
```

## Testing Functions Locally

Test your functions before deploying:

```typescript
// handler.ts
export default async function handler(req: Request): Promise<Response> {
  return Response.json({ message: "Hello!" });
}

// test.ts
import handler from "./handler";

const req = new Request("http://localhost/");
const res = await handler(req);
console.log(await res.json());
```

Run with Bun:

```bash
bun test.ts
```

## See Also

- [Getting Started](getting-started.md)
- [CLI Guide](cli-guide.md)
- [Platform API](api-reference.md)
- [Function Examples](../../functions/docs/examples.md)
