package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Hooks represents lifecycle commands to run at different events
type Hooks struct {
	PostCreate     []string `yaml:"post_create"`
	PreRemove      []string `yaml:"pre_remove"`
	PostCd         []string `yaml:"post_cd"`         // Commands to run after changing directory to worktree
	TimeoutMinutes int      `yaml:"timeout_minutes,omitempty"` // Hook execution timeout in minutes (default: 5)
}

// Config represents the YAML configuration structure
type Config struct {
	FilesToCopy []string `yaml:"files_to_copy"`
	Hooks       *Hooks   `yaml:"hooks,omitempty"`
	AutoCd      bool     `yaml:"auto_cd,omitempty"` // Automatically change directory to worktree after creation
	LoadedFrom  string   `yaml:"-"`                 // Path to the loaded config file (not serialized)
}

// LoadConfig attempts to load configuration from the specified file path,
// or defaults to .workie.yaml or workie.yaml if no custom path is provided.
// Returns an empty config if no file is found (not an error)
func LoadConfig(repoRoot string, customConfigPath string) (*Config, error) {
	// Validate repository root
	if repoRoot == "" {
		return nil, fmt.Errorf("repository root path cannot be empty")
	}

	// Verify repo root exists and is accessible
	if info, err := os.Stat(repoRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("repository root does not exist: %s", repoRoot)
		}
		return nil, fmt.Errorf("cannot access repository root %s: %w", repoRoot, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("repository root is not a directory: %s", repoRoot)
	}

	config := &Config{
		FilesToCopy: []string{},
	}

	var configPaths []string
	if customConfigPath != "" {
		// If custom config path is provided, use only that
		if filepath.IsAbs(customConfigPath) {
			configPaths = []string{customConfigPath}
		} else {
			// Relative path - resolve relative to repo root
			configPaths = []string{filepath.Join(repoRoot, customConfigPath)}
		}
	} else {
		// Use default config file locations
		configPaths = []string{
			filepath.Join(repoRoot, ".workie.yaml"),
			filepath.Join(repoRoot, "workie.yaml"),
		}
	}

	var configFile string
	var found bool

	for _, path := range configPaths {
		if info, err := os.Stat(path); err == nil {
			// Check if it's a file, not a directory
			if info.IsDir() {
				if customConfigPath != "" {
					return nil, fmt.Errorf("custom config path is a directory, not a file: %s", path)
				}
				// For default paths, just skip directories
				continue
			}
			configFile = path
			found = true
			break
		}
	}

	// If custom config path was specified but not found, return detailed error
	if customConfigPath != "" && !found {
		// Check if the custom path exists as a directory or has other issues
		customPath := customConfigPath
		if !filepath.IsAbs(customConfigPath) {
			customPath = filepath.Join(repoRoot, customConfigPath)
		}
		
		if info, err := os.Stat(customPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("custom config file not found: %s (resolved to: %s)", customConfigPath, customPath)
			}
			if os.IsPermission(err) {
				return nil, fmt.Errorf("permission denied accessing custom config file: %s", customConfigPath)
			}
			return nil, fmt.Errorf("cannot access custom config file %s: %w", customConfigPath, err)
		} else if info.IsDir() {
			return nil, fmt.Errorf("custom config path is a directory, not a file: %s", customConfigPath)
		}
		
		return nil, fmt.Errorf("custom config file not found: %s", customConfigPath)
	}

	if !found {
		// No config file found, return empty config (this is fine for default case)
		return config, nil
	}

	// Verify file is readable
	if info, err := os.Stat(configFile); err != nil {
		return nil, fmt.Errorf("config file became inaccessible: %s", configFile)
	} else {
		// Check file size - warn if suspiciously large (> 1MB)
		if info.Size() > 1024*1024 {
			return nil, fmt.Errorf("config file is suspiciously large (%d bytes): %s - please verify this is a YAML config file", info.Size(), configFile)
		}
		// Check file is not empty
		if info.Size() == 0 {
			return nil, fmt.Errorf("config file is empty: %s", configFile)
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading config file: %s - check file permissions", configFile)
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	// Validate that file contains some content and looks like YAML
	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, fmt.Errorf("config file contains no content: %s", configFile)
	}

	// Basic check if content looks like YAML (should contain colons or hyphens)
	if !strings.Contains(content, ":") && !strings.Contains(content, "-") {
		return nil, fmt.Errorf("config file does not appear to contain valid YAML: %s", configFile)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		// Provide more helpful YAML error messages
		errorStr := err.Error()
		if strings.Contains(errorStr, "line") && strings.Contains(errorStr, "column") {
			return nil, fmt.Errorf("failed to parse YAML config file %s: %s\n\nCommon YAML issues:\n  • Check indentation (use spaces, not tabs)\n  • Ensure colons are followed by spaces\n  • Quote strings containing special characters\n  • Verify bracket/brace matching", configFile, errorStr)
		}
		return nil, fmt.Errorf("failed to parse YAML config file %s: %w\n\nThis usually means the YAML syntax is invalid. Please check:\n  • File uses proper YAML format\n  • Indentation is consistent (use spaces)\n  • All keys and values are properly quoted if needed", configFile, err)
	}

	// Store the actual config file path that was loaded
	config.LoadedFrom = configFile

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed for %s: %w", configFile, err)
	}

	return config, nil
}

// HasFilesToCopy returns true if there are files configured to be copied
func (c *Config) HasFilesToCopy() bool {
	return len(c.FilesToCopy) > 0
}

// Validate checks if the configuration is valid and provides helpful error messages
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Check for duplicate entries
	seenFiles := make(map[string]bool)
	for i, file := range c.FilesToCopy {
		// Check for empty entries
		if strings.TrimSpace(file) == "" {
			return fmt.Errorf("empty file path at index %d in files_to_copy", i)
		}

		// Normalize path for duplicate checking
		normalizedFile := filepath.Clean(file)
		if seenFiles[normalizedFile] {
			return fmt.Errorf("duplicate file path in configuration: %s", file)
		}
		seenFiles[normalizedFile] = true

		// Check for obviously invalid paths
		if strings.Contains(file, "..") {
			return fmt.Errorf("file path contains '..' which could lead to security issues: %s", file)
		}

		// Check for absolute paths (usually not what you want)
		if filepath.IsAbs(file) {
			return fmt.Errorf("absolute file path found: %s - use relative paths in configuration", file)
		}

		// Check for suspicious characters
		if strings.ContainsAny(file, "*?<>|\":") {
			return fmt.Errorf("file path contains potentially problematic characters: %s", file)
		}
	}

	// Warn about excessive number of files (performance consideration)
	if len(c.FilesToCopy) > 100 {
		return fmt.Errorf("configuration contains %d files to copy - this might impact performance. Consider reducing the number of files or use directory paths instead", len(c.FilesToCopy))
	}

	// Validate hooks configuration if present
	if c.Hooks != nil {
		if err := c.validateHooks(); err != nil {
			return fmt.Errorf("hook validation failed: %w", err)
		}
	}

	return nil
}

// validateHooks validates the hooks configuration for security and performance
func (c *Config) validateHooks() error {
	if c.Hooks == nil {
		return nil
	}

	// Count total hooks for performance warning
	totalHooks := len(c.Hooks.PostCreate) + len(c.Hooks.PreRemove) + len(c.Hooks.PostCd)
	if totalHooks > 20 {
		return fmt.Errorf("configuration contains %d hooks total - this might impact performance. Consider reducing the number of hooks or combining commands", totalHooks)
	}

	// Validate post_create hooks
	if err := c.validateHookCommands("post_create", c.Hooks.PostCreate); err != nil {
		return err
	}

	// Validate pre_remove hooks
	if err := c.validateHookCommands("pre_remove", c.Hooks.PreRemove); err != nil {
		return err
	}

	// Validate post_cd hooks
	if err := c.validateHookCommands("post_cd", c.Hooks.PostCd); err != nil {
		return err
	}

	return nil
}

// validateHookCommands validates a list of hook commands for security and formatting
func (c *Config) validateHookCommands(hookType string, commands []string) error {
	seenCommands := make(map[string]bool)

	for i, cmd := range commands {
		// Check for commands that start with whitespace (before trimming)
		if len(cmd) > 0 && (cmd[0] == ' ' || cmd[0] == '\t') {
			return fmt.Errorf("command at index %d in %s hooks starts with whitespace - this might be a formatting error: '%s'", i, hookType, cmd)
		}

		// Check for empty commands
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			return fmt.Errorf("empty command at index %d in %s hooks", i, hookType)
		}

		// Check for duplicate commands
		if seenCommands[cmd] {
			return fmt.Errorf("duplicate command in %s hooks: %s", hookType, cmd)
		}
		seenCommands[cmd] = true

		// Check command length (extremely long commands might be suspicious)
		if len(cmd) > 1000 {
			return fmt.Errorf("command at index %d in %s hooks is suspiciously long (%d characters) - please verify this is correct", i, hookType, len(cmd))
		}

		// Security validation - check for obvious security risks
		if err := c.validateCommandSecurity(hookType, i, cmd); err != nil {
			return err
		}

		// Format validation - ensure commands are properly formatted
		if err := c.validateCommandFormat(hookType, i, cmd); err != nil {
			return err
		}
	}

	return nil
}

// validateCommandSecurity checks for obvious security risks in hook commands
func (c *Config) validateCommandSecurity(hookType string, index int, cmd string) error {
	// Convert to lowercase for case-insensitive checks
	cmdLower := strings.ToLower(cmd)

	// Check for dangerous commands or patterns
	dangerousPatterns := []struct {
		pattern string
		reason  string
	}{
		{"rm -rf /", "attempts to delete entire filesystem"},
		{"rm -rf /*", "attempts to delete entire filesystem"},
		{":(){ :|:& };:", "fork bomb pattern detected"},
		{"curl | sh", "pipes remote content directly to shell"},
		{"wget | sh", "pipes remote content directly to shell"},
		{"curl | bash", "pipes remote content directly to shell"},
		{"wget | bash", "pipes remote content directly to shell"},
		{"dd if=/dev/random", "potentially destructive disk operation"},
		{"mkfs", "filesystem formatting command detected"},
		{"fdisk", "disk partitioning command detected"},
		{"format c:", "Windows disk formatting command detected"},
		{"del /f /s /q c:\\", "Windows destructive deletion command"},
		{"deltree", "Windows destructive deletion command"},
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(cmdLower, dangerous.pattern) {
			return fmt.Errorf("potentially dangerous command at index %d in %s hooks: %s - %s", index, hookType, cmd, dangerous.reason)
		}
	}

	// Check for suspicious network operations
	suspiciousNetworkPatterns := []string{
		"nc -l", "ncat -l", "netcat -l", // listening network connections
		"python -m http.server", "python3 -m http.server", // HTTP servers
		"php -S", // PHP development server
	}

	for _, pattern := range suspiciousNetworkPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("potentially risky network command at index %d in %s hooks: %s - starts network service", index, hookType, cmd)
		}
	}

	// Check for privilege escalation attempts
	privilegePatterns := []string{
		"sudo su", "sudo -i", "su -", "sudo bash", "sudo sh",
	}

	for _, pattern := range privilegePatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("privilege escalation detected at index %d in %s hooks: %s - avoid using interactive shells in hooks", index, hookType, cmd)
		}
	}

	// Warn about commands that modify system files
	systemModificationPatterns := []string{
		"/etc/passwd", "/etc/shadow", "/etc/hosts", "/etc/sudoers",
		"c:\\windows\\system32", "/System/Library",
	}

	for _, pattern := range systemModificationPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("system file modification detected at index %d in %s hooks: %s - modifying system files in hooks is not recommended", index, hookType, cmd)
		}
	}

	return nil
}

// validateCommandFormat ensures hook commands are properly formatted
func (c *Config) validateCommandFormat(hookType string, index int, cmd string) error {
	// Check for unbalanced quotes
	singleQuotes := strings.Count(cmd, "'")
	doubleQuotes := strings.Count(cmd, `"`)

	if singleQuotes%2 != 0 {
		return fmt.Errorf("unbalanced single quotes in command at index %d in %s hooks: %s", index, hookType, cmd)
	}

	if doubleQuotes%2 != 0 {
		return fmt.Errorf("unbalanced double quotes in command at index %d in %s hooks: %s", index, hookType, cmd)
	}

	// Check for unbalanced parentheses
	parenCount := 0
	for _, char := range cmd {
		if char == '(' {
			parenCount++
		} else if char == ')' {
			parenCount--
		}
	}
	if parenCount != 0 {
		return fmt.Errorf("unbalanced parentheses in command at index %d in %s hooks: %s", index, hookType, cmd)
	}

	// Check for unbalanced braces
	braceCount := 0
	for _, char := range cmd {
		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
		}
	}
	if braceCount != 0 {
		return fmt.Errorf("unbalanced braces in command at index %d in %s hooks: %s", index, hookType, cmd)
	}

	// Check for unbalanced brackets
	bracketCount := 0
	for _, char := range cmd {
		if char == '[' {
			bracketCount++
		} else if char == ']' {
			bracketCount--
		}
	}
	if bracketCount != 0 {
		return fmt.Errorf("unbalanced brackets in command at index %d in %s hooks: %s", index, hookType, cmd)
	}

	// Check for null bytes or other control characters (except common ones)
	for i, char := range cmd {
		if char < 32 && char != '\t' && char != '\n' && char != '\r' {
			return fmt.Errorf("invalid control character (byte %d) at position %d in command at index %d in %s hooks", char, i, index, hookType)
		}
	}

	// Warn about commands that end with backslash (might be incomplete)
	if strings.HasSuffix(cmd, "\\") {
		return fmt.Errorf("command at index %d in %s hooks ends with backslash - this might be incomplete: %s", index, hookType, cmd)
	}

	return nil
}
