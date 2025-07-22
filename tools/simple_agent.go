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
	
	// For other queries, fall back to the OllamaAgent
	agent := NewOllamaAgent(s.llm, s.registry, s.verbose)
	return agent.Execute(ctx, query)
}