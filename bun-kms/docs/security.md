# BunKMS Security

## Master Key

- The master key (32 bytes) encrypts all data at rest in Bunder and all secret values.
- Provide it via `BUNKMS_MASTER_KEY`. Supported formats: `base64:<b64>`, hex (64 chars), or raw 32-byte string.
- Do not commit the master key. Use a secrets manager or HSM in production.
- Rotating the master key requires re-encrypting stored data; not implemented in this version.

## Authentication

- When `BUNKMS_JWT_SECRET` is set, every `/v1/*` request must include `Authorization: Bearer <token>`.
- Tokens can carry a `role` claim: `admin`, `operator`, or `reader`. The middleware validates the token; role-based checks can be extended per endpoint.
- Use HTTPS in production so tokens are not sent in clear text.

## Audit Log

- Set `BUNKMS_AUDIT_LOG` to a file path to log key and secret operations (create, rotate, revoke, encrypt, decrypt, put, get).
- Log format: one JSON object per line (timestamp, operation, resource, success, message).
- Protect the audit log file (permissions, integrity). Do not log secret or key material.

## Input Validation

- Key and secret names: alphanumeric plus `._-`, max length 256.
- Payload size limit: 512 KiB for encrypt/decrypt and secret values to reduce DoS risk.
- Base64 inputs are validated; invalid input returns 400.

## Recommendations

- Run BunKMS in a private network; expose only to trusted services.
- Use short-lived JWTs and rotate `BUNKMS_JWT_SECRET` periodically.
- Back up the Bunder data directory and audit log; secure backups with the same care as the master key.
- Prefer RSA-2048 or ECDSA P-256 for signing; AES-256 for encryption.
