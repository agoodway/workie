package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemTool provides file system operations
type FileSystemTool struct{}

// NewFileSystemTool creates a new file system tool
func NewFileSystemTool() *FileSystemTool {
	return &FileSystemTool{}
}

// Name returns the name of the tool
func (f *FileSystemTool) Name() string {
	return "filesystem"
}

// Description returns what the tool does
func (f *FileSystemTool) Description() string {
	return "Read files and get information about the file system"
}

// Parameters returns the JSON schema for the tool's parameters
func (f *FileSystemTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "The file system operation to perform",
				"enum":        []string{"read", "list", "exists", "info"},
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file or directory path",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "For read operation, limit number of lines (default: 100)",
			},
		},
		"required": []string{"operation", "path"},
	}
}

// Execute runs the tool with the given parameters
func (f *FileSystemTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return "", fmt.Errorf("operation parameter is required")
	}

	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}

	// Get the current working directory as the base directory
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	// Clean and resolve the path
	path = filepath.Clean(path)

	// If path is relative, join it with base directory
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	// Resolve any symlinks
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If file doesn't exist yet, just use the cleaned path
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to resolve path: %v", err)
		}
		resolvedPath = path
	}

	// Ensure the resolved path is within the base directory
	relPath, err := filepath.Rel(baseDir, resolvedPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("access denied: path is outside the working directory")
	}

	// Use the safe resolved path
	path = resolvedPath

	switch operation {
	case "read":
		limit := 100
		if limitParam, ok := params["limit"].(float64); ok {
			limit = int(limitParam)
		}
		return f.readFile(path, limit)

	case "list":
		return f.listDirectory(path)

	case "exists":
		return f.checkExists(path)

	case "info":
		return f.getFileInfo(path)

	default:
		return "", fmt.Errorf("unknown operation: %s", operation)
	}
}

func (f *FileSystemTool) readFile(path string, limit int) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > limit {
		lines = lines[:limit]
		return strings.Join(lines, "\n") + fmt.Sprintf("\n... (truncated to %d lines)", limit), nil
	}

	return string(content), nil
}

func (f *FileSystemTool) listDirectory(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to list directory: %v", err)
	}

	var result []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		line := fmt.Sprintf("%s %10d %s",
			info.Mode().String(),
			info.Size(),
			entry.Name())

		if entry.IsDir() {
			line += "/"
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n"), nil
}

func (f *FileSystemTool) checkExists(path string) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "Path does not exist", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to check path: %v", err)
	}

	if info.IsDir() {
		return "Path exists and is a directory", nil
	}
	return "Path exists and is a file", nil
}

func (f *FileSystemTool) getFileInfo(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}

	result := fmt.Sprintf("Name: %s\n", info.Name())
	result += fmt.Sprintf("Size: %d bytes\n", info.Size())
	result += fmt.Sprintf("Mode: %s\n", info.Mode().String())
	result += fmt.Sprintf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("IsDir: %v\n", info.IsDir())

	return result, nil
}
