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
| 20 | 682.50 | 50.83 | 86.78 |

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 20db_1conn_1w | 20 | 1 | 1 | 682.50 | 50.83 | 86.78 |
| 20db_1conn_2w | 20 | 1 | 2 | 740.63 | 180.20 | 253.16 |
| 20db_20conn_1w | 20 | 20 | 1 | 694.39 | 73.00 | 113.54 |
| 20db_20conn_2w | 20 | 20 | 2 | 684.16 | 190.09 | 301.41 |
| 20db_50conn_1w | 20 | 50 | 1 | 725.39 | 80.87 | 109.59 |
| 20db_50conn_2w | 20 | 50 | 2 | 688.10 | 199.52 | 298.09 |
