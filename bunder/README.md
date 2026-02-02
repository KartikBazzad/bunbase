# Bunder - Redis-like KV Database

**Bunder** (Bun + Thunder) is an extremely fast, Redis-like key-value database built in Go, integrated with the BunBase ecosystem.

## Features

- **Redis-like API**: GET, SET, DEL, TTL, KEYS, LPUSH, RPUSH, SADD, HSET, and more
- **Persistence**: WAL, RDB-like snapshots, optional AOF
- **High concurrency**: Sharded maps (256 shards), fine-grained locking
- **TTL**: Time-to-live with efficient timing wheel
- **Pub/Sub**: Integration with Buncast for real-time events
- **Protocols**: RESP2 (Redis), HTTP API, SSE placeholder, Prometheus metrics

## Quick Start

```bash
# Run server (TCP :6379, HTTP :8080)
go run ./cmd/server

# Or with flags
go run ./cmd/server -addr :6379 -http :8080 -data ./data

# Connect with redis-cli
redis-cli -p 6379
> SET foo bar
> GET foo
"bar"
> LPUSH mylist a b c
> LRANGE mylist 0 -1
> SADD myset x y
> SMEMBERS myset
> HSET myhash f1 v1
> HGETALL myhash
```

### CLI REPL

```bash
go run ./cmd/cli
# or
go run ./cmd/cli 127.0.0.1:6379
bunder> SET hello world
bunder> GET hello
world
bunder> QUIT
```

### Go Client

```go
package main

import (
    "context"
    "fmt"
    "github.com/kartikbazzad/bunbase/bunder/pkg/client"
)

func main() {
    ctx := context.Background()
    c, err := client.Connect(ctx, client.DefaultOptions("127.0.0.1:6379"))
    if err != nil {
        panic(err)
    }
    defer c.Close()
    if err := c.Set(ctx, "foo", []byte("bar")); err != nil {
        panic(err)
    }
    val, err := c.Get(ctx, "foo")
    if err != nil {
        panic(err)
    }
    fmt.Println(string(val)) // bar
}
```

### HTTP API

- `GET /health` - health check
- `GET /metrics` - Prometheus metrics
- `GET /kv/:key` - get value
- `PUT /kv/:key` - set value (body = value)
- `DELETE /kv/:key` - delete key
- `GET /keys?pattern=*` - list keys
- `GET /subscribe` - SSE (placeholder)

## Supported Commands (RESP)

| Command               | Description            |
| --------------------- | ---------------------- |
| GET key               | Get value              |
| SET key value         | Set value              |
| DEL key               | Delete key             |
| EXISTS key            | Check existence        |
| KEYS pattern          | List keys (\* and ?)   |
| TTL key               | Time to live (seconds) |
| EXPIRE key seconds    | Set TTL                |
| LPUSH key v1 v2 ...   | Prepend to list        |
| RPUSH key v1 v2 ...   | Append to list         |
| LPOP key              | Remove first element   |
| RPOP key              | Remove last element    |
| LRANGE key start stop | Get range              |
| LLEN key              | List length            |
| SADD key m1 m2 ...    | Add to set             |
| SREM key m1 m2 ...    | Remove from set        |
| SMEMBERS key          | All set members        |
| SISMEMBER key m       | Check membership       |
| SCARD key             | Set size               |
| HSET key field value  | Set hash field         |
| HGET key field        | Get hash field         |
| HGETALL key           | All fields and values  |
| HDEL key f1 f2 ...    | Delete hash fields     |
| HEXISTS key field     | Check field exists     |
| HLEN key              | Hash size              |
| PING [msg]            | Ping                   |
| QUIT                  | Close connection       |

## Configuration

| Flag         | Default | Description              |
| ------------ | ------- | ------------------------ |
| -data        | ./data  | Database directory       |
| -addr        | :6379   | TCP listen address       |
| -http        | :8080   | HTTP API address         |
| -buffer-pool | 10000   | Buffer pool size (pages) |
| -shards      | 256     | Number of shards         |
| -buncast     | false   | Enable Buncast pub/sub   |

## Architecture

- **Storage**: 4KB pages, SLRU buffer pool, B+Tree index, freelist
- **Persistence**: Write-Ahead Log (WAL), RDB-like snapshots, AOF
- **Concurrency**: 256 sharded maps, page latches for B+Tree
- **TTL**: Timing wheel with configurable check interval
- **Pub/Sub**: Buncast client for topic `bunder.operations`

See [docs/architecture.md](docs/architecture.md), [docs/api-reference.md](docs/api-reference.md), and [docs/implementation.md](docs/implementation.md) for implementation details.

## Running Tests

```bash
go test ./... -count=1
go test ./... -race -short
```

## Load Tests

Run load tests against a live Bunder server (start the server first in another terminal):

```bash
# Start server
go run ./cmd/server -addr 127.0.0.1:6379

# Run load test: 50 clients, 10s, mixed GET/SET
go run ./cmd/loadtest -addr 127.0.0.1:6379 -duration 10s -clients 50 -workload mixed

# Set-only, 5s, 20 clients
go run ./cmd/loadtest -addr 127.0.0.1:6379 -duration 5s -clients 20 -workload set

# Get-only (pre-populate keys first via mixed or set)
go run ./cmd/loadtest -addr 127.0.0.1:6379 -duration 5s -clients 20 -workload get
```

In-process load tests (no server needed) run with the package tests:

```bash
go test ./internal/loadtest/... -v -count=1
```

Benchmark (1s mixed workload, 10 clients):

```bash
go test ./internal/loadtest/... -bench=BenchmarkLoadTest_Mixed -benchtime=1x
```

## License

MIT
