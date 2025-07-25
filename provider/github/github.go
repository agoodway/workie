package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/agoodway/workie/provider"
)

// Provider implements the Provider interface for GitHub
type Provider struct {
	token        string
	owner        string
	repo         string
	baseURL      string
	branchPrefix map[string]string
}

// NewProvider creates a new GitHub provider
func NewProvider(config map[string]interface{}) (*Provider, error) {
	p := &Provider{
		baseURL: "https://api.github.com",
		branchPrefix: map[string]string{
			"bug":     "fix/",
			"feature": "feat/",
			"default": "issue/",
		},
	}

	// Extract settings
	if settings, ok := config["settings"].(map[string]interface{}); ok {
		// Token from environment variable
		if tokenEnv, ok := settings["token_env"].(string); ok {
			p.token = os.Getenv(tokenEnv)
		}

		// Repository information
		if owner, ok := settings["owner"].(string); ok {
			p.owner = owner
		}
		if repo, ok := settings["repo"].(string); ok {
			p.repo = repo
		}

		// Custom base URL for GitHub Enterprise
		if baseURL, ok := settings["base_url"].(string); ok {
			p.baseURL = strings.TrimRight(baseURL, "/")
		}
	}

	// Branch prefixes
	if prefixes, ok := config["branch_prefix"].(map[string]interface{}); ok {
		for key, value := range prefixes {
			if prefix, ok := value.(string); ok {
				p.branchPrefix[key] = prefix
			}
		}
	}

	return p, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "github"
}

// ValidateConfig checks if the provider is properly configured
func (p *Provider) ValidateConfig() error {
	if p.token == "" {
		return fmt.Errorf("GitHub token not configured (check token_env setting)")
	}
	if p.owner == "" {
		return fmt.Errorf("GitHub repository owner not configured")
	}
	if p.repo == "" {
		return fmt.Errorf("GitHub repository name not configured")
	}
	return nil
}

// IsConfigured returns true if the provider has necessary configuration
func (p *Provider) IsConfigured() bool {
	return p.token != "" && p.owner != "" && p.repo != ""
}

// ListIssues returns a list of GitHub issues
func (p *Provider) ListIssues(filter provider.ListFilter) (*provider.IssueList, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Build query parameters
	params := make(map[string]string)

	// Status mapping
	if filter.Status != "" {
		switch strings.ToLower(filter.Status) {
		case "open":
			params["state"] = "open"
		case "closed":
			params["state"] = "closed"
		default:
			params["state"] = "all"
		}
	} else {
		params["state"] = "open" // Default to open issues
	}

	// Assignee
	if filter.Assignee != "" {
		params["assignee"] = filter.Assignee
	}

	// Labels
	if len(filter.Labels) > 0 {
		params["labels"] = strings.Join(filter.Labels, ",")
	}

	// Limit
	perPage := 30
	if filter.Limit > 0 && filter.Limit < 100 {
		perPage = filter.Limit
	}
	params["per_page"] = strconv.Itoa(perPage)

	// Pagination
	page := 1
	if filter.Cursor != "" {
		if p, err := strconv.Atoi(filter.Cursor); err == nil {
			page = p
		}
	}
	params["page"] = strconv.Itoa(page)

	// Build URL
	url := fmt.Sprintf("%s/repos/%s/%s/issues", p.baseURL, p.owner, p.repo)

	// Make request
	resp, err := p.makeRequest("GET", url, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var ghIssues []githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Convert to provider issues
	issues := make([]provider.Issue, 0, len(ghIssues))
	for _, ghIssue := range ghIssues {
		// Skip pull requests
		if ghIssue.PullRequest != nil {
			continue
		}

		issues = append(issues, p.convertIssue(ghIssue))
	}

	// Check if there are more pages
	hasMore := len(ghIssues) == perPage
	nextCursor := ""
	if hasMore {
		nextCursor = strconv.Itoa(page + 1)
	}

	return &provider.IssueList{
		Issues:     issues,
		TotalCount: len(issues),
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// GetIssue fetches a single GitHub issue
func (p *Provider) GetIssue(issueID string) (*provider.Issue, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Validate issue ID is a number
	if _, err := strconv.Atoi(issueID); err != nil {
		return nil, fmt.Errorf("invalid GitHub issue ID: %s (must be a number)", issueID)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", p.baseURL, p.owner, p.repo, issueID)

	resp, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghIssue githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&ghIssue); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Check if it's a pull request
	if ghIssue.PullRequest != nil {
		return nil, fmt.Errorf("ID %s is a pull request, not an issue", issueID)
	}

	issue := p.convertIssue(ghIssue)
	return &issue, nil
}

// CreateBranchName generates a branch name based on the issue
func (p *Provider) CreateBranchName(issue *provider.Issue) string {
	prefix := p.branchPrefix["default"]

	// Try to determine issue type from labels
	for _, label := range issue.Labels {
		labelLower := strings.ToLower(label)
		if strings.Contains(labelLower, "bug") || strings.Contains(labelLower, "fix") {
			prefix = p.branchPrefix["bug"]
			break
		} else if strings.Contains(labelLower, "feature") || strings.Contains(labelLower, "enhancement") {
			prefix = p.branchPrefix["feature"]
			break
		}
	}

	// Create branch name
	title := provider.SanitizeBranchName(issue.Title)
	return fmt.Sprintf("%s%s-%s", prefix, issue.ID, title)
}

// makeRequest makes an HTTP request to the GitHub API
func (p *Provider) makeRequest(method, url string, params map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Add headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "workie/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "token "+p.token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// convertIssue converts a GitHub issue to a provider issue
func (p *Provider) convertIssue(ghIssue githubIssue) provider.Issue {
	labels := make([]string, len(ghIssue.Labels))
	for i, label := range ghIssue.Labels {
		labels[i] = label.Name
	}

	// Determine issue type from labels
	issueType := "issue"
	for _, label := range labels {
		labelLower := strings.ToLower(label)
		if strings.Contains(labelLower, "bug") {
			issueType = "bug"
			break
		} else if strings.Contains(labelLower, "feature") || strings.Contains(labelLower, "enhancement") {
			issueType = "feature"
			break
		}
	}

	return provider.Issue{
		ID:          strconv.Itoa(ghIssue.Number),
		Title:       ghIssue.Title,
		Description: ghIssue.Body,
		Type:        issueType,
		Status:      ghIssue.State,
		Labels:      labels,
		URL:         ghIssue.HTMLURL,
		Provider:    "github",
		Metadata: map[string]string{
			"created_at": ghIssue.CreatedAt,
			"updated_at": ghIssue.UpdatedAt,
			"author":     ghIssue.User.Login,
		},
	}
}

// GitHub API types
type githubIssue struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	State       string           `json:"state"`
	HTMLURL     string           `json:"html_url"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
	User        githubUser       `json:"user"`
	Labels      []githubLabel    `json:"labels"`
	PullRequest *json.RawMessage `json:"pull_request,omitempty"`
}

type githubUser struct {
	Login string `json:"login"`
}

type githubLabel struct {
	Name string `json:"name"`
}
