# BunKMS Architecture

## Overview

BunKMS is the centralized Key Management Service for the BunBase ecosystem. It provides key generation, encryption/decryption (envelope encryption), secret storage, and signing/verification.

## Components

- **API** (`internal/api`): HTTP JSON API with optional JWT auth, validation, and metrics.
- **Core** (`internal/core`): Vault (key storage and crypto operations), SecretStore (encrypted secrets), codec (ciphertext encoding), RSA/ECDSA signing.
- **Storage** (`internal/storage`): Pluggable store interface; Bunder implementation with AES-GCM encryption at rest.
- **Audit** (`internal/audit`): Append-only audit log for key and secret operations.
- **Auth** (`internal/auth`): JWT middleware and role-based access (admin, operator, reader).
- **Health** (`internal/health`): Health and readiness endpoints.
- **Metrics** (`internal/metrics`): Prometheus request/operation metrics.

## Data Flow

1. **Keys**: CreateKey/RotateKey/RevokeKey update in-memory vault and persist to Bunder (encrypted). Encrypt/Decrypt use latest key version from memory.
2. **Secrets**: Put/Get encrypt/decrypt with master key and persist to Bunder (encrypted again by storage layer).
3. **Audit**: Each operation logs an event (operation, resource, success) to the audit log file when configured.

## Storage

- **In-memory**: Default when `BUNKMS_DATA_PATH` is unset. No persistence.
- **Bunder**: When `BUNKMS_DATA_PATH` is set, an embedded Bunder KVStore is used. All values are encrypted with the master key before write. Keys are namespaced (`kms:key:<name>`, `kms:secret:<name>`).

## Security Model

- Master key (32 bytes) encrypts data at rest and secrets. Loaded via `BUNKMS_MASTER_KEY` (env).
- Optional JWT auth: when `BUNKMS_JWT_SECRET` is set, all `/v1/*` requests require `Authorization: Bearer <token>`.
- Input validation: key/secret names, payload sizes, base64 decoding.
