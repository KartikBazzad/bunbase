# Project Auth Service Requirements (`tenant-auth`)

This service handles authentication for **users of the applications built on BunBase**. It is distinct from `bun-auth`, which authenticates developers using the platform.

## Goal
Provide a drop-in authentication solution for BunBase projects, accessible via the `bunbase-js` Client SDK.

## Core Features
1.  **Multi-Tenancy**: Each project has its own isolated user pool.
2.  **Providers**:
    -   **Email/Password**: Hashed storage, registration, login, password reset.
    -   **OAuth2**: Google, GitHub, Apple (configurable per project).
    -   **Anonymous**: Temporary guest sessions.
    -   **Custom**: JWTs signed by external providers.
3.  **Token Management**:
    -   Issue **ID Tokens** (JWT) for client-side use.
    -   Issue **Refresh Tokens** for long-lived sessions.
    -   Verify tokens for `bundoc` and `functions` access.

## Architecture

### Database Schema (Per Project)
This data should likely live in the **Platform Postgres** but partitioned by `project_id`.

```sql
-- Table: project_users
id UUID
project_id UUID (FK)
email VARCHAR
password_hash VARCHAR
provider VARCHAR
last_login TIMESTAMP
created_at TIMESTAMP

-- Table: project_user_sessions
id UUID
user_id UUID
refresh_token VARCHAR
expires_at TIMESTAMP
```

### API Endpoints
exposed via `api.bunbase.com/v1/auth`

-   `POST /signup`: Register new user.
-   `POST /login`: Exchange credentials for ID/Refresh tokens.
-   `POST /refresh`: Get new ID token.
-   `POST /logout`: Revoke refresh token.
-   `POST /reset-password`: Send email (future integration) or generate link.

### Integration with SDK
The `bunbase-js` SDK `auth` module connects directly to these endpoints.

## Config
Stored in `project_settings` (Platform DB):
-   `auth_enabled`: boolean
-   `auth_providers`: JSON array
-   `jwt_secret`: Derived or stored in BunKMS? -> **Decision**: Use **BunKMS** to sign tokens.

## Security
-   Rate limiting on login endpoints.
-   Passwords hashed with Argon2id.
-   JWTs signed using RS256 keys managed by **BunKMS**.
