# BunKMS API Reference

Base path: `/v1`. All responses are JSON. When JWT is enabled, send `Authorization: Bearer <token>`.

## Key Management

### Create key

`POST /v1/keys`

Body: `{"name":"<name>","type":"aes-256|rsa-2048|ecdsa-p256"}` (type optional, default aes-256)

Returns: Key metadata (name, type, created_at, latest_version, versions). 201 Created.

### Get key

`GET /v1/keys/<name>`

Returns: Key metadata. 200 OK or 404 Not Found.

### Rotate key

`POST /v1/keys/<name>/rotate`

Returns: Updated key metadata. 200 OK.

### Revoke key

`POST /v1/keys/<name>/revoke`

Returns: `{"status":"revoked","name":"<name>"}`. 200 OK. Revoked keys cannot be used for encrypt/decrypt/sign/verify.

## Encryption

### Encrypt

`POST /v1/encrypt/<key_name>`

Body: `{"plaintext":"<string>"}` or `{"plaintext_b64":"<base64>"}`, optional `"aad"` / `"aad_b64"`.

Returns: `{"ciphertext":"<base64>","version":<n>,"nonce_b64":"...","data_b64":"..."}`. 200 OK. Key must be aes-256.

### Decrypt

`POST /v1/decrypt/<key_name>`

Body: `{"ciphertext":"<base64>","aad":"..."}` (aad optional).

Returns: `{"plaintext":"<string>"}` and/or `{"plaintext_b64":"<base64>"}`. 200 OK.

## Signing (RSA / ECDSA)

### Sign

`POST /v1/keys/<name>/sign`

Body: `{"digest_b64":"<base64>"}` (32-byte SHA-256 digest).

Returns: `{"signature_b64":"<base64>"}`. 200 OK. Key must be rsa-2048 or ecdsa-p256.

### Verify

`POST /v1/keys/<name>/verify`

Body: `{"digest_b64":"<base64>","signature_b64":"<base64>"}`.

Returns: `{"valid":true|false}`. 200 OK.

## Secrets

### Put secret

`POST /v1/secrets`

Body: `{"name":"<name>","value":"<string>"}` or `{"name":"<name>","value_b64":"<base64>"}`.

Returns: `{"name":"<name>","created_at":"..."}`. 201 Created.

### Get secret

`GET /v1/secrets/<name>`

Returns: `{"name":"<name>","created_at":"...","value":"<string>"}` and/or `"value_b64"`. 200 OK or 404.

## Health and Metrics

- `GET /health` – Health check (master key, storage). 200 or 503.
- `GET /ready` – Readiness (storage). 200 or 503.
- `GET /metrics` – Prometheus metrics.

## Errors

Errors: `{"error":"<message>"}` with appropriate HTTP status (400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 500 Internal Server Error).
