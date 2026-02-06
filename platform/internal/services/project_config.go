package services

import (
	"fmt"

	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// ProjectConfig holds client-ready config for a project's services (gateway URL + path patterns).
type ProjectConfig struct {
	GatewayURL  string          `json:"gateway_url"`
	ProjectID   string          `json:"project_id"`
	ProjectSlug string          `json:"project_slug"`
	KV          KVConfig        `json:"kv"`
	Bundoc      BundocConfig    `json:"bundoc"`
	Buncast     BuncastConfig   `json:"buncast"`
	Functions   FunctionsConfig `json:"functions"`
}

// KVConfig is the Bunder KV config (path behind Traefik).
type KVConfig struct {
	Path string `json:"path"`
}

// BundocConfig is the Bundoc documents path.
type BundocConfig struct {
	DocumentsPath string `json:"documents_path"`
}

// BuncastConfig is the Buncast topic prefix.
type BuncastConfig struct {
	TopicPrefix string `json:"topic_prefix"`
}

// FunctionsConfig holds function invocation paths and URLs.
type FunctionsConfig struct {
	InvokePath   string `json:"invoke_path"`
	SubdomainURL string `json:"subdomain_url,omitempty"`
	GeneratedURL string `json:"generated_url,omitempty"`
}

// ProjectConfigService builds project config from project + gateway URL.
type ProjectConfigService struct {
	gatewayURL string
}

// NewProjectConfigService creates a ProjectConfigService. gatewayURL is the Traefik base URL (e.g. https://api.example.com).
func NewProjectConfigService(gatewayURL string) *ProjectConfigService {
	return &ProjectConfigService{gatewayURL: gatewayURL}
}

// GetConfig builds client-ready config for the project. Gateway URL is trimmed of trailing slash.
func (s *ProjectConfigService) GetConfig(project *models.Project) *ProjectConfig {
	base := s.gatewayURL
	if len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}

	// Derived function URLs
	subdomainURL := ""
	if project.FunctionSubdomain != nil && *project.FunctionSubdomain != "" {
		subdomainURL = fmt.Sprintf("https://%s.bunbase.com", *project.FunctionSubdomain)
	}
	// Generated URL format: {projectID}.functions.bunbase.com
	generatedHost := fmt.Sprintf("%s.functions.bunbase.com", project.ID)
	generatedURL := fmt.Sprintf("https://%s", generatedHost)

	return &ProjectConfig{
		GatewayURL:  base,
		ProjectID:   project.ID,
		ProjectSlug: project.Slug,
		KV: KVConfig{
			Path: fmt.Sprintf("%s/kv/%s", base, project.ID),
		},
		Bundoc: BundocConfig{
			DocumentsPath: fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents", base, project.ID),
		},
		Buncast: BuncastConfig{
			TopicPrefix: fmt.Sprintf("project.%s.", project.ID),
		},
		Functions: FunctionsConfig{
			InvokePath:   fmt.Sprintf("%s/invoke", base),
			SubdomainURL: subdomainURL,
			GeneratedURL: generatedURL,
		},
	}
}
