package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// ProjectService handles project operations
type ProjectService struct {
	db *pgxpool.Pool
}

// NewProjectService creates a new ProjectService
func NewProjectService(db *pgxpool.Pool) *ProjectService {
	return &ProjectService{db: db}
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove special characters
	var result strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(name, ownerID string) (*models.Project, error) {
	ctx := context.Background()
	// Generate slug
	baseSlug := generateSlug(name)
	slug := baseSlug

	// Ensure slug is unique
	for {
		var existingID string
		err := s.db.QueryRow(ctx, "SELECT id FROM projects WHERE slug = $1", slug).Scan(&existingID)
		if err == pgx.ErrNoRows {
			break // Slug is unique
		}
		if err != nil {
			return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		// Slug exists, append random suffix
		slug = baseSlug + "-" + uuid.New().String()[:8]
	}

	projectID := uuid.New().String()
	publicAPIKey := "pk_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	now := time.Now()

	_, err := s.db.Exec(ctx,
		"INSERT INTO projects (id, name, slug, owner_id, public_api_key, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		projectID, name, slug, ownerID, publicAPIKey, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Add owner as project member with owner role
	memberID := uuid.New().String()
	_, err = s.db.Exec(ctx,
		"INSERT INTO project_members (id, project_id, user_id, role, created_at) VALUES ($1, $2, $3, $4, $5)",
		memberID, projectID, ownerID, "owner", now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return s.GetProjectByID(projectID)
}

// GetProjectByID retrieves a project by ID (includes public_api_key for single-project fetch).
func (s *ProjectService) GetProjectByID(id string) (*models.Project, error) {
	var project models.Project
	var apiKey *string
	var functionSubdomain *string

	err := s.db.QueryRow(context.Background(),
		"SELECT id, name, slug, owner_id, public_api_key, function_subdomain, created_at, updated_at FROM projects WHERE id = $1",
		id,
	).Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &apiKey, &functionSubdomain, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	project.PublicAPIKey = apiKey
	project.FunctionSubdomain = functionSubdomain
	return &project, nil
}

// GetProjectBySlug retrieves a project by slug
func (s *ProjectService) GetProjectBySlug(slug string) (*models.Project, error) {
	var project models.Project
	var functionSubdomain *string

	err := s.db.QueryRow(context.Background(),
		"SELECT id, name, slug, owner_id, function_subdomain, created_at, updated_at FROM projects WHERE slug = $1",
		slug,
	).Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &functionSubdomain, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	project.FunctionSubdomain = functionSubdomain
	return &project, nil
}

// ListProjectsByUser lists all projects for a user
func (s *ProjectService) ListProjectsByUser(userID string) ([]*models.Project, error) {
	rows, err := s.db.Query(context.Background(),
		`SELECT DISTINCT p.id, p.name, p.slug, p.owner_id, p.created_at, p.updated_at 
		 FROM projects p
		 LEFT JOIN project_members pm ON p.id = pm.project_id
		 WHERE p.owner_id = $1 OR pm.user_id = $2
		 ORDER BY p.created_at DESC`,
		userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(id, name string) (*models.Project, error) {
	now := time.Now()
	ctx := context.Background()

	// Generate new slug if name changed
	baseSlug := generateSlug(name)
	slug := baseSlug

	// Check if slug is unique (excluding current project)
	var existingID string
	err := s.db.QueryRow(ctx, "SELECT id FROM projects WHERE slug = $1 AND id != $2", slug, id).Scan(&existingID)
	if err == nil {
		// Slug exists, append random suffix
		slug = baseSlug + "-" + uuid.New().String()[:8]
	}

	_, err = s.db.Exec(ctx,
		"UPDATE projects SET name = $1, slug = $2, updated_at = $3 WHERE id = $4",
		name, slug, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return s.GetProjectByID(id)
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(id string) error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// IsProjectMember checks if a user is a member of a project
func (s *ProjectService) IsProjectMember(projectID, userID string) (bool, string, error) {
	var role string
	err := s.db.QueryRow(context.Background(),
		"SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2",
		projectID, userID,
	).Scan(&role)
	if err == pgx.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("failed to check membership: %w", err)
	}
	return true, role, nil
}

// IsProjectOwner checks if a user owns a project
func (s *ProjectService) IsProjectOwner(projectID, userID string) (bool, error) {
	var ownerID string
	err := s.db.QueryRow(context.Background(), "SELECT owner_id FROM projects WHERE id = $1", projectID).Scan(&ownerID)
	if err != nil {
		return false, fmt.Errorf("failed to check ownership: %w", err)
	}
	return ownerID == userID, nil
}

// GetProjectIDByPublicKey retrieves project ID by public API key
func (s *ProjectService) GetProjectIDByPublicKey(key string) (string, error) {
	var id string
	err := s.db.QueryRow(context.Background(), "SELECT id FROM projects WHERE public_api_key = $1", key).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("invalid api key")
		}
		return "", fmt.Errorf("failed to get project: %w", err)
	}
	return id, nil
}

// RegenerateProjectAPIKey generates a new public API key for the project and returns the project and the new key (shown once).
// Caller must ensure the user is owner or admin.
func (s *ProjectService) RegenerateProjectAPIKey(projectID string) (*models.Project, string, error) {
	ctx := context.Background()
	newKey := "pk_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err := s.db.Exec(ctx,
		"UPDATE projects SET public_api_key = $1, updated_at = $2 WHERE id = $3",
		newKey, time.Now(), projectID,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to regenerate api key: %w", err)
	}
	project, err := s.GetProjectByID(projectID)
	if err != nil {
		return nil, "", err
	}
	// Return the new key in plain form (only time it's returned)
	project.PublicAPIKey = &newKey
	return project, newKey, nil
}
