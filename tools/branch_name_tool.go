package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/agoodway/workie/provider"
)

// BranchNameTool generates intelligent branch names from issue details
type BranchNameTool struct{}

// NewBranchNameTool creates a new branch name generation tool
func NewBranchNameTool() *BranchNameTool {
	return &BranchNameTool{}
}

// Name returns the tool name
func (t *BranchNameTool) Name() string {
	return "generate_branch_name"
}

// Description returns the tool description
func (t *BranchNameTool) Description() string {
	return "Generate a concise, descriptive Git branch name from issue details"
}

// Parameters returns the JSON schema for the tool parameters
func (t *BranchNameTool) Parameters() string {
	return `{
		"type": "object",
		"properties": {
			"issue_id": {
				"type": "string",
				"description": "The issue ID (e.g., 123, PROJ-456)"
			},
			"issue_title": {
				"type": "string",
				"description": "The issue title"
			},
			"issue_description": {
				"type": "string",
				"description": "The issue description or body"
			},
			"issue_type": {
				"type": "string",
				"description": "The issue type (bug, feature, task, etc.)"
			},
			"issue_labels": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"description": "Issue labels or tags"
			},
			"branch_prefix": {
				"type": "string",
				"description": "The prefix to use for the branch (e.g., fix/, feat/, task/)"
			}
		},
		"required": ["issue_id", "issue_title", "issue_type"]
	}`
}

// Execute generates a branch name based on the issue details
func (t *BranchNameTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// Extract parameters
	issueID, _ := params["issue_id"].(string)
	issueTitle, _ := params["issue_title"].(string)
	issueDescription, _ := params["issue_description"].(string)
	issueType, _ := params["issue_type"].(string)
	branchPrefix, _ := params["branch_prefix"].(string)

	// Extract labels
	var labels []string
	if labelsRaw, ok := params["issue_labels"].([]interface{}); ok {
		for _, label := range labelsRaw {
			if str, ok := label.(string); ok {
				labels = append(labels, str)
			}
		}
	}

	// Build context for AI
	context := fmt.Sprintf(`Generate a concise Git branch name based on this issue:
Issue ID: %s
Type: %s
Title: %s`, issueID, issueType, issueTitle)

	if issueDescription != "" {
		// Limit description length to avoid overwhelming the AI
		desc := issueDescription
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		context += fmt.Sprintf("\nDescription: %s", desc)
	}

	if len(labels) > 0 {
		context += fmt.Sprintf("\nLabels: %s", strings.Join(labels, ", "))
	}

	// Build prompt for future AI integration
	_ = fmt.Sprintf(`%s

Rules for the branch name:
1. Use the format: %s%s-{descriptive-suffix}
2. The descriptive suffix should be 2-5 words, hyphen-separated
3. Use lowercase letters only
4. No special characters except hyphens
5. Make it concise but descriptive of the actual work
6. The suffix should capture the essence of what's being fixed/implemented
7. Total branch name length should not exceed 63 characters

Examples:
- For "Users can't login with special chars in password" → "fix/123-password-special-chars"
- For "Add dark mode toggle to settings" → "feat/456-dark-mode-settings"
- For "Refactor database connection pooling" → "task/789-refactor-db-pooling"

Generate only the branch name, nothing else.`, context, branchPrefix, strings.ToLower(issueID))

	// In a real implementation, this would call the AI model
	// For now, we'll return a generated branch name based on the title
	suffix := provider.SanitizeBranchName(issueTitle)

	// Truncate suffix to keep it concise
	words := strings.Split(suffix, "-")
	if len(words) > 5 {
		words = words[:5]
	}
	suffix = strings.Join(words, "-")

	branchName := fmt.Sprintf("%s%s-%s", branchPrefix, strings.ToLower(issueID), suffix)

	// Ensure total length doesn't exceed 63 characters
	if len(branchName) > 63 {
		// Calculate how much we need to truncate the suffix
		prefixLen := len(branchPrefix) + len(issueID) + 1 // +1 for the hyphen
		maxSuffixLen := 63 - prefixLen
		if maxSuffixLen > 0 {
			if len(suffix) > maxSuffixLen {
				suffix = suffix[:maxSuffixLen]
				// Remove trailing hyphen if any
				suffix = strings.TrimSuffix(suffix, "-")
			}
			branchName = fmt.Sprintf("%s%s-%s", branchPrefix, strings.ToLower(issueID), suffix)
		}
	}

	// For now, return the generated branch name
	// In a full integration, this would include the AI response
	return branchName, nil
}
