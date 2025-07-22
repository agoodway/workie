package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "worktree-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("no config file", func(t *testing.T) {
		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Errorf("Expected no error when config file doesn't exist, got: %v", err)
		}
		if config == nil {
			t.Fatal("Expected config to be returned, got nil")
		}
		if len(config.FilesToCopy) != 0 {
			t.Errorf("Expected empty FilesToCopy, got: %v", config.FilesToCopy)
		}
	})

	t.Run("valid .workie.yaml", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example
  - config/
  - scripts/setup.sh`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config == nil {
			t.Fatal("Expected config to be returned, got nil")
		}

		expectedFiles := []string{".env.example", "config/", "scripts/setup.sh"}
		if len(config.FilesToCopy) != len(expectedFiles) {
			t.Errorf("Expected %d files, got %d", len(expectedFiles), len(config.FilesToCopy))
		}

		for i, expected := range expectedFiles {
			if i >= len(config.FilesToCopy) || config.FilesToCopy[i] != expected {
				t.Errorf("Expected file %s at index %d, got %v", expected, i, config.FilesToCopy)
			}
		}
	})

	t.Run("valid workie.yaml (fallback)", func(t *testing.T) {
		// Clean up any existing .workie.yaml to test fallback
		os.Remove(filepath.Join(tempDir, ".workie.yaml"))

		configContent := `files_to_copy:
  - README.md`

		configPath := filepath.Join(tempDir, "workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config == nil {
			t.Fatal("Expected config to be returned, got nil")
		}

		if len(config.FilesToCopy) != 1 || config.FilesToCopy[0] != "README.md" {
			t.Errorf("Expected [README.md], got %v", config.FilesToCopy)
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example
invalid yaml: [[[`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for invalid YAML, got none")
		}
	})
}

func TestHasFilesToCopy(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := &Config{FilesToCopy: []string{}}
		if config.HasFilesToCopy() {
			t.Error("Expected HasFilesToCopy to return false for empty config")
		}
	})

	t.Run("config with files", func(t *testing.T) {
		config := &Config{FilesToCopy: []string{".env.example"}}
		if !config.HasFilesToCopy() {
			t.Error("Expected HasFilesToCopy to return true for config with files")
		}
	})
}

func TestHookValidation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "workie-hook-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("valid hooks", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example
hooks:
  post_create:
    - "echo 'Setup complete'"
    - "npm install"
  pre_remove:
    - "npm run cleanup"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Errorf("Expected no error for valid hooks, got: %v", err)
		}
		if config == nil {
			t.Fatal("Expected config to be returned, got nil")
		}
	})

	t.Run("empty hook command", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example
hooks:
  post_create:
    - "echo 'valid command'"
    - ""
    - "npm install"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for empty hook command, got none")
		}
		if !strings.Contains(err.Error(), "empty command") {
			t.Errorf("Expected error to mention empty command, got: %v", err)
		}
	})

	t.Run("dangerous command - rm -rf", func(t *testing.T) {
		configContent := `hooks:
  post_create:
    - "rm -rf /"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for dangerous rm -rf command, got none")
		}
		if !strings.Contains(err.Error(), "potentially dangerous command") {
			t.Errorf("Expected error to mention dangerous command, got: %v", err)
		}
	})

	t.Run("suspicious network command", func(t *testing.T) {
		configContent := `hooks:
  post_create:
    - "python -m http.server 8080"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for network command, got none")
		}
		if !strings.Contains(err.Error(), "potentially risky network command") {
			t.Errorf("Expected error to mention risky network command, got: %v", err)
		}
	})

	t.Run("unbalanced quotes", func(t *testing.T) {
		configContent := `hooks:
  post_create:
    - "echo 'hello world"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for unbalanced quotes, got none")
		}
		if !strings.Contains(err.Error(), "unbalanced") {
			t.Errorf("Expected error to mention unbalanced quotes, got: %v", err)
		}
	})

	t.Run("excessive number of hooks", func(t *testing.T) {
		configContent := `hooks:
  post_create:
`
		// Add many hook commands to exceed the limit
		for i := 0; i < 25; i++ {
			configContent += "    - \"echo command " + string(rune('0'+i%10)) + "\"\n"
		}

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for too many hooks, got none")
		}
		if !strings.Contains(err.Error(), "might impact performance") {
			t.Errorf("Expected error to mention performance impact, got: %v", err)
		}
	})

	t.Run("duplicate commands", func(t *testing.T) {
		configContent := `hooks:
  post_create:
    - "npm install"
    - "npm test"
    - "npm install"
`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for duplicate commands, got none")
		}
		if !strings.Contains(err.Error(), "duplicate command") {
			t.Errorf("Expected error to mention duplicate command, got: %v", err)
		}
	})

	t.Run("privilege escalation", func(t *testing.T) {
		configContent := `hooks:
  pre_remove:
    - "sudo bash"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for privilege escalation, got none")
		}
		if !strings.Contains(err.Error(), "privilege escalation") {
			t.Errorf("Expected error to mention privilege escalation, got: %v", err)
		}
	})

	t.Run("command with whitespace prefix", func(t *testing.T) {
		configContent := `hooks:
  post_create:
    - " echo hello"
`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tempDir, "")
		if err == nil {
			t.Error("Expected error for command starting with whitespace, got none")
		}
		if !strings.Contains(err.Error(), "starts with whitespace") {
			t.Errorf("Expected error to mention whitespace prefix, got: %v", err)
		}
	})
}
