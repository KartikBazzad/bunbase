# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 9

Baseline (1db_1conn_1w):
- Throughput: 369.56 ops/sec
- P95 Latency: 8.35 ms
- P99 Latency: 22.71 ms

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 1 | 369.56 | 8.35 | 22.71 |
| 20 | 372.44 | 8.19 | 21.67 |
| 50 | 361.95 | 8.75 | 22.07 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|
| 1 | 369.56 | 8.35 | 22.71 |

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|
| 10 | 850.41 | 28.37 | 37.47 |
| 1 | 369.56 | 8.35 | 22.71 |
| 20 | 706.04 | 72.16 | 104.79 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_1conn_1w | 10 | 1 | 1 | 850.41 | 28.37 | 37.47 |
| 10db_20conn_1w | 10 | 20 | 1 | 765.69 | 33.12 | 46.47 |
| 10db_50conn_1w | 10 | 50 | 1 | 871.83 | 28.31 | 38.08 |
| 1db_1conn_1w | 1 | 1 | 1 | 369.56 | 8.35 | 22.71 |
| 1db_20conn_1w | 1 | 20 | 1 | 372.44 | 8.19 | 21.67 |
| 1db_50conn_1w | 1 | 50 | 1 | 361.95 | 8.75 | 22.07 |
| 20db_1conn_1w | 20 | 1 | 1 | 706.04 | 72.16 | 104.79 |
| 20db_20conn_1w | 20 | 20 | 1 | 839.41 | 59.36 | 87.09 |
| 20db_50conn_1w | 20 | 50 | 1 | 804.65 | 61.93 | 90.26 |
