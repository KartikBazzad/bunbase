# BunAuth Service Specification

**BunAuth** is a dedicated microservice for authentication and authorization within the BunBase ecosystem. It replaces the embedded auth logic in the Platform API and provides a centralized verification mechanism for all services.

## Core Responsibilities

1.  **User Authentication**: Login, Registration, Session Management.
2.  **Token Issuance**: Minting JWTs (RS256) for authenticated users.
3.  **Service Authentication**: Verifying identity of internal services (Platform calling Functions).
4.  **Verification RPC**: Provide a high-performance endpoint for services to validate tokens.

## Architecture & Bootstrapping

- **Runtime**: Go (standalone binary).
- **Database**: SQLite (`./data/auth.db`), persisted via Docker volume.
- **Keys**: Generates `rsa_private.pem` and `rsa_public.pem` on first start if missing. Public key is exposed via JWKS endpoint for services to cache (optional, or use RPC verify).
- **Admin Init**:
  - Checks for `BUNAUTH_ADMIN_EMAIL` and `BUNAUTH_ADMIN_PASSWORD` env vars.
  - If set and user doesn't exist, creates a super-admin account.

## API Specification (JSON-RPC 2.0)

Base URL: `http://bun-auth:50051/rpc`

### Data Structures

#### User JWT Claims

```json
{
  "iss": "bun-auth",
  "sub": "user_uuid_123",
  "type": "user",
  "role": "admin", // or "user"
  "name": "Alice",
  "email": "alice@example.com",
  "iat": 1700000000,
  "exp": 1700086400 // 24 hours
}
```

#### Service Token Claims

```json
{
  "iss": "bun-auth",
  "sub": "platform-api",
  "type": "service",
  "scopes": ["functions:deploy", "kms:read"],
  "iat": 1700000000,
  "exp": 1700003600 // 1 hour (short lived)
}
```

### Methods

#### 1. `auth.login`

Authenticates a user and returns a JWT.

**Request:**

```json
{
  "method": "auth.login",
  "params": { "email": "...", "password": "..." },
  "id": 1
}
```

**Response:**

```json
{ "result": { "token": "ey...", "user": { ... } }, "id": 1 }
```

**Errors:**

- `1001`: Invalid Credentials
- `1002`: Account Locked

#### 2. `auth.verify`

Verifies a token (User JWT or Service Token).

**Request:**

```json
{ "method": "auth.verify", "params": { "token": "ey..." }, "id": 2 }
```

**Response:**

```json
{
  "result": {
    "valid": true,
    "claims": { "sub": "...", "role": "admin", "scopes": [...] }
  }
}
```

**Benefits**: Centralized revocation check. If we distrust a token (e.g., logout), this endpoint returns `valid: false`.

#### 3. `service.exchange`

Exchanges a long-lived Service Secret for a short-lived Access Token.

**Request:**

```json
{
  "method": "service.exchange",
  "params": {
    "client_id": "platform",
    "client_secret": "env_secret_value"
  }
}
```

**Response:**

```json
{ "result": { "token": "ey...service_jwt..." } }
```

## Database Schema (PostgreSQL)

### `users`
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary Key |
| email | VARCHAR(255) | Unique, Indexed |
| password_hash | TEXT | Bcrypt |
| name | TEXT | |
| role | VARCHAR(50) | `user`, `admin` |
| created_at | TIMESTAMP | |

### `service_secrets`
| Column | Type | Notes |
|--------|------|-------|
| client_id | VARCHAR(50) | Primary Key (e.g., `platform`) |
| secret_hash | TEXT | Bcrypt hash of api key |
| permissions | JSONB | Array of scopes |

### `revocations` 
(For immediate logout support)
| Column | Type | Notes |
|--------|------|-------|
| token_jti | UUID | JWT ID |
| expires_at | TIMESTAMP | When we can purge this row |

## Implementation Plan

1.  **Repo Setup**: `bunbase/bun-auth`
2.  **Dependencies**:
    -   `golang-jwt/jwt`
    -   `lib/pq` (or `pgx`)
    -   `crypto/bcrypt`
3.  **Bootstrapping**:
    -   On startup, connect to Postgres.
    -   Run migrations (create tables).
    -   If users table empty, create default admin from env.
