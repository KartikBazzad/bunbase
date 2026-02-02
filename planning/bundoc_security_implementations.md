# Bundoc Security Implementation Plan

**Objective**: To harden the Bundoc database system by implementing robust security controls, ensuring confidentiality, integrity, and availability of data.

---

## 1. Authentication & Authorization (AuthN & AuthZ)

**Goal**: Verify user identity and enforce access controls.

### 1.1 Authentication (AuthN)

- **Mechanism**: Challenge-Response Authentication using SCRAM-SHA-256 (Salted Challenge Response Authentication Mechanism).
- **Storage**: Store users in a dedicated system collection `admin.users`.
- **Credential Storage**: Store `salt`, `stored_key`, `server_key`, and `iteration_count` (not plain passwords).
- **Handshake**:
  1. Client sends `Connect` request with username.
  2. Server sends `Salt` + `IterationCount`.
  3. Client computes proof and sends `ClientProof`.
  4. Server verifies and grants session.

### 1.2 Authorization (AuthZ) - RBAC

- **Roles**:
  - `root`: Full access to all databases and system operations.
  - `dbOwner`: Full access to specific database.
  - `readWrite`: Read and write access to specific database.
  - `read`: Read-only access to specific database.
- **Enforcement**:
  - Middleware in `bundoc-server` intercepting every OpCode.
  - Check `Session.User.Roles` against `OpCode` requirement.

---

## 2. Network Security (Encryption in Transit)

**Goal**: Prevent eavesdropping and man-in-the-middle attacks.

### 2.1 Transport Layer Security (TLS)

- **Implementation**: Wrap the TCP listener with `crypto/tls`.
- **Configuration**:
  - `tls_enabled`: true/false
  - `tls_cert_file`: path to .crt
  - `tls_key_file`: path to .key
  - `tls_ca_file`: (Optional) CA for mTLS (Mutual TLS).
- **Client**: `go-bundoc` SDK must support `tls.Config`.

### 2.2 IP Whitelisting (Optional)

- **Config**: List of allowed CIDR ranges (e.g., `10.0.0.0/8`).
- **Enforcement**: Reject connections from unknown IPs at `Accept()` level.

---

## 3. Data Security (Encryption at Rest)

**Goal**: Protect data if the physical disk is stolen or compromised.

### 3.1 Transparent Data Encryption (TDE)

- **Scope**: Encrypt Page content before writing to Disk (`pager.go`).
- **Algorithm**: AES-256-GCM (Authenticated Encryption).
- **Key Management**:
  - **Master Key**: Loaded from environment variable or external KMS (Key Management Service).
  - **Table Keys**: Each table/collection could have its own DEK (Data Encryption Key) wrapped by the Master Key.
- **Process**:
  - **Write**: `BufferPool` -> `Encrypt(Page)` -> `Pager.Write()`.
  - **Read**: `Pager.Read()` -> `Decrypt(Page)` -> `BufferPool`.
- **Performance**: AES-NI hardware acceleration usage is critical.

---

## 4. Auditing & Logging

**Goal**: Track suspicious activities and compliance.

### 4.1 Audit Log

- **Events**:
  - Login Success/Failure.
  - User Creation/Deletion.
  - Privilege Escalation.
  - Sensitive Operations (Drop Database, Drop Collection).
- **Format**: JSON structured logs.
- **Destination**: File (`audit.json`), Syslog, or Console.

---

## 5. Implementation Roadmap

### Phase 1: Foundation (AuthN/AuthZ)

1. Define `User` and `Role` structs.
2. Implement `admin.users` collection bootstrapping.
3. Implement `SCRAM` handshake in Wire Protocol (`OpAuth`).
4. Add Permission Check middleware.

### Phase 2: Network (TLS)

1. Add TLS config to `Server` struct.
2. Update `Listener` to use `tls.Listen`.
3. Update `Client` to use `tls.Dial`.

### Phase 3: Encryption (At Rest)

1. Create `CryptoPager` wrapper around `Pager`.
2. Implement AES-GCM Encryption/Decryption hooks.
3. Add Key Loading logic.

---

## 6. Configuration Schema

---

## 7. Platform Integration & Multi-Tenancy Strategy

**Context**: Bundoc will be embedded or accessed by a multi-tenant platform. Security must ensure strictly isolated environments.

### 7.1 Data Isolation Model

- **Database-per-Tenant**: Recommended approach. Each platform tenant gets a dedicated Bundoc database (e.g., `tenant_123`, `tenant_456`).
- **Role Assignment**: Platform creates a `dbOwner` user for each tenant restricted _only_ to their specific database.
  - User `app_tenant_123` -> Roles: `[{Role: "dbOwner", DB: "tenant_123"}]`
  - Prevents cross-tenant data leaks even if application logic fails.

### 7.2 Resource Quotas (DoS Protection)

- **Connection Limits**: Enforce `max_connections` per user to prevent one tenant from exhausting the connection pool.
- **Execution Limits**:
  - `max_execution_time`: Abort queries running longer than X ms.
  - `max_scan_docs`: Limit number of documents scanned per query to prevent "stop-the-world" table scans on large collections.

### 7.3 Secure Default Configuration

- **Bind Address**: Bind to `127.0.0.1` or internal VPC IP by default, never `0.0.0.0` unless explicitly configured.
- **TLS Enforcement**: Fail startup if TLS is requested but certificates are missing. Strict mode for production.
