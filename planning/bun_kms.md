# BunKMS Implementation Plan

**Objective**: Build a lightweight, secure Key Management Service (KMS) to manage cryptographic keys for the BunBase ecosystem (Bundoc, Buncast, etc.).

## 1. Overview

BunKMS will serve as the "Root of Trust" for the platform. It will handle:

- **Key Generation**: Creating cryptographically strong keys (AES-256, RSA, ECDSA).
- **Key Storage**: Securely storing keys, encrypted by a Master Key (Master Key provided via ENV or Hardware Token).
- **Crypto Operations**: Encrypt/Decrypt operations (Envelope Encryption) so applications don't handle raw keys.
- **Secret Management**: Storing API keys and database credentials.

## 2. Architecture

### 2.1 Tech Stack

- **Language**: Go
- **Protocol**: gRPC (internal) + REST Gateway (external)
- **Storage**:
  - MVP: Encrypted BoltDB (local file) or SQLite.
  - Future: HA Etcd or Bundoc (careful with circular dependency).

### 2.2 Core Components

1.  **Vault**: The core engine responsible for crypto operations.
2.  **Store**: Persistent layer for wrapped keys.
3.  **API**: Interface for clients (e.g., Bundoc Server).
4.  **Audit**: Immutable log of all key access.

### 2.3 Master Key (Root of Trust)

The KMS needs a key to encrypt its database. This is the **Master Unseal Key**.

- **Dev Mode**: Stored in `BUNKMS_MASTER_KEY` env var.
- **Prod Mode**: Shamir's Secret Sharing (like HashiCorp Vault) or Cloud KMS wrap.

## 3. API Design

### 3.1 Key Management

- `POST /v1/keys`: Create a new named key (e.g., "bundoc-tde-key").
- `GET /v1/keys/{name}`: Get key metadata (not material).
- `POST /v1/keys/{name}/rotate`: Rotate the key version.

### 3.2 Cryptographic Operations

- `POST /v1/encrypt/{key_name}`: Encrypt plaintext. Returns `ciphertext` (Key Version + Nonce + Data).
- `POST /v1/decrypt/{key_name}`: Decrypt ciphertext.

### 3.3 Secrets

- `POST /v1/secrets`: Store a secret.
- `GET /v1/secrets/{name}`: Retrieve a secret.

## 4. Integration with Bundoc

Bundoc currently takes a raw `EncryptionKey`. We will update it to support a KMS Provider:

1.  Bundoc requests a Data Encryption Key (DEK) from BunKMS on startup.
2.  BunKMS returns the DEK (plaintext) and the Encrypted DEK (to store in `system_catalog.json`).
3.  Bundoc uses the DEK for TDE.

## 5. Implementation Stages

### Phase 1: Skeleton & API

- Initialize `bun-kms` Go module.
- Define API structs and Routes.
- Implement In-Memory Vault.

### Phase 2: Persistence & Security

- Implement BoltDB storage with AES-GCM encryption at rest.
- Implement Master Key loading.

### Phase 3: Bundoc Integration

- Update Bundoc to fetch key from BunKMS (optional configuration).

### Phase 4: UI & CLI

- CLI tool (`bun-kms-cli`) to manage keys.

## 6. Directory Structure

```
bun-kms/
├── cmd/
│   └── server/       # Main entry point
├── internal/
│   ├── api/          # HTTP/gRPC handlers
│   ├── core/         # Domain logic
│   ├── storage/      # BoltDB/SQLite
│   └── crypto/       # Low-level crypto wrappers
├── go.mod
└── README.md
```
