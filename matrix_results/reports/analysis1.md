# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 6

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|
| 20 | 699.06 | 52.91 | 84.31 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 20db_1conn_1w | 20 | 1 | 1 | 699.06 | 52.91 | 84.31 |
| 20db_1conn_2w | 20 | 1 | 2 | 687.78 | 185.17 | 286.45 |
| 20db_20conn_1w | 20 | 20 | 1 | 759.07 | 63.75 | 91.52 |
| 20db_20conn_2w | 20 | 20 | 2 | 781.51 | 172.76 | 248.18 |
| 20db_50conn_1w | 20 | 50 | 1 | 723.25 | 72.98 | 158.04 |
| 20db_50conn_2w | 20 | 50 | 2 | 741.98 | 173.74 | 325.10 |
