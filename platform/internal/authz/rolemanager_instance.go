package authz

import (
	"context"
	"log"

	"github.com/casbin/casbin/v3/rbac"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type instanceRoleManager struct {
	instance *services.InstanceService
}

var _ rbac.RoleManager = (*instanceRoleManager)(nil)

func newInstanceRoleManager(instance *services.InstanceService) *instanceRoleManager {
	return &instanceRoleManager{instance: instance}
}

func (rm *instanceRoleManager) Clear() error { return nil }

func (rm *instanceRoleManager) AddLink(name1, name2 string, domain ...string) error { return nil }

func (rm *instanceRoleManager) BuildRelationship(name1, name2 string, domain ...string) error {
	return nil
}

func (rm *instanceRoleManager) DeleteLink(name1, name2 string, domain ...string) error { return nil }

func (rm *instanceRoleManager) HasLink(name1, name2 string, domain ...string) (bool, error) {
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

func (rm *instanceRoleManager) GetRoles(name string, domain ...string) ([]string, error) {
	roles := []string{"authenticated_user"}
	admin, err := rm.instance.IsInstanceAdmin(context.Background(), name)
	if err != nil {
		// Do not strip roles on admin check failure: treat as non-admin but still authenticated.
		log.Printf("[authz] GetRoles: userID=%s roles=%v (admin check failed: %v)", name, roles, err)
		return []string{"authenticated_user"}, nil
	}
	if admin {
		roles = append(roles, "instance_admin")
	}
	log.Printf("[authz] GetRoles: userID=%s roles=%v admin=%v", name, roles, admin)
	return roles, nil
}

func (rm *instanceRoleManager) GetUsers(name string, domain ...string) ([]string, error) {
	return nil, nil
}

func (rm *instanceRoleManager) GetImplicitRoles(name string, domain ...string) ([]string, error) {
	return rm.GetRoles(name, domain...)
}

func (rm *instanceRoleManager) GetImplicitUsers(name string, domain ...string) ([]string, error) {
	return nil, nil
}

func (rm *instanceRoleManager) GetDomains(name string) ([]string, error) {
	return nil, nil
}

func (rm *instanceRoleManager) GetAllDomains() ([]string, error) {
	return nil, nil
}

func (rm *instanceRoleManager) PrintRoles() error {
	return nil
}

func (rm *instanceRoleManager) SetLogger(logger log.Logger) {}

func (rm *instanceRoleManager) Match(str, pattern string) bool {
	return str == pattern
}

func (rm *instanceRoleManager) AddMatchingFunc(name string, fn rbac.MatchingFunc) {}

func (rm *instanceRoleManager) AddDomainMatchingFunc(name string, fn rbac.MatchingFunc) {}

func (rm *instanceRoleManager) DeleteDomain(domain string) error {
	return nil
}
