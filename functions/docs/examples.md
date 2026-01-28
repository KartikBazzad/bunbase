# Function Examples

A collection of example functions demonstrating various patterns and use cases.

## Basic Examples

### Hello World

Simple greeting function:

```typescript
// examples/hello-world.ts
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
  });
}
```

**Usage:**
```bash
curl "http://localhost:8080/functions/hello-world?name=Alice"
```

---

### Echo Function

Echo back request data:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  
  const data = {
    method: req.method,
    path: url.pathname,
    query: Object.fromEntries(url.searchParams),
    headers: Object.fromEntries(req.headers.entries()),
    body: req.body ? await req.text() : null,
  };
  
  return Response.json(data);
}
```

---

### JSON API Handler

Handle JSON requests and responses:

```typescript
export default async function handler(req: Request): Promise<Response> {
  if (req.method !== "POST") {
    return Response.json(
      { error: "Method not allowed" },
      { status: 405 }
    );
  }
  
  try {
    const data = await req.json();
    
    // Process data
    const result = {
      received: data,
      processed: true,
      timestamp: new Date().toISOString(),
    };
    
    return Response.json(result);
  } catch (error) {
    return Response.json(
      { error: "Invalid JSON" },
      { status: 400 }
    );
  }
}
```

---

## Advanced Examples

### API Proxy

Proxy requests to external APIs:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const targetUrl = url.searchParams.get("url");
  
  if (!targetUrl) {
    return Response.json(
      { error: "Missing 'url' query parameter" },
      { status: 400 }
    );
  }
  
  try {
    const apiKey = process.env.API_KEY;
    const response = await fetch(targetUrl, {
      method: req.method,
      headers: {
        ...Object.fromEntries(req.headers.entries()),
        ...(apiKey && { "Authorization": `Bearer ${apiKey}` }),
      },
      body: req.body,
    });
    
    const data = await response.json();
    
    return Response.json({
      status: response.status,
      data,
    });
  } catch (error) {
    return Response.json(
      { error: error.message },
      { status: 500 }
    );
  }
}
```

---

### Request Router

Route requests based on path:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const path = url.pathname;
  
  // Remove function prefix if present
  const route = path.replace(/^\/functions\/[^/]+/, "") || "/";
  
  switch (route) {
    case "/":
      return Response.json({ message: "Home" });
    
    case "/users":
      return Response.json({ users: [] });
    
    case "/health":
      return Response.json({ status: "ok" });
    
    default:
      return Response.json(
        { error: "Not found" },
        { status: 404 }
      );
  }
}
```

---

### Form Data Handler

Handle form submissions:

```typescript
export default async function handler(req: Request): Promise<Response> {
  if (req.method !== "POST") {
    return Response.json(
      { error: "Method not allowed" },
      { status: 405 }
    );
  }
  
  try {
    const formData = await req.formData();
    const data: Record<string, string> = {};
    
    for (const [key, value] of formData.entries()) {
      data[key] = value.toString();
    }
    
    return Response.json({
      received: data,
      count: Object.keys(data).length,
    });
  } catch (error) {
    return Response.json(
      { error: "Invalid form data" },
      { status: 400 }
    );
  }
}
```

---

### Error Handler with Logging

Comprehensive error handling:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const invocationId = req.headers.get("X-Invocation-Id");
  
  try {
    console.log(`[${invocationId}] Request received`, {
      method: req.method,
      path: req.url,
    });
    
    // Your logic here
    const result = await processRequest(req);
    
    console.log(`[${invocationId}] Request processed successfully`);
    
    return Response.json(result);
  } catch (error) {
    console.error(`[${invocationId}] Request failed`, {
      error: error.message,
      stack: error.stack,
    });
    
    return Response.json(
      {
        error: "Internal server error",
        invocationId,
      },
      { status: 500 }
    );
  }
}

async function processRequest(req: Request): Promise<any> {
  // Your processing logic
  return { success: true };
}
```

---

### Timeout-Aware Function

Handle long-running operations with timeout:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const controller = new AbortController();
  
  // Set timeout to 25s (5s safety margin from 30s default)
  const timeout = setTimeout(() => {
    controller.abort();
  }, 25000);
  
  try {
    // Long-running operation
    const result = await fetch("https://slow-api.com", {
      signal: controller.signal,
    });
    
    clearTimeout(timeout);
    return Response.json(await result.json());
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

---

### Conditional Response

Return different responses based on conditions:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const format = url.searchParams.get("format") || "json";
  const data = { message: "Hello", timestamp: new Date().toISOString() };
  
  switch (format) {
    case "json":
      return Response.json(data);
    
    case "text":
      return new Response(
        `Message: ${data.message}\nTimestamp: ${data.timestamp}`,
        { headers: { "Content-Type": "text/plain" } }
      );
    
    case "xml":
      const xml = `<?xml version="1.0"?><response><message>${data.message}</message><timestamp>${data.timestamp}</timestamp></response>`;
      return new Response(xml, {
        headers: { "Content-Type": "application/xml" },
      });
    
    default:
      return Response.json(
        { error: "Invalid format" },
        { status: 400 }
      );
  }
}
```

---

## Real-World Patterns

### REST API Endpoint

Simple REST API:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const path = url.pathname.replace(/^\/functions\/[^/]+/, "") || "/";
  const method = req.method;
  
  // GET /items
  if (method === "GET" && path === "/items") {
    return Response.json({ items: [] });
  }
  
  // POST /items
  if (method === "POST" && path === "/items") {
    const data = await req.json();
    return Response.json({ created: data }, { status: 201 });
  }
  
  // GET /items/:id
  const itemMatch = path.match(/^\/items\/(.+)$/);
  if (method === "GET" && itemMatch) {
    const id = itemMatch[1];
    return Response.json({ id, name: "Item" });
  }
  
  return Response.json(
    { error: "Not found" },
    { status: 404 }
  );
}
```

---

### Webhook Handler

Handle webhook requests:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const signature = req.headers.get("X-Webhook-Signature");
  const secret = process.env.WEBHOOK_SECRET;
  
  if (!secret) {
    return Response.json(
      { error: "Webhook secret not configured" },
      { status: 500 }
    );
  }
  
  // Verify signature (simplified)
  const body = await req.text();
  const expectedSig = await computeSignature(body, secret);
  
  if (signature !== expectedSig) {
    return Response.json(
      { error: "Invalid signature" },
      { status: 401 }
    );
  }
  
  // Process webhook
  const data = JSON.parse(body);
  console.log("Webhook received", data);
  
  return Response.json({ received: true });
}

async function computeSignature(body: string, secret: string): Promise<string> {
  // Implement signature computation
  return "signature";
}
```

---

### Data Transformation

Transform request data:

```typescript
export default async function handler(req: Request): Promise<Response> {
  const data = await req.json();
  
  // Transform data
  const transformed = {
    id: data.id || generateId(),
    name: data.name?.toUpperCase(),
    email: data.email?.toLowerCase(),
    createdAt: new Date().toISOString(),
    metadata: {
      source: "function",
      version: "1.0",
    },
  };
  
  return Response.json(transformed);
}

function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}
```

---

## See Also

- [Function Development Guide](function-development.md) - Learn how to write functions
- [API Reference](api-reference.md) - Complete API documentation
- [Getting Started](getting-started.md) - Setup and first function
