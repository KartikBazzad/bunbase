# BunBase Platform API

Go backend API for the BunBase Platform, providing user authentication, project management, and function deployment.

## Features

- User authentication (email/password with session cookies)
- Project management (CRUD operations)
- Function deployment to projects
- Integration with Functions Service via IPC

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3
- Functions service running (for function deployment)

### Build

```bash
cd platform
go mod tidy
go build -o platform-server ./cmd/server
```

### Run

```bash
./platform-server \
  --db-path ./data/platform.db \
  --port 3001 \
  --functions-socket /tmp/functions.sock \
  --bundle-path ../functions/data/bundles \
  --cors-origin http://localhost:5173
```

### Environment Variables

- `PLATFORM_DB_PATH` - Database file path (default: `./data/platform.db`)
- `PLATFORM_PORT` - API server port (default: `3001`)
- `PLATFORM_DEPLOYMENT_MODE` - `cloud` (default) or `self_hosted`. When `self_hosted`, only instance admins can create projects and signup is disabled after one-time setup.
- `PLATFORM_SECRET` - Session secret key (not used yet, for future JWT)
- `PLATFORM_LIMITS_MAX_PROJECTS_PER_USER` - Max projects per user in cloud mode (0 = unlimited)
- `PLATFORM_LIMITS_MAX_FUNCTIONS_PER_PROJECT` - Max functions per project in cloud mode (0 = unlimited)
- `PLATFORM_LIMITS_MAX_API_TOKENS_PER_USER` - Max API tokens per user in cloud mode (0 = unlimited)
- `FUNCTIONS_SOCKET` - Functions service socket path (default: `/tmp/functions.sock`)
- `BUNDLE_BASE_PATH` - Base path for function bundles (default: `../functions/data/bundles`)
- `CORS_ORIGIN` - Allowed CORS origin (default: `http://localhost:5173`)

## API Endpoints

### Authentication

- `POST /api/auth/register` - Register new user (disabled when self-hosted and setup complete)
- `POST /api/auth/login` - Login user
- `POST /api/auth/logout` - Logout user
- `GET /api/auth/me` - Get current user (includes `is_instance_admin` when self-hosted)

### Instance (self-hosted)

- `POST /api/setup` - One-time bootstrap: create root admin (self-hosted only; body: `email`, `password`, `name`)
- `GET /api/instance/status` - Returns `{ "deployment_mode", "setup_complete" }` for dashboard

### Projects

- `GET /api/projects` - List user's projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project
- `PUT /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

### Functions

- `GET /api/projects/:id/functions` - List project functions
- `POST /api/projects/:id/functions` - Deploy function
- `DELETE /api/projects/:id/functions/:functionId` - Delete function

## Database Schema

See `internal/database/db.go` for the complete schema.

## Architecture

The platform API acts as a control plane that:
1. Manages users and projects
2. Handles function deployment requests
3. Communicates with the Functions Service via Unix socket IPC
4. Links functions to projects in the platform database

## Integration with Functions Service

The platform API communicates with the Functions Service using the binary IPC protocol:
- Registers functions in the Functions Service
- Deploys function versions
- Functions are stored in `functions/data/bundles/`
- Function IDs are generated as `func-{project-slug}-{function-name}`

## Security

- Password hashing: bcrypt with cost factor 10
- Session tokens: cryptographically secure random tokens
- Cookies: HttpOnly, Secure (in production), SameSite=Lax
- CORS: Configurable origin
- Input validation: Email format, password strength

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
platform/
├── cmd/server/          # Main application
├── internal/
│   ├── auth/           # Authentication logic
│   ├── database/       # Database schema
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   └── services/       # Business logic
└── pkg/
    └── functions/      # Functions service IPC client
```

## License

MIT
