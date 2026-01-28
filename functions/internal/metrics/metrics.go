package metrics

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Metrics represents function execution metrics
type Metrics struct {
	FunctionID      string
	Invocations     int64
	Errors          int64
	TotalDuration   int64 // milliseconds
	ColdStarts      int64
	LastInvoked     *time.Time
}

// Store manages metrics storage
type Store struct {
	db    *sql.DB
	mu    sync.Mutex
	cache map[string]*Metrics // functionID -> metrics
}

// NewStore creates a new metrics store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:    db,
		cache: make(map[string]*Metrics),
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS function_metrics (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		date INTEGER NOT NULL,
		invocations INTEGER NOT NULL DEFAULT 0,
		errors INTEGER NOT NULL DEFAULT 0,
		total_duration INTEGER NOT NULL DEFAULT 0,
		cold_starts INTEGER NOT NULL DEFAULT 0,
		UNIQUE(function_id, date)
	);

	CREATE TABLE IF NOT EXISTS function_metrics_minute (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		invocations INTEGER NOT NULL DEFAULT 0,
		errors INTEGER NOT NULL DEFAULT 0,
		total_duration INTEGER NOT NULL DEFAULT 0,
		cold_starts INTEGER NOT NULL DEFAULT 0,
		UNIQUE(function_id, timestamp)
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_function_id ON function_metrics(function_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_date ON function_metrics(date);
	CREATE INDEX IF NOT EXISTS idx_metrics_minute_function_id ON function_metrics_minute(function_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_minute_timestamp ON function_metrics_minute(timestamp);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// RecordInvocation records an invocation
func (s *Store) RecordInvocation(functionID string, duration time.Duration, isError, isColdStart bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	durationMS := int64(duration / time.Millisecond)
	now := time.Now()

	// Update daily metrics
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayUnix := today.Unix()

	query := `
		INSERT INTO function_metrics (id, function_id, date, invocations, errors, total_duration, cold_starts)
		VALUES (?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(function_id, date) DO UPDATE SET
			invocations = invocations + 1,
			errors = errors + ?,
			total_duration = total_duration + ?,
			cold_starts = cold_starts + ?
	`

	id := fmt.Sprintf("%s-%d", functionID, todayUnix)
	errors := int64(0)
	coldStarts := int64(0)
	if isError {
		errors = 1
	}
	if isColdStart {
		coldStarts = 1
	}

	_, err := s.db.Exec(query, id, functionID, todayUnix, errors, durationMS, coldStarts, errors, durationMS, coldStarts)
	if err != nil {
		return fmt.Errorf("failed to record daily metrics: %w", err)
	}

	// Update minute-level metrics
	minuteTimestamp := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	minuteUnix := minuteTimestamp.Unix()

	queryMinute := `
		INSERT INTO function_metrics_minute (id, function_id, timestamp, invocations, errors, total_duration, cold_starts)
		VALUES (?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(function_id, timestamp) DO UPDATE SET
			invocations = invocations + 1,
			errors = errors + ?,
			total_duration = total_duration + ?,
			cold_starts = cold_starts + ?
	`

	idMinute := fmt.Sprintf("%s-%d", functionID, minuteUnix)
	_, err = s.db.Exec(queryMinute, idMinute, functionID, minuteUnix, errors, durationMS, coldStarts, errors, durationMS, coldStarts)
	if err != nil {
		return fmt.Errorf("failed to record minute metrics: %w", err)
	}

	return nil
}

// GetMetrics retrieves metrics for a function
type MetricsPeriod string

const (
	MetricsPeriodMinute MetricsPeriod = "minute"
	MetricsPeriodHour   MetricsPeriod = "hour"
	MetricsPeriodDay    MetricsPeriod = "day"
)

// GetMetrics retrieves metrics for a function
func (s *Store) GetMetrics(functionID string, period MetricsPeriod) (*Metrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var query string
	var args []interface{}

	switch period {
	case MetricsPeriodMinute:
		// Get last hour of minute-level metrics
		oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()
		query = `
			SELECT 
				SUM(invocations) as invocations,
				SUM(errors) as errors,
				SUM(total_duration) as total_duration,
				SUM(cold_starts) as cold_starts,
				MAX(timestamp) as last_invoked
			FROM function_metrics_minute
			WHERE function_id = ? AND timestamp >= ?
		`
		args = []interface{}{functionID, oneHourAgo}
	case MetricsPeriodDay:
		// Get today's metrics
		today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
		todayUnix := today.Unix()
		query = `
			SELECT 
				invocations,
				errors,
				total_duration,
				cold_starts,
				date as last_invoked
			FROM function_metrics
			WHERE function_id = ? AND date = ?
		`
		args = []interface{}{functionID, todayUnix}
	default:
		return nil, fmt.Errorf("unsupported period: %s", period)
	}

	var m Metrics
	var lastInvokedUnix int64
	var invocations, errors, totalDuration, coldStarts sql.NullInt64

	err := s.db.QueryRow(query, args...).Scan(
		&invocations,
		&errors,
		&totalDuration,
		&coldStarts,
		&lastInvokedUnix,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return zero metrics
			return &Metrics{
				FunctionID:  functionID,
				Invocations: 0,
				Errors:      0,
				TotalDuration: 0,
				ColdStarts:  0,
				LastInvoked: nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	m.FunctionID = functionID
	if invocations.Valid {
		m.Invocations = invocations.Int64
	}
	if errors.Valid {
		m.Errors = errors.Int64
	}
	if totalDuration.Valid {
		m.TotalDuration = totalDuration.Int64
	}
	if coldStarts.Valid {
		m.ColdStarts = coldStarts.Int64
	}
	if lastInvokedUnix > 0 {
		t := time.Unix(lastInvokedUnix, 0)
		m.LastInvoked = &t
	}

	return &m, nil
}

// Close closes the metrics store
func (s *Store) Close() error {
	return s.db.Close()
}
