# DocDB Scaling Matrix Testing

This document describes how to run comprehensive scaling matrix tests to understand DocDB's performance characteristics across different configurations.

## Overview

The scaling matrix tests vary three key parameters:

- **Number of databases** (1, 3, 6, 12)
- **Connections per database** (1, 5, 10, 20)
- **Workers per database** (1, 2, 5, 10)

This creates a comprehensive test matrix to understand how DocDB scales under different configurations.

## Prerequisites

1. DocDB server running and accessible via Unix socket
2. Go 1.21 or later
3. Sufficient disk space for test results

## Quick Start

### 1. Run Full Matrix

```bash
# From project root
go run ./docdb/tests/load/cmd/matrix_runner/main.go \
  -databases "1,3,6,12" \
  -connections "1,5,10,20" \
  -workers "1,2,5,10" \
  -duration 5m \
  -socket /tmp/docdb.sock \
  -wal-dir ./docdb/data/wal \
  -output-dir ./matrix_results
```

### 2. Run Subset of Tests

```bash
# Test only baseline configurations
go run ./docdb/tests/load/cmd/matrix_runner/main.go \
  -databases "1,3,6" \
  -connections "1" \
  -workers "1" \
  -duration 5m \
  -socket /tmp/docdb.sock \
  -output-dir ./matrix_results
```

### 3. Analyze Results

```bash
go run ./docdb/tests/load/cmd/analyze_matrix/main.go \
  -results-dir ./matrix_results \
  -output ./matrix_results/reports/analysis.md
```

## Command-Line Options

### Matrix Runner (`matrix_runner`)

```
-databases string
    Comma-separated list of database counts (default: "1,3,6,12")

-connections string
    Comma-separated list of connections per DB (default: "1,5,10,20")

-workers string
    Comma-separated list of workers per DB (default: "1,2,5,10")

-duration duration
    Duration per test configuration (default: 5m)

-socket string
    IPC socket path (default: "/tmp/docdb.sock")

-wal-dir string
    WAL directory path (default: "./docdb/data/wal")

-output-dir string
    Output directory for results (default: "./matrix_results")

-doc-size int
    Document size in bytes (default: 1024)

-doc-count int
    Documents per database (default: 10000)

-read-percent int
    Read operation percentage (default: 40)

-write-percent int
    Write operation percentage (default: 30)

-update-percent int
    Update operation percentage (default: 20)

-delete-percent int
    Delete operation percentage (default: 10)

-csv
    Generate CSV output files (default: true)

-seed int
    Random seed (0 = use timestamp)

-restart-db
    Restart DocDB server between tests (not implemented)
```

### Analysis Tool (`analyze_matrix`)

```
-results-dir string
    Matrix results base directory (default: "./matrix_results") containing:
      - json/*.json
      - csv_global/*.csv
      - csv_dbs/<dbName>/latency_summary.csv

-output string
    Output path for analysis report (default: "./matrix_results/reports/analysis.md")
```

## Test Execution Order

The matrix runner executes tests in the following order:

1. **Baseline Tests** (1 DB scenarios)
   - 1db_1conn_1w (baseline)
   - 1db_1conn_2w, 1db_1conn_5w, 1db_1conn_10w
   - 1db_5conn_1w, 1db_10conn_1w, 1db_20conn_1w
   - 1db_5conn_5w, 1db_10conn_10w

2. **Multi-DB Tests** (horizontal scaling)
   - 3db_1conn_1w
   - 6db_1conn_1w
   - 12db_1conn_1w

3. **Combined Tests** (multi-DB + multi-connection)
   - 3db_5conn_1w, 3db_10conn_1w
   - 6db_5conn_1w, 6db_10conn_1w

4. **Stress Tests** (find limits)
   - 12db_20conn_10w (maximum configuration)

## Output Files

Each test configuration generates:

- `{config_name}.json` - Full test results
- `{config_name}_*.csv` - CSV files (if -csv enabled)

The matrix runner also generates:

- `summary.txt` - Test execution summary

The analysis tool generates:

- `analysis.md` - Comprehensive analysis report

## Understanding Results

### Key Metrics

- **Throughput (ops/sec)**: Total operations per second across all databases
- **P95 Latency (ms)**: 95th percentile latency across all operations
- **P99 Latency (ms)**: 99th percentile latency across all operations

### Analysis Categories

The analysis report groups results by:

1. **Connection Scaling**: How multiple connections per DB affect performance
2. **Worker Scaling**: How multiple workers per DB affect performance
3. **Database Scaling**: How multiple databases affect performance

## Example Workflow

```bash
# 1. Start DocDB server
./docdb/docdb --data-dir ./docdb/data --socket /tmp/docdb.sock

# 2. Run matrix tests (in another terminal)
go run ./docdb/tests/load/cmd/matrix_runner/main.go \
  -databases "1,3,6" \
  -connections "1,5" \
  -workers "1,2" \
  -duration 2m \
  -output-dir ./matrix_results

# 3. Analyze results
go run ./docdb/tests/load/cmd/analyze_matrix/main.go \
  -results-dir ./matrix_results

# 4. Review analysis
cat ./matrix_results/analysis.md
```

## Tips

1. **Start Small**: Run a subset first (e.g., 1-3 DBs, 1 connection, 1 worker) to verify setup
2. **Monitor Resources**: Watch CPU, memory, and disk I/O during tests
3. **Consistent Environment**: Use the same hardware and DocDB configuration for all tests
4. **Restart Between Major Changes**: Restart DocDB server between major configuration changes
5. **Check Logs**: Monitor DocDB server logs for errors or warnings

## Troubleshooting

### Tests Fail to Start

- Verify DocDB server is running: `ls -l /tmp/docdb.sock`
- Check socket path matches configuration
- Verify WAL directory exists and is accessible

### Low Throughput

- Check disk I/O performance
- Verify DocDB server has sufficient resources
- Check for errors in DocDB server logs

### Connection Errors

- Ensure DocDB server is not overloaded
- Check `MaxConnections` setting in DocDB config
- Verify socket permissions

## Next Steps

After running the matrix tests:

1. Review `analysis.md` for key findings
2. Identify optimal configurations
3. Use findings to guide architecture decisions
4. Document scaling recommendations
