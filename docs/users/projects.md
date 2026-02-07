# Managing Projects in BunBase

Projects help you organize your functions and manage access. This guide covers everything about projects.

## What are Projects?

A **project** is a container for:

- Multiple functions
- Project-specific settings
- Access control (future: team members)

Each project has:

- **Name**: Human-readable name (e.g., "My API")
- **Slug**: URL-friendly identifier (e.g., "my-api")
- **ID**: Unique identifier (UUID)

## Creating Projects

### Via Web Dashboard

1. Log in to the dashboard
2. Click **"Create Project"**
3. Enter a project name
4. Click **"Create"**

The slug is automatically generated from the name.

### Via CLI

```bash
bunbase projects create "My Project"
```

### Via API

```bash
curl -X POST http://localhost:3001/api/projects \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"My Project"}'
```

## Listing Projects

### Via Web Dashboard

Projects appear on the dashboard homepage.

### Via CLI

```bash
bunbase projects list
```

Output:

```
ID                                    Name        Slug
------------------------------------  ----------  ----------
550e8400-e29b-41d4-a716-446655440000  My Project  my-project
```

### Via API

```bash
curl http://localhost:3001/api/projects -b cookies.txt
```

## Selecting a Project

### Via CLI

Set the active project for subsequent commands:

```bash
bunbase projects use <project-id>
```

After setting, all `deploy` commands will use this project.

### Via Web Dashboard

Click on a project card to view and manage it.

## Project Details

### View Project

**API:**

```bash
curl http://localhost:3001/api/projects/<project-id> -b cookies.txt
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Project",
  "slug": "my-project",
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:00:00Z"
}
```

## Updating Projects

### Via Web Dashboard

1. Navigate to project
2. Click **"Settings"** or **"Edit"**
3. Update name
4. Save changes

### Via CLI

Currently not supported. Use API or dashboard.

### Via API

```bash
curl -X PUT http://localhost:3001/api/projects/<project-id> \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"Updated Name"}'
```

## Deleting Projects

### Via Web Dashboard

1. Navigate to project
2. Click **"Settings"**
3. Click **"Delete Project"**
4. Confirm deletion

**Warning**: This deletes all functions in the project!

### Via API

```bash
curl -X DELETE http://localhost:3001/api/projects/<project-id> \
  -b cookies.txt
```

## Project Functions

### List Functions in Project

**API:**

```bash
curl http://localhost:3001/api/projects/<project-id>/functions \
  -b cookies.txt
```

### Deploy Function to Project

**CLI:**

```bash
# Set active project first
bunbase projects use <project-id>

# Deploy function
bunbase functions deploy --file handler.ts --name api --runtime bun --handler default
```

**API:**

```bash
curl -X POST http://localhost:3001/api/projects/<project-id>/functions \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name": "api",
    "runtime": "bun",
    "handler": "default",
    "version": "v1",
    "bundle": "base64-encoded-bundle"
  }'
```

## Best Practices

### 1. One Project Per Application

Organize by application or service:

- `my-api` - Main API project
- `background-jobs` - Background processing
- `admin-tools` - Admin utilities

### 2. Use Descriptive Names

Good names:

- `user-api`
- `payment-processor`
- `analytics-service`

Avoid:

- `project1`
- `test`
- `new-project`

### 3. Keep Related Functions Together

Group functions that:

- Share the same domain
- Use the same data sources
- Have related functionality

## Project Limits

Current limits (v1):

- No limit on number of projects per user
- No limit on functions per project
- Projects are single-tenant (one owner)

## See Also

- [Getting Started](getting-started.md)
- [CLI Guide](cli-guide.md)
- [Platform API Reference](api-reference.md)
