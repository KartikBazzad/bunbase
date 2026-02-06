package models

import "time"

// Project represents a project in the system
type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	OwnerID         string    `json:"owner_id"`
	PublicAPIKey    *string   `json:"public_api_key,omitempty"`    // Only returned on get-by-id for authorized callers; never in list
	FunctionSubdomain *string `json:"function_subdomain,omitempty"` // Optional custom function subdomain (e.g. "myproject")
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ProjectMember represents a project membership
type ProjectMember struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // owner, admin, member
	CreatedAt time.Time `json:"created_at"`
}

// ProjectWithMembers includes project and its members
type ProjectWithMembers struct {
	Project
	Members []ProjectMember `json:"members,omitempty"`
}
