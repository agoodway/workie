package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/agoodway/workie/provider"
)

// Provider implements the Provider interface for Linear
type Provider struct {
	apiKey       string
	teamID       string
	baseURL      string
	branchPrefix map[string]string
}

// NewProvider creates a new Linear provider
func NewProvider(config map[string]interface{}) (*Provider, error) {
	p := &Provider{
		baseURL: "https://api.linear.app/graphql",
		branchPrefix: map[string]string{
			"bug":     "fix/",
			"feature": "feat/",
			"task":    "task/",
			"default": "issue/",
		},
	}

	// Extract settings
	if settings, ok := config["settings"].(map[string]interface{}); ok {
		// API Key from environment variable
		if apiKeyEnv, ok := settings["api_key_env"].(string); ok {
			p.apiKey = os.Getenv(apiKeyEnv)
		}
		
		// Team ID
		if teamID, ok := settings["team_id"].(string); ok {
			p.teamID = teamID
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
	return "linear"
}

// ValidateConfig checks if the provider is properly configured
func (p *Provider) ValidateConfig() error {
	if p.apiKey == "" {
		return fmt.Errorf("Linear API key not configured (check api_key_env setting)")
	}
	return nil
}

// IsConfigured returns true if the provider has necessary configuration
func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

// ListIssues returns a list of Linear issues
func (p *Provider) ListIssues(filter provider.ListFilter) (*provider.IssueList, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Build GraphQL query
	variables := make(map[string]interface{})
	filterParts := []string{}

	// Team filter
	if p.teamID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`team: { id: { eq: "%s" } }`, p.teamID))
	}

	// Status filter
	if filter.Status != "" {
		switch strings.ToLower(filter.Status) {
		case "open":
			filterParts = append(filterParts, `state: { type: { in: ["backlog", "unstarted", "started"] } }`)
		case "closed":
			filterParts = append(filterParts, `state: { type: { in: ["completed", "canceled"] } }`)
		case "in-progress":
			filterParts = append(filterParts, `state: { type: { eq: "started" } }`)
		}
	} else {
		// Default to non-completed issues
		filterParts = append(filterParts, `state: { type: { nin: ["completed", "canceled"] } }`)
	}

	// Assignee filter
	if filter.Assignee != "" {
		if filter.Assignee == "me" {
			filterParts = append(filterParts, `assignee: { isMe: { eq: true } }`)
		} else {
			filterParts = append(filterParts, fmt.Sprintf(`assignee: { email: { eq: "%s" } }`, filter.Assignee))
		}
	}

	// Labels filter
	if len(filter.Labels) > 0 {
		labelNames := make([]string, len(filter.Labels))
		for i, label := range filter.Labels {
			labelNames[i] = fmt.Sprintf(`"%s"`, label)
		}
		filterParts = append(filterParts, fmt.Sprintf(`labels: { name: { in: [%s] } }`, strings.Join(labelNames, ", ")))
	}

	// Build filter string
	filterStr := ""
	if len(filterParts) > 0 {
		filterStr = fmt.Sprintf("filter: { %s }", strings.Join(filterParts, ", "))
	}

	// Limit
	first := 50
	if filter.Limit > 0 && filter.Limit < 50 {
		first = filter.Limit
	}

	// Cursor for pagination
	afterStr := ""
	if filter.Cursor != "" {
		afterStr = fmt.Sprintf(`, after: "%s"`, filter.Cursor)
	}

	query := fmt.Sprintf(`
		query ListIssues {
			issues(first: %d%s, %s) {
				nodes {
					id
					identifier
					title
					description
					createdAt
					updatedAt
					url
					state {
						name
						type
					}
					assignee {
						name
						email
					}
					creator {
						name
					}
					labels {
						nodes {
							name
						}
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`, first, afterStr, filterStr)

	// Make request
	resp, err := p.makeGraphQLRequest(query, variables)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		Data struct {
			Issues struct {
				Nodes    []linearIssue `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"issues"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Linear response: %w", err)
	}

	// Convert to provider issues
	issues := make([]provider.Issue, len(result.Data.Issues.Nodes))
	for i, linearIssue := range result.Data.Issues.Nodes {
		issues[i] = p.convertIssue(linearIssue)
	}

	return &provider.IssueList{
		Issues:     issues,
		TotalCount: len(issues),
		HasMore:    result.Data.Issues.PageInfo.HasNextPage,
		NextCursor: result.Data.Issues.PageInfo.EndCursor,
	}, nil
}

// GetIssue fetches a single Linear issue
func (p *Provider) GetIssue(issueID string) (*provider.Issue, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}

	// Linear uses identifiers like "TEAM-123"
	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				createdAt
				updatedAt
				url
				state {
					name
					type
				}
				assignee {
					name
					email
				}
				creator {
					name
				}
				labels {
					nodes {
						name
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": issueID,
	}

	// Make request
	resp, err := p.makeGraphQLRequest(query, variables)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		Data struct {
			Issue *linearIssue `json:"issue"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Linear response: %w", err)
	}

	if result.Data.Issue == nil {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	issue := p.convertIssue(*result.Data.Issue)
	return &issue, nil
}

// CreateBranchName generates a branch name based on the issue
func (p *Provider) CreateBranchName(issue *provider.Issue) string {
	prefix := p.branchPrefix["default"]
	
	// Determine prefix based on issue metadata
	if stateType, ok := issue.Metadata["state_type"]; ok {
		switch stateType {
		case "backlog", "unstarted":
			prefix = p.branchPrefix["feature"]
		case "started":
			prefix = p.branchPrefix["task"]
		}
	}

	// Check labels for bug
	for _, label := range issue.Labels {
		if strings.Contains(strings.ToLower(label), "bug") {
			prefix = p.branchPrefix["bug"]
			break
		}
	}

	// Create branch name
	title := provider.SanitizeBranchName(issue.Title)
	return fmt.Sprintf("%s%s-%s", prefix, strings.ToLower(issue.ID), title)
}

// makeGraphQLRequest makes a GraphQL request to the Linear API
func (p *Provider) makeGraphQLRequest(query string, variables map[string]interface{}) ([]byte, error) {
	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("Authorization", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Linear API request failed: %w", err)
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Linear API returned status %d: %s", resp.StatusCode, body.String())
	}

	// Check for GraphQL errors
	var errorCheck struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body.Bytes(), &errorCheck); err == nil && len(errorCheck.Errors) > 0 {
		return nil, fmt.Errorf("Linear GraphQL error: %s", errorCheck.Errors[0].Message)
	}

	return body.Bytes(), nil
}

// convertIssue converts a Linear issue to a provider issue
func (p *Provider) convertIssue(linearIssue linearIssue) provider.Issue {
	labels := make([]string, len(linearIssue.Labels.Nodes))
	for i, label := range linearIssue.Labels.Nodes {
		labels[i] = label.Name
	}

	// Determine issue type based on state and labels
	issueType := "issue"
	for _, label := range labels {
		labelLower := strings.ToLower(label)
		if strings.Contains(labelLower, "bug") {
			issueType = "bug"
			break
		} else if strings.Contains(labelLower, "feature") {
			issueType = "feature"
			break
		}
	}

	metadata := map[string]string{
		"created_at": linearIssue.CreatedAt,
		"updated_at": linearIssue.UpdatedAt,
		"state_type": linearIssue.State.Type,
	}
	
	if linearIssue.Creator.Name != "" {
		metadata["creator"] = linearIssue.Creator.Name
	}
	if linearIssue.Assignee != nil && linearIssue.Assignee.Name != "" {
		metadata["assignee"] = linearIssue.Assignee.Name
	}

	return provider.Issue{
		ID:          linearIssue.Identifier,
		Title:       linearIssue.Title,
		Description: linearIssue.Description,
		Type:        issueType,
		Status:      linearIssue.State.Name,
		Labels:      labels,
		URL:         linearIssue.URL,
		Provider:    "linear",
		Metadata:    metadata,
	}
}

// Linear API types
type linearIssue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	URL         string `json:"url"`
	State       struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Assignee *struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"assignee"`
	Creator struct {
		Name string `json:"name"`
	} `json:"creator"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
}