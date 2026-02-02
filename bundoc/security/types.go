package security

import (
	"time"
)

// Permission represents an atomic authorization grant
type Permission string

const (
	PermRead     Permission = "read"
	PermWrite    Permission = "write"
	PermAdmin    Permission = "admin"     // Collection management
	PermSuper    Permission = "superuser" // Full system access
	PermCreateDB Permission = "create_db"
	PermDropDB   Permission = "drop_db"
)

// Role defines a named set of permissions
type Role struct {
	Name        string       `json:"name"`
	Database    string       `json:"database"` // "" for global roles, or specific DB
	Permissions []Permission `json:"permissions"`
}

// User represents an authenticated entity
type User struct {
	Username       string    `json:"username"`
	HashedPassword string    `json:"hashed_password"` // SCRAM stored key or bcrypt hash (TBD)
	Salt           string    `json:"salt"`
	Roles          []Role    `json:"roles"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Default Roles
var (
	RoleRoot = Role{
		Name:        "root",
		Database:    "", // Global
		Permissions: []Permission{PermSuper},
	}
	RoleReadWrite = Role{
		Name:        "readWrite",
		Permissions: []Permission{PermRead, PermWrite},
	}
	RoleRead = Role{
		Name:        "read",
		Permissions: []Permission{PermRead},
	}
)

// HasPermission checks if the user has the required permission for a specific database
func (u *User) HasPermission(db string, perm Permission) bool {
	for _, role := range u.Roles {
		// Check global role (Superuser)
		if role.Database == "" {
			if containsPerm(role.Permissions, PermSuper) {
				return true
			}
			// Global role applies to all DBs?
			// Typically global 'read' means read on all DBs?
			// For safety, let's say Global roles apply everywhere.
			if containsPerm(role.Permissions, perm) {
				return true
			}
		}

		// Check database specific role
		if role.Database == db {
			if containsPerm(role.Permissions, perm) {
				return true
			}
			// Start of admin perm hierarchy check (Admin implies Read/Write)
			if containsPerm(role.Permissions, PermAdmin) && (perm == PermRead || perm == PermWrite) {
				return true
			}
		}
	}
	return false
}

func containsPerm(perms []Permission, target Permission) bool {
	for _, p := range perms {
		if p == target {
			return true
		}
	}
	return false
}
