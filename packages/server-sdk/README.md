# BunBase Server SDK

Official Server SDK for BunBase - designed for server-side operations and administrative access.

## Installation

```bash
npm install @bunbase/server-sdk
# or
bun add @bunbase/server-sdk
```

## Usage

### Basic Setup

```typescript
import { createServerClient } from "@bunbase/server-sdk";

// Using API key authentication
const client = createServerClient({
  apiKey: "your-api-key",
  baseURL: "https://api.bunbase.com",
  projectId: "your-project-id",
});

// Using session-based authentication (for admin operations)
const adminClient = createServerClient({
  baseURL: "https://api.bunbase.com",
  useCookies: true, // Use session cookies
});
```

### Functions Management

#### Create HTTP Function

```typescript
const httpFunction = await client.functions.createHTTPFunction({
  name: "my-api",
  runtime: "nodejs20",
  handler: "index.handler",
  path: "/api/users",
  methods: ["GET", "POST"],
  code: `export async function handler(req) {
    return Response.json({ message: "Hello World" });
  }`,
  memory: 512,
  timeout: 30,
});
```

#### Create Callable Function

```typescript
const callableFunction = await client.functions.createCallableFunction({
  name: "send-email",
  runtime: "nodejs20",
  handler: "index.handler",
  code: `export async function handler(data, context) {
    // context contains auth info (user, session)
    return { success: true };
  }`,
  memory: 256,
  timeout: 10,
});
```

#### List Functions

```typescript
const functions = await client.functions.list();
```

#### Deploy Function

```typescript
const result = await client.functions.deploy(functionId);
console.log(result.version); // "1.0.0"
```

#### Invoke Function

```typescript
const result = await client.functions.invoke(functionId, {
  body: { userId: "123" },
});
```

#### Get Function Logs

```typescript
const logs = await client.functions.getLogs(functionId, {
  limit: 100,
  offset: 0,
});
```

#### Manage Environment Variables

```typescript
// Set environment variable
await client.functions.setEnv(functionId, "API_KEY", "secret-value");

// Delete environment variable
await client.functions.deleteEnv(functionId, "API_KEY");
```

### Admin Operations

Admin operations require session-based authentication (use `useCookies: true`).

#### Projects

```typescript
// List all projects
const projects = await adminClient.admin.projects.list();

// Create project
const project = await adminClient.admin.projects.create({
  name: "My Project",
  description: "Project description",
});

// Get project
const project = await adminClient.admin.projects.get(projectId);

// Update project
await adminClient.admin.projects.update(projectId, {
  name: "Updated Name",
});

// Delete project
await adminClient.admin.projects.delete(projectId);
```

#### Applications

```typescript
// List applications
const apps = await adminClient.admin.applications.list(projectId);

// Create application
const app = await adminClient.admin.applications.create(projectId, {
  name: "My App",
  type: "web",
});

// Generate API key
const apiKey = await adminClient.admin.applications.generateKey(appId);

// Revoke API key
await adminClient.admin.applications.revokeKey(appId);
```

#### Databases

```typescript
// List databases
const databases = await adminClient.admin.databases.list(projectId);

// Create database
const db = await adminClient.admin.databases.create(projectId, {
  name: "my-database",
});
```

#### Storage

```typescript
// List buckets
const buckets = await adminClient.admin.storage.buckets.list(projectId);

// Create bucket
const bucket = await adminClient.admin.storage.buckets.create(projectId, {
  name: "my-bucket",
});

// List files
const { files } = await adminClient.admin.storage.files.list(bucketId, {
  prefix: "uploads/",
  limit: 100,
});
```

#### Collections

```typescript
// List collections
const collections = await adminClient.admin.collections.list(databaseId);

// Create collection
const collection = await adminClient.admin.collections.create(databaseId, {
  name: "users",
});
```

## Configuration

### ServerSDKConfig

```typescript
interface ServerSDKConfig {
  apiKey?: string;           // For API key auth
  baseURL?: string;          // API base URL (default: http://localhost:3000/api)
  projectId?: string;        // Project ID
  useCookies?: boolean;      // For session-based auth (default: false)
  timeout?: number;          // Request timeout in ms (default: 30000)
  retries?: number;         // Number of retries (default: 3)
  retryDelay?: number;      // Retry delay in ms (default: 1000)
}
```

## Authentication

The ServerSDK supports two authentication methods:

1. **API Key Authentication**: Use `apiKey` in config for function operations
2. **Session Authentication**: Use `useCookies: true` for admin operations (requires browser cookies or cookie handling in Node.js)

## Error Handling

All methods throw errors with additional properties:

```typescript
try {
  await client.functions.get("invalid-id");
} catch (error) {
  console.error(error.message);  // Error message
  console.error(error.code);      // Error code
  console.error(error.status);    // HTTP status
  console.error(error.details);   // Additional details
}
```

## TypeScript Support

The SDK is fully typed with TypeScript. All types are exported:

```typescript
import type {
  FunctionResponse,
  FunctionType,
  FunctionRuntime,
  Project,
  Application,
  // ... more types
} from "@bunbase/server-sdk";
```

## License

MIT
