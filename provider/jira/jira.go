package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/agoodway/workie/provider"
)

// Provider implements the Provider interface for Jira
type Provider struct {
	baseURL      string
	email        string
	apiToken     string
	project      string
	branchPrefix map[string]string
}

// NewProvider creates a new Jira provider
func NewProvider(config map[string]interface{}) (*Provider, error) {
	p := &Provider{
		branchPrefix: map[string]string{
			"bug":     "fix/",
			"story":   "feat/",
			"task":    "task/",
			"default": "issue/",
		},
	}

	// Extract settings
	if settings, ok := config["settings"].(map[string]interface{}); ok {
		// Base URL (required)
		if baseURL, ok := settings["base_url"].(string); ok {
			p.baseURL = strings.TrimRight(baseURL, "/")
		}

		// Authentication
		if emailEnv, ok := settings["email_env"].(string); ok {
			p.email = os.Getenv(emailEnv)
		}
		if tokenEnv, ok := settings["api_token_env"].(string); ok {
			p.apiToken = os.Getenv(tokenEnv)
		}

		// Project key
		if project, ok := settings["project"].(string); ok {
			p.project = project
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
	return "jira"
}

// ValidateConfig checks if the provider is properly configured
func (p *Provider) ValidateConfig() error {
	if p.baseURL == "" {
		return fmt.Errorf("Jira base URL not configured")
	}
	if p.email == "" {
		return fmt.Errorf("Jira email not configured (check email_env setting)")
	}
	if p.apiToken == "" {
		return fmt.Errorf("Jira API token not configured (check api_token_env setting)")
	}
	if p.project == "" {
		return fmt.Errorf("Jira project key not configured")
	}
	return nil
}

// IsConfigured returns true if the provider has necessary configuration
func (p *Provider) IsConfigured() bool {
	return p.baseURL != "" && p.email != "" && p.apiToken != "" && p.project != ""
}

// ListIssues returns a list of Jira issues
func (p *Provider) ListIssues(filter provider.ListFilter) (*provider.IssueList, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Build JQL query
	jql := fmt.Sprintf("project = %s", p.project)

	// Status filter
	if filter.Status != "" {
		switch strings.ToLower(filter.Status) {
		case "open":
			jql += " AND status != Done AND status != Closed"
		case "closed":
			jql += " AND (status = Done OR status = Closed)"
		case "in-progress":
			jql += " AND status = 'In Progress'"
		}
	} else {
		// Default to non-closed issues
		jql += " AND status != Done AND status != Closed"
	}

	// Assignee filter
	if filter.Assignee != "" {
		if filter.Assignee == "me" {
			jql += " AND assignee = currentUser()"
		} else {
			jql += fmt.Sprintf(" AND assignee = '%s'", filter.Assignee)
		}
	}

	// Labels filter
	if len(filter.Labels) > 0 {
		labelConditions := make([]string, len(filter.Labels))
		for i, label := range filter.Labels {
			labelConditions[i] = fmt.Sprintf("labels = '%s'", label)
		}
		jql += fmt.Sprintf(" AND (%s)", strings.Join(labelConditions, " OR "))
	}

	// Type filter
	if filter.Type != "" {
		jql += fmt.Sprintf(" AND issuetype = '%s'", filter.Type)
	}

	// Free text search
	if filter.Query != "" {
		jql += fmt.Sprintf(" AND text ~ '%s'", filter.Query)
	}

	// Order by updated date
	jql += " ORDER BY updated DESC"

	// Set max results
	maxResults := 50
	if filter.Limit > 0 && filter.Limit < maxResults {
		maxResults = filter.Limit
	}

	startAt := 0
	if filter.Cursor != "" {
		fmt.Sscanf(filter.Cursor, "%d", &startAt)
	}

	// Make request
	url := fmt.Sprintf("%s/rest/api/3/search", p.baseURL)
	params := map[string]string{
		"jql":        jql,
		"maxResults": fmt.Sprintf("%d", maxResults),
		"startAt":    fmt.Sprintf("%d", startAt),
		"fields":     "key,summary,description,issuetype,status,labels,created,updated,reporter,assignee",
	}

	resp, err := p.makeRequest("GET", url, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var searchResult jiraSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse Jira response: %w", err)
	}

	// Convert to provider issues
	issues := make([]provider.Issue, len(searchResult.Issues))
	for i, jiraIssue := range searchResult.Issues {
		issues[i] = p.convertIssue(jiraIssue)
	}

	// Check if there are more results
	hasMore := searchResult.StartAt+len(searchResult.Issues) < searchResult.Total
	nextCursor := ""
	if hasMore {
		nextCursor = fmt.Sprintf("%d", searchResult.StartAt+len(searchResult.Issues))
	}

	return &provider.IssueList{
		Issues:     issues,
		TotalCount: searchResult.Total,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// GetIssue fetches a single Jira issue
func (p *Provider) GetIssue(issueID string) (*provider.Issue, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Validate issue ID format (should be PROJECT-123)
	if !strings.Contains(issueID, "-") {
		// If just a number is provided, prepend the project key
		issueID = fmt.Sprintf("%s-%s", p.project, issueID)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s", p.baseURL, issueID)

	resp, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jiraIssue jiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&jiraIssue); err != nil {
		return nil, fmt.Errorf("failed to parse Jira response: %w", err)
	}

	issue := p.convertIssue(jiraIssue)
	return &issue, nil
}

// CreateBranchName generates a branch name based on the issue
func (p *Provider) CreateBranchName(issue *provider.Issue) string {
	prefix := p.branchPrefix["default"]

	// Use issue type to determine prefix
	issueTypeLower := strings.ToLower(issue.Type)
	if strings.Contains(issueTypeLower, "bug") {
		prefix = p.branchPrefix["bug"]
	} else if strings.Contains(issueTypeLower, "story") {
		prefix = p.branchPrefix["story"]
	} else if strings.Contains(issueTypeLower, "task") {
		prefix = p.branchPrefix["task"]
	}

	// Create branch name
	title := provider.SanitizeBranchName(issue.Title)
	return fmt.Sprintf("%s%s-%s", prefix, strings.ToLower(issue.ID), title)
}

// makeRequest makes an HTTP request to the Jira API
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
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.email, p.apiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Jira API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Jira API returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// convertIssue converts a Jira issue to a provider issue
func (p *Provider) convertIssue(jiraIssue jiraIssue) provider.Issue {
	labels := make([]string, len(jiraIssue.Fields.Labels))
	copy(labels, jiraIssue.Fields.Labels)

	// Extract description (handle different formats)
	description := ""
	if jiraIssue.Fields.Description != nil {
		// Try to extract text from ADF format
		if content, ok := jiraIssue.Fields.Description.(map[string]interface{}); ok {
			description = extractTextFromADF(content)
		} else if desc, ok := jiraIssue.Fields.Description.(string); ok {
			description = desc
		}
	}

	metadata := map[string]string{
		"created_at": jiraIssue.Fields.Created,
		"updated_at": jiraIssue.Fields.Updated,
	}

	if jiraIssue.Fields.Reporter != nil {
		metadata["reporter"] = jiraIssue.Fields.Reporter.DisplayName
	}
	if jiraIssue.Fields.Assignee != nil {
		metadata["assignee"] = jiraIssue.Fields.Assignee.DisplayName
	}

	return provider.Issue{
		ID:          jiraIssue.Key,
		Title:       jiraIssue.Fields.Summary,
		Description: description,
		Type:        jiraIssue.Fields.IssueType.Name,
		Status:      jiraIssue.Fields.Status.Name,
		Labels:      labels,
		URL:         fmt.Sprintf("%s/browse/%s", p.baseURL, jiraIssue.Key),
		Provider:    "jira",
		Metadata:    metadata,
	}
}

// extractTextFromADF extracts plain text from Atlassian Document Format
func extractTextFromADF(adf map[string]interface{}) string {
	var texts []string

	if content, ok := adf["content"].([]interface{}); ok {
		for _, node := range content {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if nodeType, ok := nodeMap["type"].(string); ok && nodeType == "paragraph" {
					if nodeContent, ok := nodeMap["content"].([]interface{}); ok {
						for _, textNode := range nodeContent {
							if textMap, ok := textNode.(map[string]interface{}); ok {
								if text, ok := textMap["text"].(string); ok {
									texts = append(texts, text)
								}
							}
						}
					}
				}
			}
		}
	}

	return strings.Join(texts, "\n")
}

// Jira API types
type jiraSearchResult struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Issues     []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string     `json:"key"`
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Summary     string        `json:"summary"`
	Description interface{}   `json:"description"` // Can be string or ADF object
	IssueType   jiraIssueType `json:"issuetype"`
	Status      jiraStatus    `json:"status"`
	Labels      []string      `json:"labels"`
	Created     string        `json:"created"`
	Updated     string        `json:"updated"`
	Reporter    *jiraUser     `json:"reporter"`
	Assignee    *jiraUser     `json:"assignee"`
}

type jiraIssueType struct {
	Name string `json:"name"`
}

type jiraStatus struct {
	Name string `json:"name"`
}

type jiraUser struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}
