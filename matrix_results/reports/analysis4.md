# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 5

Baseline (1db_1conn_1w):
- Throughput: 325.62 ops/sec
- P95 Latency: 9.34 ms
- P99 Latency: 23.09 ms

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 1 | 325.62 | 9.34 | 23.09 |
| 20 | 317.01 | 10.51 | 25.83 |
| 50 | 311.65 | 10.82 | 28.12 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|
| 1 | 325.62 | 9.34 | 23.09 |

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|
| 10 | 683.24 | 34.49 | 50.92 |
| 1 | 325.62 | 9.34 | 23.09 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_1conn_1w | 10 | 1 | 1 | 683.24 | 34.49 | 50.92 |
| 10db_20conn_1w | 10 | 20 | 1 | 716.47 | 35.11 | 52.44 |
| 1db_1conn_1w | 1 | 1 | 1 | 325.62 | 9.34 | 23.09 |
| 1db_20conn_1w | 1 | 20 | 1 | 317.01 | 10.51 | 25.83 |
| 1db_50conn_1w | 1 | 50 | 1 | 311.65 | 10.82 | 28.12 |
