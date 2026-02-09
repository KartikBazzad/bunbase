package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InstanceService handles instance-level state (e.g. self-hosted root admins).
type InstanceService struct {
	db             *pgxpool.Pool
	deploymentMode string
}

// NewInstanceService creates a new InstanceService.
// deploymentMode should be "cloud" or "self_hosted".
func NewInstanceService(db *pgxpool.Pool, deploymentMode string) *InstanceService {
	if deploymentMode == "" {
		deploymentMode = "cloud"
	}
	return &InstanceService{db: db, deploymentMode: deploymentMode}
}

// IsInstanceAdmin returns true if userID is in instance_admins.
func (s *InstanceService) IsInstanceAdmin(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM instance_admins WHERE user_id = $1)",
		userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check instance admin: %w", err)
	}
	return exists, nil
}

// BootstrapAdmin inserts userID into instance_admins. Idempotent: no error if already present.
func (s *InstanceService) BootstrapAdmin(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO instance_admins (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING",
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to bootstrap admin: %w", err)
	}
	return nil
}

// SetupComplete returns true when the instance is considered "set up":
// - cloud: always true
// - self_hosted: true if at least one row exists in instance_admins
func (s *InstanceService) SetupComplete(ctx context.Context) (bool, error) {
	if s.deploymentMode != "self_hosted" {
		return true, nil
	}
	var count int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM instance_admins").Scan(&count)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check setup complete: %w", err)
	}
	return count > 0, nil
}

// DeploymentMode returns the current deployment mode.
func (s *InstanceService) DeploymentMode() string {
	return s.deploymentMode
}
