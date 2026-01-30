# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 9

Baseline (1db_1conn_1w):
- Throughput: 17628.07 ops/sec
- P95 Latency: 0.07 ms
- P99 Latency: 0.09 ms

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 1 | 17628.07 | 0.07 | 0.09 |
| 20 | 17738.46 | 0.07 | 0.09 |
| 50 | 17723.60 | 0.07 | 0.09 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|
| 1 | 17628.07 | 0.07 | 0.09 |

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|
| 10 | 1043.82 | 24.92 | 32.45 |
| 1 | 17628.07 | 0.07 | 0.09 |
| 20 | 155623.76 | 0.21 | 0.29 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_1conn_1w | 10 | 1 | 1 | 1043.82 | 24.92 | 32.45 |
| 10db_20conn_1w | 10 | 20 | 1 | 962.69 | 27.00 | 35.06 |
| 10db_50conn_1w | 10 | 50 | 1 | 2174.36 | 19.77 | 27.28 |
| 1db_1conn_1w | 1 | 1 | 1 | 17628.07 | 0.07 | 0.09 |
| 1db_20conn_1w | 1 | 20 | 1 | 17738.46 | 0.07 | 0.09 |
| 1db_50conn_1w | 1 | 50 | 1 | 17723.60 | 0.07 | 0.09 |
| 20db_1conn_1w | 20 | 1 | 1 | 155623.76 | 0.21 | 0.29 |
| 20db_20conn_1w | 20 | 20 | 1 | 152443.41 | 0.22 | 0.29 |
| 20db_50conn_1w | 20 | 50 | 1 | 143147.85 | 0.23 | 0.31 |
