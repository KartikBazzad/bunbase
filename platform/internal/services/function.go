package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/pkg/functions"
)

// FunctionService handles function operations
type FunctionService struct {
	db              *sql.DB
	functionsClient *functions.Client
	bundleBasePath  string
}

// NewFunctionService creates a new FunctionService
func NewFunctionService(db *sql.DB, functionsSocketPath, bundleBasePath string) (*FunctionService, error) {
	client, err := functions.NewClient(functionsSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create functions client: %w", err)
	}

	return &FunctionService{
		db:              db,
		functionsClient: client,
		bundleBasePath:  bundleBasePath,
	}, nil
}

// Close closes the functions client
func (s *FunctionService) Close() error {
	if s.functionsClient != nil {
		return s.functionsClient.Close()
	}
	return nil
}

// DeployFunction deploys a function to a project
func (s *FunctionService) DeployFunction(projectID, name, runtime, handler, version string, bundleData []byte) (*models.Function, error) {
	// Get project first (needed for function service ID generation)
	project, err := s.getProjectByID(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if function already exists for this project
	var existingFunction models.Function
	var existingCreatedAt, existingUpdatedAt int64
	err = s.db.QueryRow(
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE project_id = ? AND name = ?",
		projectID, name,
	).Scan(&existingFunction.ID, &existingFunction.ProjectID, &existingFunction.FunctionServiceID, &existingFunction.Name, &existingFunction.Runtime, &existingCreatedAt, &existingUpdatedAt)

	var functionServiceID string
	if err == sql.ErrNoRows {
		// Function doesn't exist, register it
		// Generate function service ID: func-{project-slug}-{function-name}
		functionServiceID = fmt.Sprintf("func-%s-%s", project.Slug, name)

		// Register function in functions service
		_, err = s.functionsClient.RegisterFunction(functionServiceID, runtime, handler)
		if err != nil {
			return nil, fmt.Errorf("failed to register function: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing function: %w", err)
	} else {
		// Function exists, use existing function_service_id
		functionServiceID = existingFunction.FunctionServiceID
		existingFunction.CreatedAt = time.Unix(existingCreatedAt, 0)
		existingFunction.UpdatedAt = time.Unix(existingUpdatedAt, 0)
	}

	// Save bundle to filesystem
	bundleDir := filepath.Join(s.bundleBasePath, functionServiceID, version)
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bundle directory: %w", err)
	}

	bundlePath := filepath.Join(bundleDir, "bundle.js")
	if err := os.WriteFile(bundlePath, bundleData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write bundle: %w", err)
	}

	// Deploy function version
	_, err = s.functionsClient.DeployFunction(functionServiceID, version, bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy function: %w", err)
	}

	// Save or update function in platform database
	now := time.Now().Unix()
	
	// Check if we need to create or update
	var existingID string
	err = s.db.QueryRow(
		"SELECT id FROM functions WHERE project_id = ? AND name = ?",
		projectID, name,
	).Scan(&existingID)
	
	if err == sql.ErrNoRows {
		// Create new function record
		functionID := uuid.New().String()
		_, err = s.db.Exec(
			"INSERT INTO functions (id, project_id, function_service_id, name, runtime, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			functionID, projectID, functionServiceID, name, runtime, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create function record: %w", err)
		}

		return s.GetFunctionByID(functionID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check function: %w", err)
	} else {
		// Update existing function record
		_, err = s.db.Exec(
			"UPDATE functions SET runtime = ?, updated_at = ? WHERE id = ?",
			runtime, now, existingID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update function record: %w", err)
		}

		return s.GetFunctionByID(existingID)
	}
}

// GetFunctionByID retrieves a function by ID
func (s *FunctionService) GetFunctionByID(id string) (*models.Function, error) {
	var function models.Function
	var createdAt, updatedAt int64

	err := s.db.QueryRow(
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE id = ?",
		id,
	).Scan(&function.ID, &function.ProjectID, &function.FunctionServiceID, &function.Name, &function.Runtime, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("function not found")
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	function.CreatedAt = time.Unix(createdAt, 0)
	function.UpdatedAt = time.Unix(updatedAt, 0)
	return &function, nil
}

// ListFunctionsByProject lists all functions for a project
func (s *FunctionService) ListFunctionsByProject(projectID string) ([]*models.Function, error) {
	rows, err := s.db.Query(
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE project_id = ? ORDER BY created_at DESC",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}
	defer rows.Close()

	var functions []*models.Function
	for rows.Next() {
		var function models.Function
		var createdAt, updatedAt int64
		if err := rows.Scan(&function.ID, &function.ProjectID, &function.FunctionServiceID, &function.Name, &function.Runtime, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}
		function.CreatedAt = time.Unix(createdAt, 0)
		function.UpdatedAt = time.Unix(updatedAt, 0)
		functions = append(functions, &function)
	}

	return functions, nil
}

// DeleteFunction deletes a function
func (s *FunctionService) DeleteFunction(id string) error {
	_, err := s.db.Exec("DELETE FROM functions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}
	return nil
}

// Helper function to get project by ID
func (s *FunctionService) getProjectByID(id string) (*models.Project, error) {
	var project models.Project
	var createdAt, updatedAt int64

	err := s.db.QueryRow(
		"SELECT id, name, slug, owner_id, created_at, updated_at FROM projects WHERE id = ?",
		id,
	).Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.CreatedAt = time.Unix(createdAt, 0)
	project.UpdatedAt = time.Unix(updatedAt, 0)
	return &project, nil
}
