package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ShellTool provides safe shell command execution
type ShellTool struct {
	allowedCommands []string
}

// NewShellTool creates a new shell tool with a whitelist of allowed commands
func NewShellTool() *ShellTool {
	return &ShellTool{
		allowedCommands: []string{
			"pwd", "ls", "cat", "head", "tail", "grep", "find",
			"echo", "date", "whoami", "hostname", "uname",
		},
	}
}

// Name returns the name of the tool
func (s *ShellTool) Name() string {
	return "shell"
}

// Description returns what the tool does
func (s *ShellTool) Description() string {
	return "Execute safe shell commands to get system information"
}

// Parameters returns the JSON schema for the tool's parameters
func (s *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
				"enum":        s.allowedCommands,
			},
			"args": map[string]interface{}{
				"type":        "array",
				"description": "Arguments for the command",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the tool with the given parameters
func (s *ShellTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}

	// Check if command is allowed
	allowed := false
	for _, cmd := range s.allowedCommands {
		if cmd == command {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("command '%s' is not allowed", command)
	}

	// Build command arguments
	args := []string{}
	if argsParam, ok := params["args"].([]interface{}); ok {
		for _, arg := range argsParam {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}

	// Execute the command
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %v\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		result = "Command executed successfully with no output"
	}

	return result, nil
}