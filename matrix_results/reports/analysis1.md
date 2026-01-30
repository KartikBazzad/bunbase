# DocDB Scaling Matrix Analysis

## Summary

Total Tests: 6

## Results by Category

### Connection Scaling (1 DB, 1 Worker)

| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-------------|----------------------|------------------|------------------|
| 100 | 515.16 | 5.19 | 6.98 |
| 50 | 532.65 | 5.08 | 6.76 |

### Worker Scaling (1 DB, 1 Connection)

| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|---------|----------------------|------------------|------------------|

### Database Scaling (1 Connection, 1 Worker)

| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |
|-----------|----------------------|------------------|------------------|

## All Results

| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |
|------|-----|---------|------|------------|-----|-----|
| 10db_100conn_1w | 10 | 100 | 1 | 543.28 | 47.41 | 72.64 |
| 10db_50conn_1w | 10 | 50 | 1 | 458.60 | 58.13 | 85.77 |
| 1db_100conn_1w | 1 | 100 | 1 | 515.16 | 5.19 | 6.98 |
| 1db_50conn_1w | 1 | 50 | 1 | 532.65 | 5.08 | 6.76 |
| 20db_100conn_1w | 20 | 100 | 1 | 498.96 | 108.89 | 163.87 |
| 20db_50conn_1w | 20 | 50 | 1 | 528.41 | 104.22 | 155.62 |

---

## A. Interpretation (scheduler / UnsafeMultiWriter)

These results were taken with the **default single scheduler worker** (UnsafeMultiWriter false, WorkerCount=1). That one worker serializes all requests globally: it picks one request from a per-DB queue, runs handleRequest (OpenDB + Execute), then picks the next. So only one request is in flight for the entire server.

- **Throughput cap (~515–533 ops/sec):** Matches a single worker processing one request at a time. Adding connections (50 vs 100) or DBs (1 vs 10 vs 20) does not increase throughput because the same worker still processes everything.
- **Latency growth with DB count:** With more DBs, the single worker round-robins across more queues. Each DB gets less service time, so requests wait longer in queue → P95/P99 rise (1 DB ~5 ms, 10 DBs ~47–58 ms, 20 DBs ~104–109 ms).
- **Next step:** Start DocDB with **UnsafeMultiWriter: true** and **WorkerCount** / **MaxWorkers** &gt; 1 (e.g. 4 and 16). Re-run the same matrix; you should see higher throughput (especially with 10/20 DBs) and lower latency growth, since multiple requests can be in flight and OpenDB is now safe for concurrent callers (dbsMu).

---

## B. Matrix flags to fill Worker scaling and Database scaling

To populate the **Worker Scaling** and **Database Scaling** tables in the report, add these dimensions to the matrix runner:

**Worker scaling (1 DB, 1 connection, varying workers):**
- Include `-workers "1,2,5,10"` (or a subset like `"1,2,5"`) and **one** connection and **one** database, e.g.:
  - `-databases "1" -connections "1" -workers "1,2,5,10"`

**Database scaling (1 connection, 1 worker, varying DBs):**
- Include multiple DBs with **one** connection and **one** worker, e.g.:
  - `-databases "1,3,6,10,20" -connections "1" -workers "1"`

**Example full matrix that fills all three categories:**
```bash
go run ./docdb/tests/load/cmd/matrix_runner/main.go \
  -databases "1,3,6,10,20" \
  -connections "1,5,10,50,100" \
  -workers "1,2,5,10" \
  -duration 2m \
  -socket /tmp/docdb.sock \
  -output-dir ./matrix_results
```
Then re-run the analyzer; Connection scaling, Worker scaling, and Database scaling will all have rows (subset above keeps the run size manageable; adjust duration/dims as needed).

---

## C. One-paragraph Results section (paste into doc)

**Results:** With a single scheduler worker (default), DocDB matrix runs cap at ~515–533 ops/sec regardless of connections per DB (50 vs 100) or number of databases (1, 10, 20). Latency stays low for one DB (P95 ~5 ms) but grows sharply with DB count: 10 DBs reach P95 ~47–58 ms and 20 DBs ~104–109 ms, as the single worker spreads time across more queues. Enabling multiple scheduler workers (UnsafeMultiWriter: true and WorkerCount/MaxWorkers &gt; 1) is expected to raise throughput and reduce latency growth when re-running the same matrix; filling Worker scaling and Database scaling requires adding workers (e.g. 1,2,5,10) and DB counts (e.g. 1,3,6,10,20) with 1 conn and 1 worker in the matrix config.
