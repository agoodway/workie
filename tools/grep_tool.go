package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool provides code search functionality for the LLM
type GrepTool struct{}

// NewGrepTool creates a new grep tool
func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

// Name returns the name of the tool
func (g *GrepTool) Name() string {
	return "grep"
}

// Description returns what the tool does
func (g *GrepTool) Description() string {
	return "Search for patterns in files within the codebase"
}

// Parameters returns the JSON schema for the tool's parameters
func (g *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The search pattern (supports regular expressions)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory or file to search in (default: current directory)",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string",
				"description": "File name pattern to filter files (e.g., '*.go', '*.js')",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the search should be case sensitive (default: true)",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 100)",
			},
			"include_line_numbers": map[string]interface{}{
				"type":        "boolean",
				"description": "Include line numbers in results (default: true)",
			},
			"context_lines": map[string]interface{}{
				"type":        "integer",
				"description": "Number of context lines to show before and after matches (default: 0)",
			},
		},
		"required": []string{"pattern"},
	}
}

// Execute runs the grep tool with the given parameters
func (g *GrepTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// Extract parameters
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter is required")
	}

	searchPath := "."
	if path, ok := params["path"].(string); ok {
		searchPath = path
	}

	filePattern := "*"
	if fp, ok := params["file_pattern"].(string); ok {
		filePattern = fp
	}

	caseSensitive := true
	if cs, ok := params["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	maxResults := 100
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	includeLineNumbers := true
	if iln, ok := params["include_line_numbers"].(bool); ok {
		includeLineNumbers = iln
	}

	contextLines := 0
	if cl, ok := params["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	// Compile the regex pattern
	var re *regexp.Regexp
	var err error
	if caseSensitive {
		re, err = regexp.Compile(pattern)
	} else {
		re, err = regexp.Compile("(?i)" + pattern)
	}
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %v", err)
	}

	// Get the current working directory as the base directory
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	// Clean and resolve the search path
	searchPath = filepath.Clean(searchPath)
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(baseDir, searchPath)
	}

	// Ensure the search path is within the base directory
	relPath, err := filepath.Rel(baseDir, searchPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("access denied: path is outside the working directory")
	}

	// Perform the search
	results := []string{}
	resultCount := 0

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Skip directories and binary files
		if info.IsDir() || isBinaryFile(path) {
			return nil
		}

		// Skip hidden files and directories
		if strings.Contains(path, "/.") {
			return nil
		}

		// Check file pattern
		matched, err := filepath.Match(filePattern, filepath.Base(path))
		if err != nil || !matched {
			return nil
		}

		// Search in the file
		fileResults, count, err := searchInFile(path, re, includeLineNumbers, contextLines, maxResults-resultCount)
		if err != nil {
			return nil // Skip files with errors
		}

		if len(fileResults) > 0 {
			relPath, _ := filepath.Rel(baseDir, path)
			results = append(results, fmt.Sprintf("\n=== %s ===", relPath))
			results = append(results, fileResults...)
			resultCount += count
		}

		if resultCount >= maxResults {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("error during search: %v", err)
	}

	if len(results) == 0 {
		return "No matches found", nil
	}

	result := strings.Join(results, "\n")
	if resultCount >= maxResults {
		result += fmt.Sprintf("\n\n... (search limited to %d results)", maxResults)
	}

	return result, nil
}

// searchInFile searches for pattern in a single file
func searchInFile(path string, re *regexp.Regexp, includeLineNumbers bool, contextLines int, maxResults int) ([]string, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var results []string
	var buffer []string
	lineNum := 0
	resultCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Keep a buffer for context lines
		if contextLines > 0 {
			buffer = append(buffer, line)
			if len(buffer) > contextLines*2+1 {
				buffer = buffer[1:]
			}
		}

		if re.MatchString(line) {
			// Add context lines before match
			if contextLines > 0 && len(buffer) > 1 {
				start := 0
				if len(buffer) > contextLines+1 {
					start = len(buffer) - contextLines - 1
				}
				for i := start; i < len(buffer)-1; i++ {
					contextLine := buffer[i]
					contextLineNum := lineNum - (len(buffer) - i - 1)
					if includeLineNumbers {
						results = append(results, fmt.Sprintf("  %4d: %s", contextLineNum, contextLine))
					} else {
						results = append(results, fmt.Sprintf("  %s", contextLine))
					}
				}
			}

			// Add the matching line
			if includeLineNumbers {
				results = append(results, fmt.Sprintf("* %4d: %s", lineNum, line))
			} else {
				results = append(results, fmt.Sprintf("* %s", line))
			}

			resultCount++
			if resultCount >= maxResults {
				break
			}

			// Add context lines after match
			if contextLines > 0 {
				for i := 0; i < contextLines && scanner.Scan(); i++ {
					lineNum++
					contextLine := scanner.Text()
					buffer = append(buffer, contextLine)
					if includeLineNumbers {
						results = append(results, fmt.Sprintf("  %4d: %s", lineNum, contextLine))
					} else {
						results = append(results, fmt.Sprintf("  %s", contextLine))
					}
				}
			}
		}
	}

	return results, resultCount, scanner.Err()
}

// isBinaryFile checks if a file is likely to be binary
func isBinaryFile(path string) bool {
	// Common binary file extensions
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".o",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".zip", ".tar", ".gz", ".bz2", ".7z",
		".bin", ".dat", ".db", ".sqlite",
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}

	// Check if file is in common binary directories
	if strings.Contains(path, "/node_modules/") ||
		strings.Contains(path, "/.git/") ||
		strings.Contains(path, "/vendor/") ||
		strings.Contains(path, "/dist/") ||
		strings.Contains(path, "/build/") {
		return true
	}

	return false
}
