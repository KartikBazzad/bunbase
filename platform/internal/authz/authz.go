package authz

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v3"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

//go:embed model_instance.conf policy_instance.csv model_project.conf policy_project.csv
var embedFS embed.FS

// Enforcer holds instance and project enforcers and exposes InstanceEnforce and ProjectEnforce.
type Enforcer struct {
	instance        *casbin.Enforcer
	project         *casbin.Enforcer
	instanceService *services.InstanceService
	projectService  *services.ProjectService
}

// NewEnforcer creates both enforcers with custom role managers. Embedded model and policy files are used.
func NewEnforcer(instanceService *services.InstanceService, projectService *services.ProjectService) (*Enforcer, error) {
	dir, err := os.MkdirTemp("", "bunbase-casbin-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if err := writeEmbedToDir(dir, "model_instance.conf", "policy_instance.csv", "model_project.conf", "policy_project.csv"); err != nil {
		return nil, err
	}

	instanceEnforcer, err := casbin.NewEnforcer(
		filepath.Join(dir, "model_instance.conf"),
		filepath.Join(dir, "policy_instance.csv"),
	)
	if err != nil {
		return nil, err
	}
	instanceEnforcer.SetRoleManager(newInstanceRoleManager(instanceService))

	projectEnforcer, err := casbin.NewEnforcer(
		filepath.Join(dir, "model_project.conf"),
		filepath.Join(dir, "policy_project.csv"),
	)
	if err != nil {
		return nil, err
	}
	projectEnforcer.SetRoleManager(newProjectRoleManager(projectService))

	return &Enforcer{
		instance:        instanceEnforcer,
		project:         projectEnforcer,
		instanceService: instanceService,
		projectService:  projectService,
	}, nil
}

func writeEmbedToDir(dir string, names ...string) error {
	for _, name := range names {
		data, err := embedFS.ReadFile(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			return err
		}
	}
	return nil
}

// InstanceEnforce checks if user can perform action in the given deployment mode.
// action e.g. "create_project"; deploymentMode "cloud" or "self_hosted".
func (e *Enforcer) InstanceEnforce(userID, action, deploymentMode string) (bool, error) {
	// Determine the user's role based on instance admin status
	role := "authenticated_user"

	// Check if user is instance admin (only relevant for self_hosted, but we check anyway)
	if e.instanceService != nil {
		isAdmin, err := e.instanceService.IsInstanceAdmin(context.Background(), userID)
		if err != nil {
			log.Printf("[authz] InstanceEnforce: failed to check admin status for userID=%s: %v", userID, err)
			// Continue with authenticated_user role on error
		} else if isAdmin {
			role = "instance_admin"
		}
	}

	log.Printf("[authz] InstanceEnforce: userID=%s role=%s action=%s deploymentMode=%s", userID, role, action, deploymentMode)

	// Enforce with the role, not the userID
	allowed, err := e.instance.Enforce(role, action, "allow", deploymentMode)
	log.Printf("[authz] InstanceEnforce: role=%s action=%s deploymentMode=%s allowed=%v err=%v", role, action, deploymentMode, allowed, err)
	return allowed, err
}

// AddInstanceAdminPolicy adds a policy for an instance admin user.
// This should be called during setup or when granting admin privileges.
func (e *Enforcer) AddInstanceAdminPolicy(userID string) error {
	_, err := e.instance.AddPolicy(userID, "create_project", "allow", "self_hosted")
	return err
}

// RemoveInstanceAdminPolicy removes admin policies for a user.
func (e *Enforcer) RemoveInstanceAdminPolicy(userID string) error {
	_, err := e.instance.RemovePolicy(userID, "create_project", "allow", "self_hosted")
	return err
}

// ProjectEnforce checks if user can perform action on resource in project.
// resource e.g. "project", "function", "database", "config"; action e.g. "read", "update", "delete", "deploy", "regenerate_key".
func (e *Enforcer) ProjectEnforce(userID, projectID, resource, action string) (bool, error) {
	// Determine the user's role in this project
	role := ""

	if e.projectService != nil {
		userRole, found, err := e.projectService.GetRoleInProject(context.Background(), projectID, userID)
		if err != nil {
			log.Printf("[authz] ProjectEnforce: failed to get role for userID=%s projectID=%s: %v", userID, projectID, err)
			return false, err
		}
		if found {
			role = userRole
		}
	}

	// If no role found, user has no access to this project
	if role == "" {
		log.Printf("[authz] ProjectEnforce: userID=%s has no role in projectID=%s", userID, projectID)
		return false, nil
	}

	log.Printf("[authz] ProjectEnforce: userID=%s projectID=%s role=%s resource=%s action=%s", userID, projectID, role, resource, action)

	// Enforce with the role (domain is still needed in request even though we don't use it in matcher)
	allowed, err := e.project.Enforce(role, projectID, resource, action)
	log.Printf("[authz] ProjectEnforce: role=%s resource=%s action=%s allowed=%v err=%v", role, resource, action, allowed, err)
	return allowed, err
}
