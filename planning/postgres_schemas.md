# PostgreSQL Schema Requirements

This document defines the database schemas for the **BunAuth** and **Platform** services. We will use two separate logical databases (or schemas) to ensure service isolation.

## Common Standards
- **IDs**: Use `UUID` type. Application generates UUID v4 or DB uses `gen_random_uuid()`.
- **Timestamps**: Use `TIMESTAMPTZ` (Timestamp with time zone). Stored as UTC.
- **Strings**: Use `TEXT` or `VARCHAR(255)` as appropriate.
- **Encoding**: UTF-8.

---

## Database 1: `bun_auth`
**Owner**: `bun-auth` service.
**Purpose**: Centralized identity management.

### Table: `users`
Core user identity.
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | User's unique ID. |
| `email` | `VARCHAR(255)` | `UNIQUE NOT NULL` | Login email (indexed). |
| `password_hash` | `TEXT` | `NOT NULL` | Bcrypt hash. |
| `name` | `TEXT` | `NOT NULL` | Display name. |
| `role` | `VARCHAR(50)` | `DEFAULT 'user'` | `user` or `admin`. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

### Table: `sessions`
Web console sessions.
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | Session ID. |
| `user_id` | `UUID` | `REFERENCES users(id) ON DELETE CASCADE` | |
| `token` | `TEXT` | `UNIQUE NOT NULL` | Session token (Secure random). |
| `user_agent` | `TEXT` | | (Optional) |
| `ip_address` | `INET` | | (Optional) |
| `expires_at` | `TIMESTAMPTZ` | `NOT NULL` | |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

### Table: `service_secrets`
Credentials for internal services (e.g., Platform API authentication).
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `client_id` | `VARCHAR(50)` | `PRIMARY KEY` | Service ID (e.g. `platform-api`). |
| `secret_hash` | `TEXT` | `NOT NULL` | Bcrypt hash of the API key/Secret. |
| `permissions` | `JSONB` | `DEFAULT '[]'` | Array of scopes (e.g. `["kms:read"]`). |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

---

## Database 2: `bun_platform`
**Owner**: `platform` service.
**Purpose**: User resources (Projects, Functions).
**Note**: `user_id` refers to `bun_auth.users(id)` but enforced at application level (Soft FK).

### Table: `projects`
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | |
| `name` | `TEXT` | `NOT NULL` | Project display name. |
| `slug` | `VARCHAR(63)` | `UNIQUE NOT NULL` | URL-safe identifier (indexed). |
| `owner_id` | `UUID` | `NOT NULL` | Refers to `bun_auth.users(id)`. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

### Table: `project_members`
Collaboration logic.
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | |
| `project_id` | `UUID` | `REFERENCES projects(id) ON DELETE CASCADE` | |
| `user_id` | `UUID` | `NOT NULL` | Refers to `bun_auth.users(id)`. |
| `role` | `VARCHAR(50)` | `DEFAULT 'member'` | `owner`, `admin`, `member`, `viewer`. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| **Constraint** | `UNIQUE(project_id, user_id)` | | One role per user per project. |

### Table: `functions`
Connects Platform projects to the Functions Service.
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | Platform-side ID. |
| `project_id` | `UUID` | `REFERENCES projects(id) ON DELETE CASCADE` | |
| `name` | `VARCHAR(63)` | `NOT NULL` | Internal function name. |
| `function_service_id` | `VARCHAR(255)` | `NOT NULL` | ID in the `functions` subsystem. |
| `runtime` | `VARCHAR(50)` | `NOT NULL` | e.g., `bun-v1`. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| **Constraint** | `UNIQUE(project_id, name)` | | Unique function names in a project. |

### Table: `api_tokens`
User-generated API keys for CLI access to the Platform.
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | `PRIMARY KEY` | |
| `user_id` | `UUID` | `NOT NULL` | Refers to `bun_auth.users(id)`. |
| `name` | `TEXT` | `NOT NULL` | Token description (e.g. "CI/CD"). |
| `token_hash` | `TEXT` | `UNIQUE NOT NULL` | Hash of the `bpt_...` token. |
| `scopes` | `JSONB` | `DEFAULT '["*"]'` | Permissions. |
| `expires_at` | `TIMESTAMPTZ` | `NULL` | Optional expiration. |
| `last_used_at` | `TIMESTAMPTZ` | `NULL` | |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

## Migration Strategy
1.  Services will auto-migrate on startup (using `golang-migrate` or custom logic).
2.  `bun-auth` starts first -> creates `bun_auth` DB tables.
3.  `platform` starts second -> creates `bun_platform` DB tables.
