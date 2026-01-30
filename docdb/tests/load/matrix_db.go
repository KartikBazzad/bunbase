package load

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	matrixDBFilename = "matrix_results.db"
)

// MatrixDBPath returns the path to the matrix results SQLite DB for a given results directory.
func MatrixDBPath(resultsDir string) string {
	return filepath.Join(resultsDir, matrixDBFilename)
}

// OpenMatrixDB opens or creates the matrix results database at the given path.
func OpenMatrixDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open matrix db: %w", err)
	}
	if err := initMatrixSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func initMatrixSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			output_dir TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT,
			total_tests INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			fail_count INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id INTEGER NOT NULL REFERENCES runs(id),
			config_name TEXT NOT NULL,
			databases INTEGER NOT NULL,
			connections_per_db INTEGER NOT NULL,
			workers_per_db INTEGER NOT NULL,
			duration_sec REAL NOT NULL,
			total_ops INTEGER NOT NULL,
			throughput REAL NOT NULL,
			p95_latency_ms REAL NOT NULL,
			p99_latency_ms REAL NOT NULL,
			success INTEGER NOT NULL,
			result_file TEXT NOT NULL
		);
	`)
	return err
}

// InsertRun inserts a new run row and returns its id.
func InsertRun(db *sql.DB, outputDir string) (int64, error) {
	startedAt := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO runs (output_dir, started_at) VALUES (?, ?)`,
		outputDir, startedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert run: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

// UpdateRun updates the run with final counts and finished_at.
func UpdateRun(db *sql.DB, runID int64, totalTests, successCount, failCount int) error {
	finishedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE runs SET finished_at = ?, total_tests = ?, success_count = ?, fail_count = ? WHERE id = ?`,
		finishedAt, totalTests, successCount, failCount, runID,
	)
	return err
}

// InsertResult inserts a single result row for the given run.
func InsertResult(db *sql.DB, runID int64, r *TestResultData) error {
	success := 0
	if r.Success {
		success = 1
	}
	_, err := db.Exec(
		`INSERT INTO results (
			run_id, config_name, databases, connections_per_db, workers_per_db,
			duration_sec, total_ops, throughput, p95_latency_ms, p99_latency_ms,
			success, result_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, r.Config.Name, r.Config.Databases, r.Config.ConnectionsPerDB, r.Config.WorkersPerDB,
		r.Duration, r.TotalOps, r.Throughput, r.P95Latency, r.P99Latency,
		success, r.ResultFile,
	)
	return err
}

// QueryLatestRunID returns the id of the most recent run, or 0 if none.
func QueryLatestRunID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM runs ORDER BY id DESC LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

// QueryResultsByRunID returns all result rows for the given run as TestResultData.
func QueryResultsByRunID(db *sql.DB, runID int64) ([]TestResultData, error) {
	rows, err := db.Query(
		`SELECT config_name, databases, connections_per_db, workers_per_db,
			duration_sec, total_ops, throughput, p95_latency_ms, p99_latency_ms,
			success, result_file
		 FROM results WHERE run_id = ? ORDER BY config_name`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TestResultData
	for rows.Next() {
		var (
			configName       string
			databases        int
			connectionsPerDB int
			workersPerDB     int
			durationSec      float64
			totalOps         int64
			throughput       float64
			p95Ms            float64
			p99Ms            float64
			success          int
			resultFile       string
		)
		if err := rows.Scan(
			&configName, &databases, &connectionsPerDB, &workersPerDB,
			&durationSec, &totalOps, &throughput, &p95Ms, &p99Ms,
			&success, &resultFile,
		); err != nil {
			return nil, err
		}
		list = append(list, TestResultData{
			Config: TestConfiguration{
				Name:             configName,
				Databases:        databases,
				ConnectionsPerDB: connectionsPerDB,
				WorkersPerDB:     workersPerDB,
			},
			ResultFile: resultFile,
			Throughput: throughput,
			P95Latency: p95Ms,
			P99Latency: p99Ms,
			TotalOps:   totalOps,
			Duration:   durationSec,
			Success:    success != 0,
		})
	}
	return list, rows.Err()
}
