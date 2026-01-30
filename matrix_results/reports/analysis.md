# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 6

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 100 | 523.83 | 5.19 | 6.96 |
| 50 | 522.16 | 4.77 | 6.59 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_100conn_1w | 10 | 100 | 1 | 418.99 | 59.47 | 87.90 |
| 10db_50conn_1w | 10 | 50 | 1 | 413.84 | 60.17 | 89.47 |
| 1db_100conn_1w | 1 | 100 | 1 | 523.83 | 5.19 | 6.96 |
| 1db_50conn_1w | 1 | 50 | 1 | 522.16 | 4.77 | 6.59 |
| 20db_100conn_1w | 20 | 100 | 1 | 420.52 | 130.52 | 197.91 |
| 20db_50conn_1w | 20 | 50 | 1 | 421.64 | 127.79 | 191.60 |
