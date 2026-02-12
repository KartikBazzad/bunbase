package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Limit errors for handlers to map to 403 with a clear message.
var (
	ErrProjectLimitReached  = errors.New("project limit reached")
	ErrFunctionLimitReached = errors.New("function limit reached")
	ErrAPITokenLimitReached = errors.New("api token limit reached")
)

// LimitsConfig holds per-user resource limits (0 = unlimited).
type LimitsConfig struct {
	MaxProjectsPerUser     int
	MaxFunctionsPerProject int
	MaxAPITokensPerUser    int
}

// LimitService checks resource limits in cloud mode.
type LimitService struct {
	db       *pgxpool.Pool
	mode     string
	limits   LimitsConfig
}

// NewLimitService creates a limit service. mode should be "cloud" or "self_hosted".
// When mode is "self_hosted", all check methods return nil (no limits applied).
func NewLimitService(db *pgxpool.Pool, mode string, limits LimitsConfig) *LimitService {
	return &LimitService{db: db, mode: mode, limits: limits}
}

// CheckProjectLimit returns nil if the user is under the project limit, or ErrProjectLimitReached (with limit set) if at or over.
func (s *LimitService) CheckProjectLimit(ctx context.Context, userID string) error {
	if s.mode != "cloud" || s.limits.MaxProjectsPerUser <= 0 {
		return nil
	}
	var n int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM projects WHERE owner_id = $1", userID).Scan(&n)
	if err != nil {
		return fmt.Errorf("count projects: %w", err)
	}
	if n >= s.limits.MaxProjectsPerUser {
		return ErrProjectLimitReached
	}
	return nil
}

// CheckFunctionLimit returns nil if under the function limit (per project), or ErrFunctionLimitReached if at or over.
func (s *LimitService) CheckFunctionLimit(ctx context.Context, projectID string) error {
	if s.mode != "cloud" || s.limits.MaxFunctionsPerProject <= 0 {
		return nil
	}
	var n int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM functions WHERE project_id = $1", projectID).Scan(&n)
	if err != nil {
		return fmt.Errorf("count functions: %w", err)
	}
	if n >= s.limits.MaxFunctionsPerProject {
		return ErrFunctionLimitReached
	}
	return nil
}

// CheckAPITokenLimit returns nil if the user is under the API token limit, or ErrAPITokenLimitReached if at or over.
func (s *LimitService) CheckAPITokenLimit(ctx context.Context, userID string) error {
	if s.mode != "cloud" || s.limits.MaxAPITokensPerUser <= 0 {
		return nil
	}
	var n int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM api_tokens WHERE user_id = $1", userID).Scan(&n)
	if err != nil {
		return fmt.Errorf("count api tokens: %w", err)
	}
	if n >= s.limits.MaxAPITokensPerUser {
		return ErrAPITokenLimitReached
	}
	return nil
}

// LimitMessage returns a user-facing error message for the given limit error.
func (s *LimitService) LimitMessage(err error) string {
	switch {
	case errors.Is(err, ErrProjectLimitReached):
		return fmt.Sprintf("Project limit reached (max %d per user). Delete a project or contact support.", s.limits.MaxProjectsPerUser)
	case errors.Is(err, ErrFunctionLimitReached):
		return fmt.Sprintf("Function limit reached (max %d per project). Delete a function or contact support.", s.limits.MaxFunctionsPerProject)
	case errors.Is(err, ErrAPITokenLimitReached):
		return fmt.Sprintf("API token limit reached (max %d per user). Delete a token or contact support.", s.limits.MaxAPITokensPerUser)
	default:
		return "Resource limit exceeded."
	}
}
