# Multi-Database Load Test Analysis (Run 1)

**Source**: `multidb_results-1.json`  
**Note**: This run was executed **before** WAL replay optimizations (single-pass replay + WriteNoSync). It includes WAL group commit and scheduler fairness.

---

## Executive Summary

| Metric               | Value                   |
| -------------------- | ----------------------- |
| **Duration**         | 300 seconds (5 minutes) |
| **Total Operations** | 180,857                 |
| **Throughput**       | **~603 ops/sec**        |
| **Databases**        | 3 (db1, db2, db3)       |
| **Workers**          | 10 per DB (30 total)    |

- **P95 latency**: 115–135 ms across operations (global).
- **WAL growth**: ~47 MB total; ~52 KB/s per DB.
- **Healing**: 0 events; no overhead.

---

## Test Configuration

| Parameter      | Value            |
| -------------- | ---------------- |
| Duration       | 300 s            |
| Workers per DB | 10               |
| Read %         | 40               |
| Create %       | 30               |
| Update %       | 20               |
| Delete %       | 10               |
| Document size  | 1024 bytes       |
| Document count | 10,000 per DB    |
| Socket         | /tmp/docdb.sock  |
| WAL directory  | ./docdb/data/wal |
| CSV output     | Yes              |

---

## Throughput

### Per-Database Operations

| Database  | Operations  | Ops/sec  | Create     | Read       | Update     | Delete     |
| --------- | ----------- | -------- | ---------- | ---------- | ---------- | ---------- |
| db1       | 60,994      | ~203     | 18,332     | 24,378     | 12,140     | 6,144      |
| db2       | 58,973      | ~197     | 17,673     | 23,497     | 11,958     | 5,845      |
| db3       | 60,915      | ~203     | 18,218     | 24,444     | 12,028     | 6,225      |
| **Total** | **180,882** | **~603** | **54,223** | **72,319** | **36,126** | **18,214** |

**Analysis**: Load is well balanced across the three databases (~60k ops each). Scheduler fairness (queue-depth–aware picking) is in effect.

### Operation Distribution (Global)

| Operation | Count  | %     | Target % | Status |
| --------- | ------ | ----- | -------- | ------ |
| Read      | 72,319 | 40.0% | 40%      | ✅     |
| Create    | 54,223 | 30.0% | 30%      | ✅     |
| Update    | 36,126 | 20.0% | 20%      | ✅     |
| Delete    | 18,214 | 10.1% | 10%      | ✅     |

CRUD mix matches the configured workload.

---

## Latency

### Per-Database P95 (ms)

| Database    | Create    | Read      | Update    | Delete    |
| ----------- | --------- | --------- | --------- | --------- |
| db1         | 123.3     | 115.7     | 133.7     | 115.6     |
| db2         | 125.6     | 117.2     | 135.3     | 121.0     |
| db3         | 123.0     | 114.6     | 135.6     | 116.9     |
| **Average** | **124.0** | **115.8** | **134.9** | **117.8** |

**Analysis**: P95 is consistent across DBs (within ~6 ms). Reads are fastest; updates are slowest, as expected for write path (WAL + datafile).

### Global Aggregated Latency

| Operation | P50 (ms) | P95 (ms) | P99 (ms) | P999 (ms) | Min (ms) | Max (ms) | Count  |
| --------- | -------- | -------- | -------- | --------- | -------- | -------- | ------ |
| Create    | 51.1     | 124.0    | 167.5    | 228.1     | 0.044    | 323.5    | 54,223 |
| Read      | 46.8     | 115.8    | 158.1    | 218.7     | 0.021    | 313.5    | 72,319 |
| Update    | 60.5     | 134.9    | 179.6    | 240.6     | 0.048    | 305.6    | 36,126 |
| Delete    | 49.8     | 117.8    | 158.5    | 214.4     | 0.022    | 313.3    | 18,214 |

**Analysis**:

- P95 in the 115–135 ms range indicates group commit and scheduler are effective.
- P99 157–180 ms; P999 214–241 ms; tail latencies are bounded.
- Min/Max and counts are populated; global aggregation is correct.

---

## WAL Growth

### Per-Database WAL

| Database  | Initial (MB) | Final (MB) | Growth (MB) | Rate (KB/s) |
| --------- | ------------ | ---------- | ----------- | ----------- |
| db1       | 20.2         | 35.1       | 14.9        | 52.2        |
| db2       | 19.9         | 34.9       | 15.0        | 52.1        |
| db3       | 20.2         | 35.1       | 14.9        | 52.2        |
| **Total** | **60.2**     | **105.1**  | **44.8**    | **~156**    |

(Reported total in JSON: **TotalWALGrowth** 46,962,590 bytes ≈ 44.8 MB.)

**Analysis**:

- WAL growth tracking is consistent across DBs.
- ~52 KB/s per DB aligns with write-heavy mix (30% create + 20% update).
- Sample count 302 over ~300 s indicates ~1 sample/sec.

---

## Healing

| Metric           | Value |
| ---------------- | ----- |
| Total healings   | 0     |
| Documents healed | 0     |
| Healing time (s) | 0     |
| Overhead %       | 0     |

No corruption detected; healing did not run.

---

## Conclusions

1. **Throughput**: ~603 ops/sec over 3 DBs is strong for this workload and reflects WAL group commit and scheduler fairness.
2. **Latency**: Global P95 115–135 ms and P99 &lt; 180 ms are good; no DB is starved.
3. **Balance**: Per-DB ops and latency are even; queue-depth–aware scheduling is effective.
4. **WAL**: Growth and rates are stable; tracking is correct.
5. **Healing**: Not triggered; no overhead.

**Context**: This run is **before** WAL replay optimizations (single-pass + WriteNoSync). Replay changes only affect **startup time** when reopening DBs with large WALs; they do not change steady-state throughput or latency. After building with replay optimizations, expect similar numbers here but much faster DB open on restart.
