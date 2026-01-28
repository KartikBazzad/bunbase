package models

import "time"

// Function represents a function linked to a project
type Function struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	FunctionServiceID string    `json:"function_service_id"` // ID in functions service
	Name              string    `json:"name"`
	Runtime           string    `json:"runtime"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
