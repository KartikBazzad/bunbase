package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kartikbazzad/bunbase/buncast/pkg/client"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/pkg/functions"
)

// FunctionService handles function operations
type FunctionService struct {
	db              *pgxpool.Pool
	functionsClient *functions.Client
	buncastClient   *client.Client // optional: publish events on deploy
	bundleBasePath  string
}

// NewFunctionService creates a new FunctionService.
// buncastSocketPath is optional; if non-empty, deploy events are published to Buncast.
func NewFunctionService(db *pgxpool.Pool, functionsSocketPath, bundleBasePath, buncastSocketPath string) (*FunctionService, error) {
	fc, err := functions.NewClient(functionsSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create functions client: %w", err)
	}

	svc := &FunctionService{
		db:              db,
		functionsClient: fc,
		bundleBasePath:  bundleBasePath,
	}
	if buncastSocketPath != "" {
		svc.buncastClient = client.New(buncastSocketPath)
	}
	return svc, nil
}

// Close closes the functions and buncast clients
func (s *FunctionService) Close() error {
	if s.functionsClient != nil {
		_ = s.functionsClient.Close()
	}
	if s.buncastClient != nil {
		_ = s.buncastClient.Close()
	}
	return nil
}

// DeployFunction deploys a function to a project
func (s *FunctionService) DeployFunction(projectID, name, runtime, handler, version string, bundleData []byte) (*models.Function, error) {
	ctx := context.Background()
	// Get project first (needed for function service ID generation)
	project, err := s.getProjectByID(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if function already exists for this project
	var existingFunction models.Function
	err = s.db.QueryRow(ctx,
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE project_id = $1 AND name = $2",
		projectID, name,
	).Scan(&existingFunction.ID, &existingFunction.ProjectID, &existingFunction.FunctionServiceID, &existingFunction.Name, &existingFunction.Runtime, &existingFunction.CreatedAt, &existingFunction.UpdatedAt)

	var functionServiceID string
	if err == pgx.ErrNoRows {
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

	// Publish deploy event to Buncast if configured
	if s.buncastClient != nil {
		event := map[string]string{
			"project_id": projectID,
			"name":       name,
			"version":    version,
			"service_id": functionServiceID,
		}
		if payload, e := json.Marshal(event); e == nil {
			_ = s.buncastClient.Publish("functions.deployments", payload)
		}
	}

	// Save or update function in platform database
	now := time.Now()

	// Check if we need to create or update
	var existingID string
	err = s.db.QueryRow(ctx,
		"SELECT id FROM functions WHERE project_id = $1 AND name = $2",
		projectID, name,
	).Scan(&existingID)

	if err == pgx.ErrNoRows {
		// Create new function record
		functionID := uuid.New().String()
		_, err = s.db.Exec(ctx,
			"INSERT INTO functions (id, project_id, function_service_id, name, runtime, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
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
		_, err = s.db.Exec(ctx,
			"UPDATE functions SET runtime = $1, updated_at = $2 WHERE id = $3",
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

	err := s.db.QueryRow(context.Background(),
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE id = $1",
		id,
	).Scan(&function.ID, &function.ProjectID, &function.FunctionServiceID, &function.Name, &function.Runtime, &function.CreatedAt, &function.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("function not found")
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return &function, nil
}

// GetFunctionByName retrieves a function by name and project ID
func (s *FunctionService) GetFunctionByName(projectID, name string) (*models.Function, error) {
	var function models.Function

	err := s.db.QueryRow(context.Background(),
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE project_id = $1 AND name = $2",
		projectID, name,
	).Scan(&function.ID, &function.ProjectID, &function.FunctionServiceID, &function.Name, &function.Runtime, &function.CreatedAt, &function.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("function not found")
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return &function, nil
}

// ListFunctionsByProject lists all functions for a project
func (s *FunctionService) ListFunctionsByProject(projectID string) ([]*models.Function, error) {
	rows, err := s.db.Query(context.Background(),
		"SELECT id, project_id, function_service_id, name, runtime, created_at, updated_at FROM functions WHERE project_id = $1 ORDER BY created_at DESC",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}
	defer rows.Close()

	var functions []*models.Function
	for rows.Next() {
		var function models.Function
		if err := rows.Scan(&function.ID, &function.ProjectID, &function.FunctionServiceID, &function.Name, &function.Runtime, &function.CreatedAt, &function.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}
		functions = append(functions, &function)
	}

	return functions, nil
}

// DeleteFunction deletes a function
func (s *FunctionService) DeleteFunction(id string) error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM functions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}
	return nil
}

// Helper function to get project by ID
func (s *FunctionService) getProjectByID(id string) (*models.Project, error) {
	var project models.Project

	err := s.db.QueryRow(context.Background(),
		"SELECT id, name, slug, owner_id, created_at, updated_at FROM projects WHERE id = $1",
		id,
	).Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows || err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}
