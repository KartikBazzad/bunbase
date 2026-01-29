# DocDB Load Testing Suite

A comprehensive load testing framework for DocDB that measures critical performance metrics including P95/P99 latency, WAL growth rate, and healing overhead.

## Features

- **Latency Measurement**: Tracks P50, P95, P99, and P99.9 percentiles for all operations (Create, Read, Update, Delete)
- **WAL Growth Tracking**: Monitors WAL size growth including all segments over time
- **Healing Overhead**: Measures healing operation overhead and frequency
- **Configurable Workloads**: Support for read-heavy, write-heavy, mixed, and custom workload patterns
- **Multiple Output Formats**: JSON and CSV output for analysis and visualization

## Installation

The load test suite is part of the DocDB project. Ensure you have:

- Go 1.21 or later
- DocDB server running and accessible via Unix socket

## Usage

### Basic Usage

Run a load test with default configuration (5 minutes, 10 workers, mixed workload):

```bash
go run docdb/tests/load/cmd/loadtest/main.go \
  -socket /tmp/docdb.sock \
  -db-name loadtest \
  -wal-dir ./docdb/data/wal \
  -output results.json
```

**Important:** The `-wal-dir` path must match where DocDB actually stores WAL files. If you started DocDB with `--data-dir ./data`, the WAL directory will be `./data/wal` (relative to where DocDB was started). Use an absolute path or ensure the path matches your DocDB configuration.

### Command-Line Options

```
-duration duration
    Test duration (e.g., 5m, 1h). Default: 5m
    If 0, uses -operations instead.

-workers int
    Number of concurrent workers. Default: 10

-operations int
    Total operations to execute (0 = use duration). Default: 0

-read-percent int
    Percentage of read operations. Default: 40

-write-percent int
    Percentage of write operations. Default: 30

-update-percent int
    Percentage of update operations. Default: 20

-delete-percent int
    Percentage of delete operations. Default: 10

-doc-size int
    Document payload size in bytes. Default: 1024

-doc-count int
    Number of unique documents. Default: 10000

-metrics-interval duration
    How often to collect metrics. Default: 1s

-socket string
    IPC socket path. Default: /tmp/docdb.sock

-db-name string
    Database name. Default: loadtest

-wal-dir string
    WAL directory path. Default: /tmp/docdb/wal

-output string
    Output JSON file path. Default: loadtest_results.json

-csv
    Generate CSV output files. Default: false

-seed int
    Random seed for reproducibility. Default: current timestamp
```

### Workload Patterns

#### Read-Heavy Workload

```bash
go run docdb/tests/load/cmd/loadtest/main.go \
  -duration 10m \
  -workers 20 \
  -read-percent 80 \
  -write-percent 10 \
  -update-percent 10 \
  -delete-percent 0 \
  -socket /tmp/docdb.sock \
  -db-name readtest \
  -wal-dir /tmp/docdb/wal \
  -output read_heavy_results.json
```

#### Write-Heavy Workload

```bash
go run docdb/tests/load/cmd/loadtest/main.go \
  -duration 10m \
  -workers 20 \
  -read-percent 10 \
  -write-percent 80 \
  -update-percent 10 \
  -delete-percent 0 \
  -socket /tmp/docdb.sock \
  -db-name writetest \
  -wal-dir /tmp/docdb/wal \
  -output write_heavy_results.json
```

#### Mixed Workload with CSV Output

```bash
go run docdb/tests/load/cmd/loadtest/main.go \
  -duration 15m \
  -workers 15 \
  -read-percent 40 \
  -write-percent 30 \
  -update-percent 20 \
  -delete-percent 10 \
  -doc-size 2048 \
  -doc-count 50000 \
  -metrics-interval 500ms \
  -socket /tmp/docdb.sock \
  -db-name mixedtest \
  -wal-dir /tmp/docdb/wal \
  -output mixed_results.json \
  -csv
```

#### Fixed Operation Count

```bash
go run docdb/tests/load/cmd/loadtest/main.go \
  -operations 100000 \
  -workers 10 \
  -socket /tmp/docdb.sock \
  -db-name fixedtest \
  -wal-dir /tmp/docdb/wal \
  -output fixed_ops_results.json
```

## Output Format

### JSON Output

The JSON output file contains comprehensive test results:

```json
{
  "test_config": {
    "duration": "5m0s",
    "workers": 10,
    "read_percent": 40,
    "write_percent": 30,
    "update_percent": 20,
    "delete_percent": 10,
    ...
  },
  "duration_seconds": 300.0,
  "total_operations": 125000,
  "latency": {
    "create": {
      "p50": 1.2,
      "p95": 2.5,
      "p99": 5.1,
      "p999": 12.3,
      "mean": 1.5,
      "min": 0.8,
      "max": 15.2,
      "count": 37500
    },
    "read": {
      "p50": 0.8,
      "p95": 1.5,
      "p99": 3.2,
      "p999": 8.5,
      "mean": 1.0,
      "min": 0.5,
      "max": 10.1,
      "count": 50000
    },
    ...
  },
  "wal_growth": {
    "initial_size_bytes": 0,
    "final_size_bytes": 10485760,
    "max_size_bytes": 10485760,
    "growth_rate_bytes_per_sec": 34952.33,
    "sample_count": 300,
    "duration_seconds": 300.0
  },
  "healing": {
    "total_healings": 5,
    "total_documents_healed": 5,
    "healing_time_seconds": 0.5,
    "overhead_percent": 0.17,
    "event_count": 5,
    "initial_stats": {
      "total_scans": 10,
      "documents_healed": 0,
      ...
    },
    "final_stats": {
      "total_scans": 15,
      "documents_healed": 5,
      ...
    }
  }
}
```

### CSV Output

When `-csv` flag is used, three CSV files are generated:

1. **latency_samples.csv**: Individual operation latencies

   ```csv
   operation,latency_ms,timestamp
   create,1.234,2026-01-28T10:00:00.123456789Z
   read,0.856,2026-01-28T10:00:00.123789012Z
   ...
   ```

2. **wal_growth.csv**: WAL size over time

   ```csv
   timestamp,size_bytes,size_mb
   2026-01-28T10:00:00Z,0,0.00
   2026-01-28T10:00:01Z,34952,0.03
   ...
   ```

3. **healing_events.csv**: Healing operation events
   ```csv
   timestamp,duration_ms,documents_healed,type
   2026-01-28T10:05:00Z,100.5,1,on-demand
   ...
   ```

## Metrics Explained

### Latency Percentiles

- **P50 (Median)**: 50% of operations complete within this time
- **P95**: 95% of operations complete within this time
- **P99**: 99% of operations complete within this time
- **P99.9**: 99.9% of operations complete within this time

Lower percentiles indicate better performance. P95 and P99 are commonly used SLI targets.

### WAL Growth Rate

The WAL (Write-Ahead Log) growth rate indicates how quickly the database is accumulating write operations. Higher growth rates indicate:

- More write operations per second
- Larger document sizes
- More frequent updates

Monitor WAL growth to:

- Estimate disk space requirements
- Plan for WAL rotation and trimming
- Understand write workload intensity

### Healing Overhead

Healing overhead measures the time spent on document healing operations as a percentage of total test duration. This metric helps understand:

- Impact of corruption detection and recovery
- Background healing service overhead
- On-demand healing frequency

Lower overhead is better, but some overhead is expected for data integrity.

## Best Practices

1. **Warm-up Period**: Allow the database to warm up before starting measurements
2. **Consistent Environment**: Run tests on dedicated hardware with consistent load
3. **Multiple Runs**: Run tests multiple times and average results for reliability
4. **Monitor Resources**: Monitor CPU, memory, and disk I/O during tests
5. **Reproducibility**: Use `-seed` flag for reproducible test runs
6. **Gradual Scaling**: Start with fewer workers and gradually increase

## Troubleshooting

### Connection Errors

If you see connection errors:

- Ensure DocDB server is running
- Verify socket path is correct (`-socket` flag)
- Check socket file permissions

### WAL Size Tracking Errors

If WAL size tracking fails:

- Verify WAL directory path (`-wal-dir` flag) matches where DocDB actually stores WAL files
- The WAL directory is typically `{data-dir}/wal` where `data-dir` is what you passed to DocDB server
- If DocDB was started with `--data-dir ./data`, use `-wal-dir ./data/wal` (or absolute path)
- Ensure database name matches (`-db-name` flag)
- Check directory permissions
- Use `ls {wal-dir}/{db-name}.wal*` to verify WAL files exist

### Healing Stats Errors

If healing stats collection fails:

- Ensure healing is enabled in DocDB configuration
- Verify database ID is correct
- Check IPC connection

## Integration with CI/CD

Example GitHub Actions workflow:

```yaml
name: Load Test

on:
  schedule:
    - cron: "0 2 * * *" # Daily at 2 AM

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: "1.21"
      - name: Start DocDB
        run: |
          # Start DocDB server
      - name: Run Load Test
        run: |
          go run docdb/tests/load/cmd/loadtest/main.go \
            -duration 5m \
            -workers 10 \
            -socket /tmp/docdb.sock \
            -db-name ci_test \
            -wal-dir ./data/wal \
            -output ci_results.json \
            -csv
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: |
            ci_results.json
            *.csv
```

## Performance Tuning

Based on load test results, consider tuning:

1. **Worker Count**: Adjust `-workers` based on CPU cores and desired concurrency
2. **Document Size**: Test with realistic document sizes for your use case
3. **Workload Mix**: Match your production workload distribution
4. **Metrics Interval**: Use shorter intervals for more detailed analysis (higher overhead)

## Limitations

- WAL size tracking requires file system access to WAL directory
- Healing overhead measurement requires healing to be enabled
- Percentile calculation uses all samples (may use significant memory for long tests)
- CSV generation may be slow for very large test runs

## Contributing

When adding new metrics or features:

1. Update `config.go` for new configuration options
2. Add metrics collection in appropriate tracker
3. Update `reporter.go` for output formatting
4. Update this README with usage examples
5. Add tests for new functionality

## Multi-Database Load Testing

The load testing suite now supports testing multiple databases concurrently with variable workload profiles, load spikes, and per-database CRUD configurations.

### Multi-Database Usage

#### Command-Line (Simple)

```bash
go run docdb/tests/load/cmd/multidb_loadtest/main.go \
  -databases db1,db2,db3 \
  -workers-per-db 10 \
  -duration 5m \
  -socket /tmp/docdb.sock \
  -wal-dir ./data/wal \
  -output multidb_results.json \
  -csv
```

#### Configuration File

Create a JSON configuration file (`multidb_config.json`):

```json
{
  "databases": [
    {
      "name": "db1",
      "workers": 10,
      "doc_size": 1024,
      "doc_count": 10000,
      "crud": {
        "read": 50,
        "write": 30,
        "update": 15,
        "delete": 5
      },
      "wal_dir": "./data/wal"
    },
    {
      "name": "db2",
      "workers": 5,
      "doc_size": 2048,
      "doc_count": 5000,
      "crud": {
        "read": 80,
        "write": 10,
        "update": 5,
        "delete": 5
      }
    }
  ],
  "workload_profile": {
    "phases": [
      {
        "name": "warmup",
        "start_time": "0s",
        "duration": "1m",
        "workers": 5,
        "crud": {
          "read": 90,
          "write": 5,
          "update": 3,
          "delete": 2
        }
      },
      {
        "name": "spike",
        "start_time": "1m",
        "duration": "30s",
        "workers": 50,
        "crud": {
          "read": 20,
          "write": 60,
          "update": 15,
          "delete": 5
        }
      },
      {
        "name": "steady",
        "start_time": "1m30s",
        "duration": "3m30s",
        "workers": 10,
        "crud": {
          "read": 40,
          "write": 30,
          "update": 20,
          "delete": 10
        }
      }
    ]
  },
  "test": {
    "duration": "5m",
    "socket": "/tmp/docdb.sock",
    "metrics_interval": "1s",
    "output": "multidb_results.json",
    "csv_output": true,
    "seed": 12345
  }
}
```

Then run:

```bash
go run docdb/tests/load/cmd/multidb_loadtest/main.go \
  -config multidb_config.json
```

### Workload Profiles

Workload profiles allow you to define time-based workload changes:

#### Phased Workloads

Define distinct phases with different characteristics:

- **Warmup Phase**: Low load to initialize the system
- **Spike Phase**: Sudden load increase to test system resilience
- **Steady Phase**: Sustained load to measure steady-state performance

#### Gradual CRUD Transitions

For gradual transitions, use `crud_transition` instead of `crud`:

```json
{
  "name": "read_to_write_transition",
  "start_time": "0s",
  "duration": "5m",
  "workers": 10,
  "crud_transition": {
    "start": { "read": 80, "write": 10, "update": 5, "delete": 5 },
    "end": { "read": 10, "write": 80, "update": 5, "delete": 5 }
  }
}
```

This will linearly interpolate CRUD percentages over the phase duration.

### Per-Database CRUD Percentages

When `per_database_crud` is enabled (or when databases have their own `crud` config), each database uses its own CRUD percentages instead of the phase's CRUD percentages. This allows testing different workload patterns across databases simultaneously.

### Multi-Database Output

Multi-database tests generate:

1. **JSON Output**: Contains per-database metrics and aggregated global metrics
2. **Per-Database CSV**: Each database gets its own directory with latency summaries
3. **Global CSV Files**:
   - `global_wal_growth.csv`: WAL growth across all databases
   - `phase_stats.csv`: Statistics for each workload phase

### Example Multi-Database Results

```json
{
  "test_config": {...},
  "duration_seconds": 300,
  "total_operations": 75000,
  "databases": {
    "db1": {
      "total_operations": 50000,
      "latency": {...},
      "wal_growth": {...},
      "healing": {...}
    },
    "db2": {
      "total_operations": 25000,
      "latency": {...},
      "wal_growth": {...},
      "healing": {...}
    }
  },
  "global": {
    "total_operations": 75000,
    "aggregated_latency": {...},
    "total_wal_growth": 10485760,
    "phase_stats": [
      {
        "name": "warmup",
        "start_time": "0s",
        "duration": "1m",
        "operations": 10000,
        "workers": 5
      },
      {
        "name": "spike",
        "start_time": "1m",
        "duration": "30s",
        "operations": 50000,
        "workers": 50
      }
    ]
  }
}
```

### Command-Line Options for Multi-Database Tests

```
-config string
    Path to configuration file (JSON)

-databases string
    Comma-separated database names (alternative to config file)

-workers-per-db int
    Workers per database (when using -databases). Default: 10

-duration duration
    Test duration. Default: 5m

-socket string
    IPC socket path. Default: /tmp/docdb.sock

-wal-dir string
    WAL directory path. Default: ./data/wal

-output string
    Output JSON file path. Default: multidb_results.json

-csv
    Generate CSV output files. Default: false

-seed int
    Random seed for reproducibility. Default: current timestamp
```

## License

Part of the DocDB project. See main project LICENSE file.
