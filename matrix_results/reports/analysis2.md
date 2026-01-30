# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 9

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|
| 2 | 597.70 | 11.56 | 18.00 |

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_1conn_2w | 10 | 1 | 2 | 720.63 | 71.15 | 96.78 |
| 10db_20conn_2w | 10 | 20 | 2 | 746.28 | 69.92 | 94.15 |
| 10db_50conn_2w | 10 | 50 | 2 | 806.16 | 65.61 | 87.07 |
| 1db_1conn_2w | 1 | 1 | 2 | 597.70 | 11.56 | 18.00 |
| 1db_20conn_2w | 1 | 20 | 2 | 553.45 | 13.37 | 20.31 |
| 1db_50conn_2w | 1 | 50 | 2 | 675.77 | 11.21 | 16.96 |
| 20db_1conn_2w | 20 | 1 | 2 | 862.17 | 115.26 | 161.07 |
| 20db_20conn_2w | 20 | 20 | 2 | 979.00 | 103.22 | 139.09 |
| 20db_50conn_2w | 20 | 50 | 2 | 918.66 | 110.60 | 149.63 |
