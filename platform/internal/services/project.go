package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// ProjectService handles project operations
type ProjectService struct {
	db *sql.DB
}

// NewProjectService creates a new ProjectService
func NewProjectService(db *sql.DB) *ProjectService {
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
	// Generate slug
	baseSlug := generateSlug(name)
	slug := baseSlug

	// Ensure slug is unique
	for {
		var existingID string
		err := s.db.QueryRow("SELECT id FROM projects WHERE slug = ?", slug).Scan(&existingID)
		if err == sql.ErrNoRows {
			break // Slug is unique
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		// Slug exists, append random suffix
		slug = baseSlug + "-" + uuid.New().String()[:8]
	}

	projectID := uuid.New().String()
	now := time.Now().Unix()

	_, err := s.db.Exec(
		"INSERT INTO projects (id, name, slug, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		projectID, name, slug, ownerID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Add owner as project member with owner role
	memberID := uuid.New().String()
	_, err = s.db.Exec(
		"INSERT INTO project_members (id, project_id, user_id, role, created_at) VALUES (?, ?, ?, ?, ?)",
		memberID, projectID, ownerID, "owner", now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return s.GetProjectByID(projectID)
}

// GetProjectByID retrieves a project by ID
func (s *ProjectService) GetProjectByID(id string) (*models.Project, error) {
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

// GetProjectBySlug retrieves a project by slug
func (s *ProjectService) GetProjectBySlug(slug string) (*models.Project, error) {
	var project models.Project
	var createdAt, updatedAt int64

	err := s.db.QueryRow(
		"SELECT id, name, slug, owner_id, created_at, updated_at FROM projects WHERE slug = ?",
		slug,
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

// ListProjectsByUser lists all projects for a user
func (s *ProjectService) ListProjectsByUser(userID string) ([]*models.Project, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT p.id, p.name, p.slug, p.owner_id, p.created_at, p.updated_at 
		 FROM projects p
		 LEFT JOIN project_members pm ON p.id = pm.project_id
		 WHERE p.owner_id = ? OR pm.user_id = ?
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
		var createdAt, updatedAt int64
		if err := rows.Scan(&project.ID, &project.Name, &project.Slug, &project.OwnerID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		project.CreatedAt = time.Unix(createdAt, 0)
		project.UpdatedAt = time.Unix(updatedAt, 0)
		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(id, name string) (*models.Project, error) {
	now := time.Now().Unix()

	// Generate new slug if name changed
	baseSlug := generateSlug(name)
	slug := baseSlug

	// Check if slug is unique (excluding current project)
	var existingID string
	err := s.db.QueryRow("SELECT id FROM projects WHERE slug = ? AND id != ?", slug, id).Scan(&existingID)
	if err == nil {
		// Slug exists, append random suffix
		slug = baseSlug + "-" + uuid.New().String()[:8]
	}

	_, err = s.db.Exec(
		"UPDATE projects SET name = ?, slug = ?, updated_at = ? WHERE id = ?",
		name, slug, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return s.GetProjectByID(id)
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(id string) error {
	_, err := s.db.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// IsProjectMember checks if a user is a member of a project
func (s *ProjectService) IsProjectMember(projectID, userID string) (bool, string, error) {
	var role string
	err := s.db.QueryRow(
		"SELECT role FROM project_members WHERE project_id = ? AND user_id = ?",
		projectID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows {
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
	err := s.db.QueryRow("SELECT owner_id FROM projects WHERE id = ?", projectID).Scan(&ownerID)
	if err != nil {
		return false, fmt.Errorf("failed to check ownership: %w", err)
	}
	return ownerID == userID, nil
}
