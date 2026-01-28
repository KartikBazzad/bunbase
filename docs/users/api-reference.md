# BunBase Platform API Reference

Complete REST API reference for the BunBase Platform.

## Base URL

```
http://localhost:3001/api
```

(Replace with your deployment URL in production)

## Authentication

Most endpoints require authentication via session cookies. Register or login to obtain a session.

### Register

```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "created_at": "2026-01-28T12:00:00Z"
}
```

### Login

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com"
  }
}
```

Sets a session cookie automatically.

### Logout

```http
POST /api/auth/logout
```

**Response:**

```json
{
  "message": "Logged out"
}
```

### Get Current User

```http
GET /api/auth/me
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "created_at": "2026-01-28T12:00:00Z"
}
```

## Projects

### List Projects

```http
GET /api/projects
```

**Response:**

```json
[
  {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "My Project",
    "slug": "my-project",
    "created_at": "2026-01-28T12:00:00Z",
    "updated_at": "2026-01-28T12:00:00Z"
  }
]
```

### Create Project

```http
POST /api/projects
Content-Type: application/json

{
  "name": "My New Project"
}
```

**Response:**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "name": "My New Project",
  "slug": "my-new-project",
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:00:00Z"
}
```

### Get Project

```http
GET /api/projects/:id
```

**Response:**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "name": "My Project",
  "slug": "my-project",
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:00:00Z"
}
```

### Update Project

```http
PUT /api/projects/:id
Content-Type: application/json

{
  "name": "Updated Project Name"
}
```

**Response:**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "name": "Updated Project Name",
  "slug": "updated-project-name",
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:30:00Z"
}
```

### Delete Project

```http
DELETE /api/projects/:id
```

**Response:**

```json
{
  "message": "Project deleted"
}
```

## Functions

### List Functions

```http
GET /api/projects/:projectId/functions
```

**Response:**

```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "project_id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "hello-world",
    "runtime": "bun",
    "function_service_id": "func-my-project-hello-world",
    "created_at": "2026-01-28T12:00:00Z",
    "updated_at": "2026-01-28T12:00:00Z"
  }
]
```

### Deploy Function

```http
POST /api/projects/:projectId/functions
Content-Type: application/json

{
  "name": "hello-world",
  "runtime": "bun",
  "handler": "default",
  "version": "v1",
  "bundle": "base64-encoded-bundle-content"
}
```

**Request Fields:**

- `name` (string, required): Function name
- `runtime` (string, required): Runtime type (`bun`)
- `handler` (string, required): Handler export name
- `version` (string, optional): Version tag (default: `v1`)
- `bundle` (string, required): Base64-encoded function bundle

**Response:**

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440000",
  "project_id": "660e8400-e29b-41d4-a716-446655440000",
  "name": "hello-world",
  "runtime": "bun",
  "function_service_id": "func-my-project-hello-world",
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:00:00Z"
}
```

### Delete Function

```http
DELETE /api/projects/:projectId/functions/:functionId
```

**Response:**

```json
{
  "message": "Function deleted"
}
```

## Error Responses

All endpoints may return error responses:

```json
{
  "error": "Error message here"
}
```

Common HTTP status codes:

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Examples

### Complete Workflow

```bash
# 1. Register
curl -X POST http://localhost:3001/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' \
  -c cookies.txt

# 2. Create Project
curl -X POST http://localhost:3001/api/projects \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"My API"}'

# 3. Deploy Function
BUNDLE=$(base64 -i bundle.js)
curl -X POST http://localhost:3001/api/projects/PROJECT_ID/functions \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d "{
    \"name\": \"hello\",
    \"runtime\": \"bun\",
    \"handler\": \"default\",
    \"version\": \"v1\",
    \"bundle\": \"$BUNDLE\"
  }"
```

### Using JavaScript/TypeScript

```typescript
const API_BASE = "http://localhost:3001/api";

// Login
const loginRes = await fetch(`${API_BASE}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  credentials: "include",
  body: JSON.stringify({
    email: "user@example.com",
    password: "password123",
  }),
});

// Create Project
const projectRes = await fetch(`${API_BASE}/projects`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  credentials: "include",
  body: JSON.stringify({ name: "My Project" }),
});

const project = await projectRes.json();

// Deploy Function
const bundle = await Bun.file("bundle.js").arrayBuffer();
const bundleBase64 = btoa(String.fromCharCode(...new Uint8Array(bundle)));

const deployRes = await fetch(`${API_BASE}/projects/${project.id}/functions`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  credentials: "include",
  body: JSON.stringify({
    name: "hello",
    runtime: "bun",
    handler: "default",
    version: "v1",
    bundle: bundleBase64,
  }),
});
```

## Rate Limits

Currently, there are no rate limits. This may change in future versions.

## See Also

- [Getting Started](getting-started.md)
- [CLI Guide](cli-guide.md)
- [Writing Functions](writing-functions.md)
