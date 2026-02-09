package authz

import (
	"context"

	"github.com/casbin/casbin/v3/log"
	"github.com/casbin/casbin/v3/rbac"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type projectRoleManager struct {
	project *services.ProjectService
}

var _ rbac.RoleManager = (*projectRoleManager)(nil)

func newProjectRoleManager(project *services.ProjectService) *projectRoleManager {
	return &projectRoleManager{project: project}
}

func (rm *projectRoleManager) Clear() error { return nil }

func (rm *projectRoleManager) AddLink(name1, name2 string, domain ...string) error { return nil }

func (rm *projectRoleManager) BuildRelationship(name1, name2 string, domain ...string) error { return nil }

func (rm *projectRoleManager) DeleteLink(name1, name2 string, domain ...string) error { return nil }

func (rm *projectRoleManager) HasLink(name1, name2 string, domain ...string) (bool, error) {
	roles, err := rm.GetRoles(name1, domain...)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == name2 {
			return true, nil
		}
	}
	return false, nil
}

func (rm *projectRoleManager) GetRoles(name string, domain ...string) ([]string, error) {
	if len(domain) < 1 {
		return nil, nil
	}
	projectID := domain[0]
	role, found, err := rm.project.GetRoleInProject(context.Background(), projectID, name)
	if err != nil || !found {
		return nil, nil
	}
	return []string{role}, nil
}

func (rm *projectRoleManager) GetUsers(name string, domain ...string) ([]string, error) {
	return nil, nil
}

func (rm *projectRoleManager) GetImplicitRoles(name string, domain ...string) ([]string, error) {
	return rm.GetRoles(name, domain...)
}

func (rm *projectRoleManager) GetImplicitUsers(name string, domain ...string) ([]string, error) {
	return nil, nil
}

func (rm *projectRoleManager) GetDomains(name string) ([]string, error) {
	return nil, nil
}

func (rm *projectRoleManager) GetAllDomains() ([]string, error) {
	return nil, nil
}

func (rm *projectRoleManager) PrintRoles() error {
	return nil
}

func (rm *projectRoleManager) SetLogger(logger log.Logger) {}

func (rm *projectRoleManager) Match(str, pattern string) bool {
	return str == pattern
}

func (rm *projectRoleManager) AddMatchingFunc(name string, fn rbac.MatchingFunc) {}

func (rm *projectRoleManager) AddDomainMatchingFunc(name string, fn rbac.MatchingFunc) {}

func (rm *projectRoleManager) DeleteDomain(domain string) error {
	return nil
}
