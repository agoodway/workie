package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// AIBranchNameGenerator generates branch names using AI
type AIBranchNameGenerator struct {
	llm llms.Model
}

// NewAIBranchNameGenerator creates a new AI-powered branch name generator
func NewAIBranchNameGenerator(llm llms.Model) *AIBranchNameGenerator {
	return &AIBranchNameGenerator{
		llm: llm,
	}
}

// GenerateBranchName generates an AI-powered branch name for the given issue
func (g *AIBranchNameGenerator) GenerateBranchName(issue *Issue, branchPrefix string) (string, error) {
	// Build the prompt
	prompt := g.buildPrompt(issue, branchPrefix)
	
	// Call the AI model
	ctx := context.Background()
	response, err := g.llm.Call(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("AI model error: %w", err)
	}
	
	// Extract and clean the branch name from the response
	branchName := strings.TrimSpace(response)
	branchName = strings.Trim(branchName, "`\"'")
	
	// Validate the generated branch name
	if !strings.HasPrefix(branchName, branchPrefix) {
		// If AI didn't include the prefix, add it
		branchName = fmt.Sprintf("%s%s-%s", branchPrefix, strings.ToLower(issue.ID), branchName)
	}
	
	// Ensure it's properly sanitized
	branchName = SanitizeBranchName(branchName)
	
	// Final validation
	if len(branchName) > 63 {
		// Fallback to traditional method if AI generates too long name
		return g.fallbackBranchName(issue, branchPrefix), nil
	}
	
	return branchName, nil
}

// buildPrompt creates the AI prompt for branch name generation
func (g *AIBranchNameGenerator) buildPrompt(issue *Issue, branchPrefix string) string {
	// Prepare issue context
	issueContext := fmt.Sprintf("Issue ID: %s\nType: %s\nTitle: %s", 
		issue.ID, issue.Type, issue.Title)
	
	if issue.Description != "" {
		// Limit description length
		desc := issue.Description
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		issueContext += fmt.Sprintf("\nDescription: %s", desc)
	}
	
	if len(issue.Labels) > 0 {
		issueContext += fmt.Sprintf("\nLabels: %s", strings.Join(issue.Labels, ", "))
	}
	
	return fmt.Sprintf(`Generate a Git branch name for the following issue:

%s

Requirements:
1. Format: %s%s-{descriptive-suffix}
2. The descriptive suffix should be 2-5 words that capture the essence of the work
3. Use only lowercase letters and hyphens
4. Make it concise but descriptive
5. Total length must not exceed 63 characters
6. Focus on WHAT is being done, not HOW

Examples:
- Issue: "Users can't login with special characters in password" → fix/123-password-special-chars
- Issue: "Add dark mode toggle to settings page" → feat/456-dark-mode-settings
- Issue: "Refactor database connection pooling for better performance" → task/789-refactor-db-pooling
- Issue: "Update user documentation for API v2" → docs/101-api-v2-docs

Generate ONLY the branch name, nothing else:`, issueContext, branchPrefix, strings.ToLower(issue.ID))
}

// fallbackBranchName generates a branch name using the traditional method
func (g *AIBranchNameGenerator) fallbackBranchName(issue *Issue, branchPrefix string) string {
	suffix := SanitizeBranchName(issue.Title)
	
	// Truncate suffix to keep it concise
	words := strings.Split(suffix, "-")
	if len(words) > 5 {
		words = words[:5]
	}
	suffix = strings.Join(words, "-")
	
	branchName := fmt.Sprintf("%s%s-%s", branchPrefix, strings.ToLower(issue.ID), suffix)
	
	// Ensure total length doesn't exceed 63 characters
	if len(branchName) > 63 {
		prefixLen := len(branchPrefix) + len(issue.ID) + 1
		maxSuffixLen := 63 - prefixLen
		if maxSuffixLen > 0 && len(suffix) > maxSuffixLen {
			suffix = suffix[:maxSuffixLen]
			suffix = strings.TrimSuffix(suffix, "-")
			branchName = fmt.Sprintf("%s%s-%s", branchPrefix, strings.ToLower(issue.ID), suffix)
		}
	}
	
	return branchName
}