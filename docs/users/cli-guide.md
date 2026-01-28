# BunBase CLI Guide

The BunBase CLI (`bunbase`) is a command-line tool for managing your projects and deploying functions.

## Installation

### Build from Source

```bash
cd platform
go build -o bunbase ./cmd/cli
```

Add to your PATH (optional):

```bash
sudo mv bunbase /usr/local/bin/
```

## Authentication

### Login

```bash
bunbase auth login
```

Prompts for:

- Email address
- Password

Stores session cookie for subsequent commands.

### Logout

```bash
bunbase auth logout
```

Clears stored session.

### Check Current User

```bash
bunbase auth me
```

Shows your current user information.

## Projects

### List Projects

```bash
bunbase projects list
```

Output:

```
ID                                    Name        Slug
------------------------------------  ----------  ----------
550e8400-e29b-41d4-a716-446655440000  My Project  my-project
```

### Create Project

```bash
bunbase projects create <name>
```

Example:

```bash
bunbase projects create "My Awesome App"
# Creates project with slug: my-awesome-app
```

### Use Project

Set the active project for subsequent commands:

```bash
bunbase projects use <project-id>
```

The project ID is shown in `projects list`. After setting, all `deploy` commands will use this project.

### Get Project Info

```bash
bunbase projects get <project-id>
```

Shows project details including:

- Name and slug
- Created date
- Function count

## Function Deployment

### Deploy a Function

```bash
bunbase deploy <file> --name <name> --runtime <runtime> --handler <handler> [--version <version>]
```

**Required flags:**

- `--name`: Function name (e.g., `hello-world`)
- `--runtime`: Runtime type (currently: `bun`)
- `--handler`: Handler export name (usually `default`)

**Optional flags:**

- `--version`: Version tag (default: `v1`)
- `--project`: Project ID (uses active project if not specified)

**Example:**

```bash
bunbase deploy src/handler.ts \
  --name api-handler \
  --runtime bun \
  --handler default \
  --version v2
```

### Function File Format

Your function file must export a default handler:

```typescript
export default async function handler(req: Request): Promise<Response> {
  // Your function logic
  return Response.json({ message: "Hello!" });
}
```

### Bundle Requirements

Before deploying, ensure your function:

- Exports a default async function
- Accepts a `Request` object
- Returns a `Response` object
- Is bundled (if using dependencies)

**Bundle with Bun:**

```bash
bun build src/handler.ts --outfile bundle.js --target bun
```

Then deploy the bundle:

```bash
bunbase deploy bundle.js --name handler --runtime bun --handler default
```

## Function Management

### List Functions

```bash
bunbase functions list [--project <project-id>]
```

Lists all functions in the active (or specified) project.

### Delete Function

```bash
bunbase functions delete <function-id> [--project <project-id>]
```

Removes a function from the project and functions service.

## Configuration

### Environment Variables

Set environment variables for functions via the dashboard or API. They're available in your function via `process.env`.

### Project Context

The CLI stores your active project in a local config file. To see current context:

```bash
bunbase projects current
```

## Examples

### Complete Workflow

```bash
# 1. Login
bunbase auth login

# 2. Create project
bunbase projects create "My API"

# 3. Set as active
bunbase projects use <project-id>

# 4. Deploy function
bunbase deploy api.ts --name api --runtime bun --handler default

# 5. List functions
bunbase functions list
```

### Deploying Multiple Versions

```bash
# Deploy v1
bunbase deploy handler.ts --name api --runtime bun --handler default --version v1

# Deploy v2
bunbase deploy handler.ts --name api --runtime bun --handler default --version v2
```

## Troubleshooting

### "Not authenticated"

Run `bunbase auth login` to authenticate.

### "No active project"

Set an active project with `bunbase projects use <project-id>`.

### "Function deployment failed"

- Verify the function file exists and is readable
- Check that the function exports a default handler
- Ensure the runtime is supported (`bun`)
- Review function logs in the dashboard

### "Project not found"

- List projects: `bunbase projects list`
- Verify project ID is correct
- Ensure you have access to the project

## Advanced Usage

### Using with Scripts

```bash
#!/bin/bash
# deploy.sh

PROJECT_ID=$(bunbase projects list | grep "My Project" | awk '{print $1}')
bunbase projects use "$PROJECT_ID"
bunbase deploy src/api.ts --name api --runtime bun --handler default
```

### CI/CD Integration

```bash
# In your CI pipeline
bunbase auth login --email "$BUNBASE_EMAIL" --password "$BUNBASE_PASSWORD"
bunbase projects use "$BUNBASE_PROJECT_ID"
bunbase deploy dist/handler.js --name "$FUNCTION_NAME" --runtime bun --handler default
```

## See Also

- [Getting Started](getting-started.md)
- [Writing Functions](writing-functions.md)
- [Platform API](api-reference.md)
