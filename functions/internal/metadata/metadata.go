package metadata

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	_ "github.com/mattn/go-sqlite3"
)

// FunctionStatus represents the status of a function
type FunctionStatus string

const (
	FunctionStatusRegistered FunctionStatus = "registered"
	FunctionStatusBuilt      FunctionStatus = "built"
	FunctionStatusDeployed   FunctionStatus = "deployed"
)

// Function represents a function definition
type Function struct {
	ID              string
	Name            string
	Runtime         string
	Handler         string
	Status          FunctionStatus
	ActiveVersionID string
	Capabilities    *capabilities.Capabilities // Security capabilities
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// FunctionVersion represents a function code version
type FunctionVersion struct {
	ID         string
	FunctionID string
	Version    string
	BundlePath string
	CreatedAt  time.Time
}

// FunctionDeployment represents a function deployment
type FunctionDeployment struct {
	ID         string
	FunctionID string
	VersionID  string
	Status     string
	CreatedAt  time.Time
}

// Store manages function metadata in SQLite
type Store struct {
	db *sql.DB
}

// NewStore creates a new metadata store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS functions (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		runtime TEXT NOT NULL,
		handler TEXT NOT NULL,
		status TEXT NOT NULL,
		active_version_id TEXT,
		capabilities_json TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS function_versions (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		version TEXT NOT NULL,
		bundle_path TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (function_id) REFERENCES functions(id),
		UNIQUE(function_id, version)
	);

	CREATE TABLE IF NOT EXISTS function_deployments (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		version_id TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (function_id) REFERENCES functions(id),
		FOREIGN KEY (version_id) REFERENCES function_versions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_functions_name ON functions(name);
	CREATE INDEX IF NOT EXISTS idx_versions_function_id ON function_versions(function_id);
	CREATE INDEX IF NOT EXISTS idx_deployments_function_id ON function_deployments(function_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// RegisterFunction registers a new function
func (s *Store) RegisterFunction(id, name, runtime, handler string, caps *capabilities.Capabilities) (*Function, error) {
	now := time.Now().Unix()
	capsJSON := s.capabilitiesToJSON(caps)
	query := `
		INSERT INTO functions (id, name, runtime, handler, status, capabilities_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, id, name, runtime, handler, FunctionStatusRegistered, capsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to register function: %w", err)
	}

	return s.GetFunctionByID(id)
}

// GetFunctionByID gets a function by ID
func (s *Store) GetFunctionByID(id string) (*Function, error) {
	query := `
		SELECT id, name, runtime, handler, status, active_version_id, capabilities_json, created_at, updated_at
		FROM functions
		WHERE id = ?
	`

	var f Function
	var capsJSON sql.NullString
	var activeVersionID sql.NullString
	var createdAt, updatedAt int64
	err := s.db.QueryRow(query, id).Scan(
		&f.ID, &f.Name, &f.Runtime, &f.Handler, &f.Status, &activeVersionID, &capsJSON,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("function not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	if activeVersionID.Valid {
		f.ActiveVersionID = activeVersionID.String
	}
	f.Capabilities = s.jsonToCapabilities(capsJSON)
	f.CreatedAt = time.Unix(createdAt, 0)
	f.UpdatedAt = time.Unix(updatedAt, 0)
	return &f, nil
}

// GetFunctionByName gets a function by name
func (s *Store) GetFunctionByName(name string) (*Function, error) {
	query := `
		SELECT id, name, runtime, handler, status, active_version_id, capabilities_json, created_at, updated_at
		FROM functions
		WHERE name = ?
	`

	var f Function
	var capsJSON sql.NullString
	var activeVersionID sql.NullString
	var createdAt, updatedAt int64
	err := s.db.QueryRow(query, name).Scan(
		&f.ID, &f.Name, &f.Runtime, &f.Handler, &f.Status, &activeVersionID, &capsJSON,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("function not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	if activeVersionID.Valid {
		f.ActiveVersionID = activeVersionID.String
	}
	f.Capabilities = s.jsonToCapabilities(capsJSON)
	f.CreatedAt = time.Unix(createdAt, 0)
	f.UpdatedAt = time.Unix(updatedAt, 0)
	return &f, nil
}

// ListFunctions lists all functions
func (s *Store) ListFunctions() ([]*Function, error) {
	query := `
		SELECT id, name, runtime, handler, status, active_version_id, capabilities_json, created_at, updated_at
		FROM functions
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}
	defer rows.Close()

	var functions []*Function
	for rows.Next() {
		var f Function
		var capsJSON sql.NullString
		var activeVersionID sql.NullString
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&f.ID, &f.Name, &f.Runtime, &f.Handler, &f.Status, &activeVersionID, &capsJSON,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}
		if activeVersionID.Valid {
			f.ActiveVersionID = activeVersionID.String
		}
		f.Capabilities = s.jsonToCapabilities(capsJSON)
		f.CreatedAt = time.Unix(createdAt, 0)
		f.UpdatedAt = time.Unix(updatedAt, 0)
		functions = append(functions, &f)
	}

	return functions, nil
}

// CreateVersion creates a new function version
func (s *Store) CreateVersion(id, functionID, version, bundlePath string) (*FunctionVersion, error) {
	now := time.Now().Unix()
	query := `
		INSERT INTO function_versions (id, function_id, version, bundle_path, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, id, functionID, version, bundlePath, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return s.GetVersionByID(id)
}

// GetVersionByID gets a version by ID
func (s *Store) GetVersionByID(id string) (*FunctionVersion, error) {
	query := `
		SELECT id, function_id, version, bundle_path, created_at
		FROM function_versions
		WHERE id = ?
	`

	var v FunctionVersion
	var createdAt int64
	err := s.db.QueryRow(query, id).Scan(&v.ID, &v.FunctionID, &v.Version, &v.BundlePath, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("version not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	v.CreatedAt = time.Unix(createdAt, 0)
	return &v, nil
}

// GetVersionsByFunctionID gets all versions for a function
func (s *Store) GetVersionsByFunctionID(functionID string) ([]*FunctionVersion, error) {
	query := `
		SELECT id, function_id, version, bundle_path, created_at
		FROM function_versions
		WHERE function_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, functionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}
	defer rows.Close()

	var versions []*FunctionVersion
	for rows.Next() {
		var v FunctionVersion
		var createdAt int64
		if err := rows.Scan(&v.ID, &v.FunctionID, &v.Version, &v.BundlePath, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		v.CreatedAt = time.Unix(createdAt, 0)
		versions = append(versions, &v)
	}

	return versions, nil
}

// DeployFunction deploys a function version
func (s *Store) DeployFunction(deploymentID, functionID, versionID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	// Deactivate existing deployments
	_, err = tx.Exec(`
		UPDATE function_deployments
		SET status = 'inactive'
		WHERE function_id = ? AND status = 'active'
	`, functionID)
	if err != nil {
		return fmt.Errorf("failed to deactivate deployments: %w", err)
	}

	// Create new deployment
	_, err = tx.Exec(`
		INSERT INTO function_deployments (id, function_id, version_id, status, created_at)
		VALUES (?, ?, ?, 'active', ?)
	`, deploymentID, functionID, versionID, now)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Update function status and active version
	_, err = tx.Exec(`
		UPDATE functions
		SET status = ?, active_version_id = ?, updated_at = ?
		WHERE id = ?
	`, FunctionStatusDeployed, versionID, now, functionID)
	if err != nil {
		return fmt.Errorf("failed to update function: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetActiveDeployment gets the active deployment for a function
func (s *Store) GetActiveDeployment(functionID string) (*FunctionDeployment, error) {
	query := `
		SELECT id, function_id, version_id, status, created_at
		FROM function_deployments
		WHERE function_id = ? AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var d FunctionDeployment
	var createdAt int64
	err := s.db.QueryRow(query, functionID).Scan(&d.ID, &d.FunctionID, &d.VersionID, &d.Status, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active deployment found for function: %s", functionID)
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	d.CreatedAt = time.Unix(createdAt, 0)
	return &d, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// capabilitiesToJSON converts capabilities to JSON string
func (s *Store) capabilitiesToJSON(caps *capabilities.Capabilities) string {
	if caps == nil {
		return ""
	}
	data, err := json.Marshal(caps)
	if err != nil {
		return ""
	}
	return string(data)
}

// jsonToCapabilities converts JSON string to capabilities
func (s *Store) jsonToCapabilities(capsJSON sql.NullString) *capabilities.Capabilities {
	if !capsJSON.Valid || capsJSON.String == "" {
		return nil
	}
	var caps capabilities.Capabilities
	if err := json.Unmarshal([]byte(capsJSON.String), &caps); err != nil {
		return nil
	}
	return &caps
}

// UpdateFunctionCapabilities updates the capabilities for a function
func (s *Store) UpdateFunctionCapabilities(functionID string, caps *capabilities.Capabilities) error {
	capsJSON := s.capabilitiesToJSON(caps)
	now := time.Now().Unix()
	query := `
		UPDATE functions
		SET capabilities_json = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query, capsJSON, now, functionID)
	if err != nil {
		return fmt.Errorf("failed to update capabilities: %w", err)
	}
	return nil
}
