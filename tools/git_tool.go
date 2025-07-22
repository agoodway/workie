package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitTool provides Git operations
type GitTool struct{}

// NewGitTool creates a new Git tool
func NewGitTool() *GitTool {
	return &GitTool{}
}

// Name returns the name of the tool
func (g *GitTool) Name() string {
	return "git"
}

// Description returns what the tool does
func (g *GitTool) Description() string {
	return "Execute Git commands to get repository information. Use 'branch' command to get current branch name, 'status' for repository status, 'log' for commit history"
}

// Parameters returns the JSON schema for the tool's parameters
func (g *GitTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The git subcommand to execute (e.g., 'branch', 'status', 'log')",
				"enum":        []string{"branch", "status", "log", "remote", "diff", "show"},
			},
			"args": map[string]interface{}{
				"type":        "array",
				"description": "Additional arguments for the git command",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the tool with the given parameters
func (g *GitTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}

	// Build the git command
	args := []string{command}
	
	// Add additional arguments if provided
	if argsParam, ok := params["args"].([]interface{}); ok {
		for _, arg := range argsParam {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}

	// Special handling for common queries
	switch command {
	case "branch":
		// If no args, default to showing current branch
		if len(args) == 1 {
			args = append(args, "--show-current")
		}
	case "log":
		// Limit log output by default
		if len(args) == 1 {
			args = append(args, "--oneline", "-n", "10")
		}
	}

	// Execute the git command
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %v\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		result = "Command executed successfully with no output"
	}

	return result, nil
}