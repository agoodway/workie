package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// SimpleAgent provides a simpler approach for tool calling
type SimpleAgent struct {
	llm      llms.Model
	registry *ToolRegistry
	verbose  bool
}

// NewSimpleAgent creates a new simple agent
func NewSimpleAgent(llm llms.Model, registry *ToolRegistry, verbose bool) *SimpleAgent {
	return &SimpleAgent{
		llm:      llm,
		registry: registry,
		verbose:  verbose,
	}
}

// Execute processes a query with a simplified approach
func (s *SimpleAgent) Execute(ctx context.Context, query string) (string, error) {
	// Check for common queries and handle them directly
	lowerQuery := strings.ToLower(query)
	
	// Direct handling for file listing (check this first)
	if strings.Contains(lowerQuery, "list") && (strings.Contains(lowerQuery, "file") || strings.Contains(lowerQuery, "directory")) {
		if s.verbose {
			fmt.Println("Detected list files query, using shell tool directly")
		}
		
		tool, _ := s.registry.Get("shell")
		result, err := tool.Execute(ctx, map[string]interface{}{
			"command": "ls",
			"args": []interface{}{"-la"},
		})
		
		if err != nil {
			return "", err
		}
		
		return fmt.Sprintf("Files in current directory:\n%s", result), nil
	}
	
	// Direct handling for branch queries
	if strings.Contains(lowerQuery, "branch") && strings.Contains(lowerQuery, "current") {
		if s.verbose {
			fmt.Println("Detected branch query, using git tool directly")
		}
		
		tool, _ := s.registry.Get("git")
		result, err := tool.Execute(ctx, map[string]interface{}{
			"command": "branch",
		})
		
		if err != nil {
			return "", err
		}
		
		return fmt.Sprintf("The current branch is: %s", strings.TrimSpace(result)), nil
	}
	
	// Direct handling for pwd/directory queries
	if strings.Contains(lowerQuery, "current") && (strings.Contains(lowerQuery, "directory") || strings.Contains(lowerQuery, "folder")) {
		if s.verbose {
			fmt.Println("Detected pwd query, using shell tool directly")
		}
		
		tool, _ := s.registry.Get("shell")
		result, err := tool.Execute(ctx, map[string]interface{}{
			"command": "pwd",
		})
		
		if err != nil {
			return "", err
		}
		
		return fmt.Sprintf("The current directory is: %s", strings.TrimSpace(result)), nil
	}
	
	// Direct handling for commit message generation
	if strings.Contains(lowerQuery, "commit") && strings.Contains(lowerQuery, "message") {
		if s.verbose {
			fmt.Println("Detected commit message query, using commit_message tool directly")
		}
		
		tool, exists := s.registry.Get("commit_message")
		if !exists {
			// Fallback to using git tools
			return s.generateCommitMessageWithGit(ctx)
		}
		
		// Check if user wants detailed format
		format := "conventional"
		if strings.Contains(lowerQuery, "detailed") || strings.Contains(lowerQuery, "detail") {
			format = "detailed"
		}
		
		result, err := tool.Execute(ctx, map[string]interface{}{
			"type": "all",
			"format": format,
		})
		
		if err != nil {
			return "", err
		}
		
		return fmt.Sprintf("Suggested commit message:\n\n%s", result), nil
	}
	
	// For other queries, fall back to the OllamaAgent
	agent := NewOllamaAgent(s.llm, s.registry, s.verbose)
	return agent.Execute(ctx, query)
}

// generateCommitMessageWithGit uses git tools to analyze changes
func (s *SimpleAgent) generateCommitMessageWithGit(ctx context.Context) (string, error) {
	gitTool, _ := s.registry.Get("git")
	
	// Get status
	statusResult, err := gitTool.Execute(ctx, map[string]interface{}{
		"command": "status",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %v", err)
	}
	
	// Get diff
	diffResult, err := gitTool.Execute(ctx, map[string]interface{}{
		"command": "diff",
		"args": []interface{}{"--stat"},
	})
	if err != nil {
		// Try staged diff
		diffResult, _ = gitTool.Execute(ctx, map[string]interface{}{
			"command": "diff",
			"args": []interface{}{"--cached", "--stat"},
		})
	}
	
	// Combine results
	var message strings.Builder
	message.WriteString("Based on the current changes:\n\n")
	message.WriteString("Status:\n")
	message.WriteString(statusResult)
	message.WriteString("\n\nChanges:\n")
	message.WriteString(diffResult)
	message.WriteString("\n\nTo create a commit message:\n")
	message.WriteString("1. Stage your changes: git add <files>\n")
	message.WriteString("2. Create a descriptive commit message based on the changes above\n")
	message.WriteString("3. Commit: git commit -m \"your message\"\n")
	
	return message.String(), nil
}