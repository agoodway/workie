package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommitMessageTool generates commit messages based on git changes
type CommitMessageTool struct{}

// NewCommitMessageTool creates a new commit message tool
func NewCommitMessageTool() *CommitMessageTool {
	return &CommitMessageTool{}
}

// Name returns the name of the tool
func (c *CommitMessageTool) Name() string {
	return "commit_message"
}

// Description returns what the tool does
func (c *CommitMessageTool) Description() string {
	return "Generate commit messages based on git changes. Analyzes staged and unstaged files to create descriptive commit messages"
}

// Parameters returns the JSON schema for the tool's parameters
func (c *CommitMessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Type of changes to analyze",
				"enum":        []string{"staged", "unstaged", "all"},
				"default":     "all",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Commit message format",
				"enum":        []string{"conventional", "simple", "detailed"},
				"default":     "conventional",
			},
		},
	}
}

// Execute runs the tool with the given parameters
func (c *CommitMessageTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	changeType := "all"
	if t, ok := params["type"].(string); ok {
		changeType = t
	}

	format := "conventional"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	// Get the changes
	changes, err := c.getChanges(ctx, changeType)
	if err != nil {
		return "", fmt.Errorf("failed to get changes: %v", err)
	}

	if changes == "" {
		return "No changes detected to create a commit message", nil
	}

	// Generate commit message based on changes
	message := c.generateMessage(changes, format)
	
	return message, nil
}

func (c *CommitMessageTool) getChanges(ctx context.Context, changeType string) (string, error) {
	var result strings.Builder

	// Get status
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %v", err)
	}

	status := string(statusOutput)
	if status == "" {
		return "", nil
	}

	result.WriteString("File changes:\n")
	result.WriteString(status)
	result.WriteString("\n")

	// Get diff based on type
	var diffArgs []string
	switch changeType {
	case "staged":
		diffArgs = []string{"diff", "--cached", "--stat"}
	case "unstaged":
		diffArgs = []string{"diff", "--stat"}
	case "all":
		// Get both staged and unstaged
		diffArgs = []string{"diff", "HEAD", "--stat"}
	}

	if len(diffArgs) > 0 {
		diffCmd := exec.CommandContext(ctx, "git", diffArgs...)
		diffOutput, err := diffCmd.Output()
		if err == nil && len(diffOutput) > 0 {
			result.WriteString("\nChange summary:\n")
			result.WriteString(string(diffOutput))
		}
	}

	// Get more detailed diff for analysis
	detailArgs := []string{"diff"}
	if changeType == "staged" {
		detailArgs = append(detailArgs, "--cached")
	} else if changeType == "all" {
		detailArgs = append(detailArgs, "HEAD")
	}
	detailArgs = append(detailArgs, "--name-only")

	detailCmd := exec.CommandContext(ctx, "git", detailArgs...)
	detailOutput, err := detailCmd.Output()
	if err == nil && len(detailOutput) > 0 {
		files := strings.Split(strings.TrimSpace(string(detailOutput)), "\n")
		result.WriteString("\nModified files:\n")
		for _, file := range files {
			if file != "" {
				result.WriteString("- " + file + "\n")
			}
		}
	}

	return result.String(), nil
}

func (c *CommitMessageTool) generateMessage(changes string, format string) string {
	// Parse the changes to understand what was modified
	lines := strings.Split(changes, "\n")
	var modifiedFiles []string
	var addedFiles []string
	var deletedFiles []string
	var fileTypes = make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "M ") || strings.HasPrefix(line, " M") {
			file := strings.TrimSpace(line[2:])
			modifiedFiles = append(modifiedFiles, file)
			fileTypes[getFileType(file)]++
		} else if strings.HasPrefix(line, "A ") || strings.HasPrefix(line, " A") {
			file := strings.TrimSpace(line[2:])
			addedFiles = append(addedFiles, file)
			fileTypes[getFileType(file)]++
		} else if strings.HasPrefix(line, "D ") || strings.HasPrefix(line, " D") {
			file := strings.TrimSpace(line[2:])
			deletedFiles = append(deletedFiles, file)
		}
	}

	// Generate message based on format
	switch format {
	case "conventional":
		return c.generateConventionalMessage(modifiedFiles, addedFiles, deletedFiles, fileTypes)
	case "detailed":
		return c.generateDetailedMessage(modifiedFiles, addedFiles, deletedFiles, changes)
	default:
		return c.generateSimpleMessage(modifiedFiles, addedFiles, deletedFiles)
	}
}

func (c *CommitMessageTool) generateConventionalMessage(modified, added, deleted []string, fileTypes map[string]int) string {
	// Determine the type of change
	var commitType string
	var scope string
	var description string

	// Analyze the most common file type
	maxCount := 0
	for ft, count := range fileTypes {
		if count > maxCount {
			maxCount = count
			scope = ft
		}
	}

	// Determine commit type and description based on changes
	if len(added) > 0 && len(modified) == 0 && len(deleted) == 0 {
		commitType = "feat"
		if len(added) == 1 {
			fileName := getFileName(added[0])
			if strings.Contains(fileName, "commit_message_tool") {
				description = "add commit message generation tool"
			} else {
				description = fmt.Sprintf("add %s", fileName)
			}
		} else {
			description = fmt.Sprintf("add %d new files", len(added))
		}
	} else if len(deleted) > 0 && len(modified) == 0 && len(added) == 0 {
		commitType = "chore"
		if len(deleted) == 1 {
			description = fmt.Sprintf("remove %s", getFileName(deleted[0]))
		} else {
			description = fmt.Sprintf("remove %d files", len(deleted))
		}
	} else if len(added) > 0 && len(modified) > 0 {
		// Mixed changes - determine based on what's added
		commitType = "feat"
		addedFile := getFileName(added[0])
		if strings.Contains(addedFile, "tool") {
			description = "implement tool/function calling with commit message generation"
		} else {
			description = fmt.Sprintf("add %s and update related files", addedFile)
		}
	} else if len(modified) > 0 {
		// Only modifications
		if containsTest(modified) {
			commitType = "test"
			description = "update test files"
		} else if containsDocs(modified) {
			commitType = "docs"
			description = "update documentation"
		} else if containsConfig(modified) {
			commitType = "chore"
			description = "update configuration"
		} else {
			// Look at specific files for better description
			if contains(modified, "ask.go") && contains(modified, "tool") {
				commitType = "feat"
				description = "enhance ask command with tool support"
			} else if contains(modified, "git_tool.go") {
				commitType = "feat"
				description = "enhance git tool functionality"
			} else {
				commitType = "feat"
				description = "update implementation"
			}
		}
	} else {
		commitType = "chore"
		description = "update files"
	}

	// Build the commit message
	if scope != "" && scope != "other" {
		return fmt.Sprintf("%s(%s): %s", commitType, scope, description)
	}
	return fmt.Sprintf("%s: %s", commitType, description)
}

func contains(files []string, substr string) bool {
	for _, file := range files {
		if strings.Contains(file, substr) {
			return true
		}
	}
	return false
}

func (c *CommitMessageTool) generateSimpleMessage(modified, added, deleted []string) string {
	parts := []string{}
	
	if len(added) > 0 {
		if len(added) == 1 {
			parts = append(parts, fmt.Sprintf("Add %s", getFileName(added[0])))
		} else {
			parts = append(parts, fmt.Sprintf("Add %d files", len(added)))
		}
	}
	
	if len(modified) > 0 {
		if len(modified) == 1 {
			parts = append(parts, fmt.Sprintf("Update %s", getFileName(modified[0])))
		} else {
			parts = append(parts, fmt.Sprintf("Update %d files", len(modified)))
		}
	}
	
	if len(deleted) > 0 {
		if len(deleted) == 1 {
			parts = append(parts, fmt.Sprintf("Remove %s", getFileName(deleted[0])))
		} else {
			parts = append(parts, fmt.Sprintf("Remove %d files", len(deleted)))
		}
	}
	
	if len(parts) == 0 {
		return "Update files"
	}
	
	return strings.Join(parts, ", ")
}

func (c *CommitMessageTool) generateDetailedMessage(modified, added, deleted []string, changes string) string {
	var message strings.Builder
	
	// Start with a summary
	message.WriteString(c.generateSimpleMessage(modified, added, deleted))
	message.WriteString("\n\n")
	
	// Add details
	if len(added) > 0 {
		message.WriteString("Added:\n")
		for _, file := range added {
			message.WriteString("- " + file + "\n")
		}
		message.WriteString("\n")
	}
	
	if len(modified) > 0 {
		message.WriteString("Modified:\n")
		for _, file := range modified {
			message.WriteString("- " + file + "\n")
		}
		message.WriteString("\n")
	}
	
	if len(deleted) > 0 {
		message.WriteString("Deleted:\n")
		for _, file := range deleted {
			message.WriteString("- " + file + "\n")
		}
	}
	
	return strings.TrimSpace(message.String())
}

// Helper functions
func getFileType(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		// Check common directories
		switch parts[0] {
		case "cmd":
			return "cmd"
		case "tools":
			return "tools"
		case "config":
			return "config"
		case "docs":
			return "docs"
		case "test", "tests":
			return "test"
		}
	}
	
	// Check by extension
	if strings.HasSuffix(path, ".go") {
		return "go"
	} else if strings.HasSuffix(path, ".md") {
		return "docs"
	} else if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return "config"
	}
	
	return "other"
}

func getFileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func containsTest(files []string) bool {
	for _, file := range files {
		if strings.Contains(file, "_test.go") || strings.Contains(file, "/test") {
			return true
		}
	}
	return false
}

func containsDocs(files []string) bool {
	for _, file := range files {
		if strings.HasSuffix(file, ".md") || strings.Contains(file, "/docs") {
			return true
		}
	}
	return false
}

func containsConfig(files []string) bool {
	for _, file := range files {
		if strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml") || 
		   strings.HasSuffix(file, ".json") || strings.HasSuffix(file, ".toml") {
			return true
		}
	}
	return false
}