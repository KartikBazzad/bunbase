# Bunder API Reference

## RESP (TCP) Commands

Commands are sent as RESP arrays of bulk strings. Example: `*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n`.

### Key-Value

- **GET key** – Returns the value or nil bulk string if not set.
- **SET key value** – Sets key to value. Returns `+OK\r\n`.
- **DEL key** – Deletes key. Returns `:1` if removed, `:0` if not present.
- **EXISTS key** – Returns `:1` or `:0`.
- **KEYS pattern** – Returns array of keys. Pattern supports `*` (any) and `?` (single char).
- **TTL key** – Returns remaining seconds, `-1` if no TTL, `-2` if key not found.
- **EXPIRE key seconds** – Sets TTL. Returns `:1`.

### List

- **LPUSH key v1 [v2 ...]** – Prepend elements. Returns new length.
- **RPUSH key v1 [v2 ...]** – Append elements. Returns new length.
- **LPOP key** – Remove and return first element. Nil if empty.
- **RPOP key** – Remove and return last element. Nil if empty.
- **LRANGE key start stop** – Return range (0-based; negative = from end).
- **LLEN key** – Return list length.

### Set

- **SADD key m1 [m2 ...]** – Add members. Returns count of new members added.
- **SREM key m1 [m2 ...]** – Remove members. Returns count removed.
- **SMEMBERS key** – Return all members.
- **SISMEMBER key member** – Return `:1` or `:0`.
- **SCARD key** – Return set size.

### Hash

- **HSET key field value** – Set field. Returns `:1` if new, `:0` if updated.
- **HGET key field** – Return value or nil.
- **HGETALL key** – Return array of field, value, field, value, ...
- **HDEL key f1 [f2 ...]** – Delete fields. Returns count removed.
- **HEXISTS key field** – Return `:1` or `:0`.
- **HLEN key** – Return number of fields.

### Connection

- **PING [message]** – Returns `+PONG\r\n` or the message as bulk string.
- **QUIT** – Returns `+OK\r\n`; server may close connection.

## HTTP API

- **GET /health** – JSON `{"status":"ok"}`.
- **GET /metrics** – Prometheus text format (counters and gauges).
- **GET /kv/:key** – Get value (200 + body, 404 if not found).
- **PUT /kv/:key** – Set value; body = raw bytes.
- **DELETE /kv/:key** – Delete key (200, or 404 if not found).
- **GET /keys?pattern=\*** – JSON array of key strings.
- **GET /subscribe** – SSE stream (placeholder).

## Go Client (pkg/client)

- **Connect(ctx, opts)** – Connect to server.
- **Get(ctx, key)** – Get value; nil if not set.
- **Set(ctx, key, value)** – Set value.
- **Delete(ctx, key)** – Delete; returns true if key was present.
- **Exists(ctx, key)** – Returns true if key exists.
- **Keys(ctx, pattern)** – Returns slice of keys.
- **Ping(ctx, msg)** – Ping; optional message.
- **Do(ctx, cmd, args...)** – Raw command.
- **Close()** – Close connection.

## Response Types (RESP)

- `+` Simple string (e.g. OK, PONG)
- `-` Error (e.g. ERR message)
- `:` Integer (e.g. count, TTL)
- `$` Bulk string (nil = `$-1\r\n`)
- `*` Array of RESP values
