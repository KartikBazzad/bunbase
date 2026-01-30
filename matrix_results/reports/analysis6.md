# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 9

Baseline (1db_1conn_1w):
- Throughput: 406.61 ops/sec
- P95 Latency: 5.73 ms
- P99 Latency: 8.38 ms

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 1 | 406.61 | 5.73 | 8.38 |
| 20 | 409.50 | 5.95 | 9.15 |
| 50 | 416.99 | 5.98 | 9.20 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|
| 1 | 406.61 | 5.73 | 8.38 |

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|
| 10 | 681.22 | 33.43 | 47.76 |
| 1 | 406.61 | 5.73 | 8.38 |
| 20 | 648.82 | 78.71 | 116.38 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_1conn_1w | 10 | 1 | 1 | 681.22 | 33.43 | 47.76 |
| 10db_20conn_1w | 10 | 20 | 1 | 626.99 | 39.14 | 54.78 |
| 10db_50conn_1w | 10 | 50 | 1 | 714.35 | 35.12 | 48.99 |
| 1db_1conn_1w | 1 | 1 | 1 | 406.61 | 5.73 | 8.38 |
| 1db_20conn_1w | 1 | 20 | 1 | 409.50 | 5.95 | 9.15 |
| 1db_50conn_1w | 1 | 50 | 1 | 416.99 | 5.98 | 9.20 |
| 20db_1conn_1w | 20 | 1 | 1 | 648.82 | 78.71 | 116.38 |
| 20db_20conn_1w | 20 | 20 | 1 | 726.76 | 70.79 | 102.22 |
| 20db_50conn_1w | 20 | 50 | 1 | 641.47 | 82.21 | 122.88 |
