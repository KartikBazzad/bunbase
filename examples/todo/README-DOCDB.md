# DocDB Load Testing

This directory contains load tests that can run against both SQLite (via BunBase) and DocDB backends to compare performance.

## Running DocDB Load Tests

### Prerequisites

1. Build DocDB:
```bash
cd ../../docdb
zig build
```

2. Start DocDB server:
```bash
./zig-out/bin/docdb --socket /tmp/docdb.sock --data-dir ./docdb-data
```

### Run Tests

```bash
# Run with default CRUD scenario
bun load-test-docdb.ts

# Run with specific scenario
bun load-test-docdb.ts --scenario=crud

# Run with specific profile
bun load-test-docdb.ts --profile=heavy

# Export results
bun load-test-docdb.ts --scenario=crud --export=json
```

## Comparing SQLite vs DocDB

### SQLite (via BunBase)
```bash
bun load-test.ts --scenario=crud
```

### DocDB
```bash
bun load-test-docdb.ts --scenario=crud
```

## Expected Results

### SQLite Behavior
- **P95/P99 latency grows unbounded** under concurrent writes
- Single-writer bottleneck causes queue buildup
- Latency increases linearly with concurrent users

### DocDB Behavior
- **P95/P99 latency remains bounded** under concurrent writes
- Fair scheduling prevents starvation
- Latency stays consistent regardless of concurrent users

## Key Metrics

- **P95 Latency**: Should stay < 100ms for DocDB
- **P99 Latency**: Should remain bounded (no unbounded growth)
- **Success Rate**: Should remain high (> 99%)
- **Throughput**: Should scale with concurrent users

## Environment Variables

- `DOCDB_SOCKET`: Unix socket path (default: `/tmp/docdb.sock`)
- `DOCDB_DATA_DIR`: Data directory (default: `./docdb-data`)
