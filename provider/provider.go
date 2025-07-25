package provider

import (
	"fmt"
	"strings"
)

// Issue represents a single issue from any provider
type Issue struct {
	ID          string            // Provider-specific ID (e.g., "123" for GitHub, "PROJ-123" for Jira)
	Title       string            // Issue title
	Description string            // Issue description/body
	Type        string            // Issue type (bug, feature, task, etc.)
	Status      string            // Current status
	Labels      []string          // Labels/tags
	URL         string            // Web URL to the issue
	Provider    string            // Provider name (github, jira, linear)
	Metadata    map[string]string // Provider-specific metadata
}

// IssueList represents a list of issues with pagination info
type IssueList struct {
	Issues     []Issue
	TotalCount int
	HasMore    bool
	NextCursor string // For cursor-based pagination
}

// Provider defines the interface for issue tracking providers
type Provider interface {
	// Name returns the provider name (e.g., "github", "jira", "linear")
	Name() string

	// ListIssues returns a list of issues based on the filter criteria
	ListIssues(filter ListFilter) (*IssueList, error)

	// GetIssue fetches a single issue by ID
	GetIssue(issueID string) (*Issue, error)

	// CreateBranchName generates a branch name based on the issue
	CreateBranchName(issue *Issue) string

	// ValidateConfig checks if the provider is properly configured
	ValidateConfig() error

	// IsConfigured returns true if the provider has necessary configuration
	IsConfigured() bool
}

// ListFilter defines filtering options for listing issues
type ListFilter struct {
	Status   string   // Filter by status (open, closed, in-progress, etc.)
	Assignee string   // Filter by assignee
	Labels   []string // Filter by labels
	Type     string   // Filter by issue type
	Limit    int      // Maximum number of issues to return
	Cursor   string   // Pagination cursor
	Query    string   // Free-text search query
}

// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	Enabled      bool                   `yaml:"enabled"`
	Type         string                 `yaml:"type"` // github, jira, linear
	BranchPrefix map[string]string      `yaml:"branch_prefix,omitempty"`
	Settings     map[string]interface{} `yaml:"settings,omitempty"`
}

// Registry manages available providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) error {
	name := provider.Name()
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ListConfigured returns names of configured providers
func (r *Registry) ListConfigured() []string {
	names := make([]string, 0, len(r.providers))
	for name, provider := range r.providers {
		if provider.IsConfigured() {
			names = append(names, name)
		}
	}
	return names
}

// ParseIssueReference parses a reference like "github:123" or "jira:PROJ-123"
func ParseIssueReference(ref string) (provider, issueID string, err error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid issue reference format: expected 'provider:id', got '%s'", ref)
	}

	provider = strings.ToLower(strings.TrimSpace(parts[0]))
	issueID = strings.TrimSpace(parts[1])

	if provider == "" || issueID == "" {
		return "", "", fmt.Errorf("invalid issue reference: provider and ID cannot be empty")
	}

	return provider, issueID, nil
}

// SanitizeBranchName cleans up a string to be safe for use as a git branch name
func SanitizeBranchName(name string) string {
	// Replace spaces and special characters with hyphens
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		".", "-",
		",", "-",
		";", "-",
		"'", "-",
		"`", "-",
		"~", "-",
		"!", "-",
		"@", "-",
		"#", "-",
		"$", "-",
		"%", "-",
		"^", "-",
		"&", "-",
		"(", "-",
		")", "-",
		"[", "-",
		"]", "-",
		"{", "-",
		"}", "-",
		"=", "-",
		"+", "-",
	)

	name = replacer.Replace(name)

	// Replace multiple consecutive hyphens with a single hyphen
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Remove leading and trailing hyphens
	name = strings.Trim(name, "-")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Limit length to 63 characters (git branch name limit is 255, but let's be conservative)
	if len(name) > 63 {
		name = name[:63]
		// Remove trailing hyphen if truncation created one
		name = strings.TrimRight(name, "-")
	}

	return name
}
