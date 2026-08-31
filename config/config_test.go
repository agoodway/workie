package config

import (
	"os"
	"path/filepath"
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

func TestSpriteConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workie-sprite-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("valid sprite config", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example
sprite:
  name: "my-dev-vm"
  org: "my-org"
  pre_setup:
    - "mkdir -p ~/go/bin"
    - "echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc"
  post_setup:
    - "which td && td --version"
    - "echo 'Setup complete'"`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.Sprite == nil {
			t.Fatal("Expected sprite config to be set")
		}
		if config.Sprite.Name != "my-dev-vm" {
			t.Errorf("Expected sprite name 'my-dev-vm', got '%s'", config.Sprite.Name)
		}
		if config.Sprite.Org != "my-org" {
			t.Errorf("Expected sprite org 'my-org', got '%s'", config.Sprite.Org)
		}
		if len(config.Sprite.PreSetup) != 2 {
			t.Fatalf("Expected 2 pre_setup commands, got %d", len(config.Sprite.PreSetup))
		}
		if config.Sprite.PreSetup[0] != "mkdir -p ~/go/bin" {
			t.Errorf("Expected first pre_setup command 'mkdir -p ~/go/bin', got '%s'", config.Sprite.PreSetup[0])
		}
		if config.Sprite.PreSetup[1] != "echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc" {
			t.Errorf("Expected second pre_setup command, got '%s'", config.Sprite.PreSetup[1])
		}
		if len(config.Sprite.PostSetup) != 2 {
			t.Fatalf("Expected 2 post_setup commands, got %d", len(config.Sprite.PostSetup))
		}
		if config.Sprite.PostSetup[0] != "which td && td --version" {
			t.Errorf("Expected first post_setup command, got '%s'", config.Sprite.PostSetup[0])
		}
		if config.Sprite.PostSetup[1] != "echo 'Setup complete'" {
			t.Errorf("Expected second post_setup command, got '%s'", config.Sprite.PostSetup[1])
		}
	})

	t.Run("missing sprite section", func(t *testing.T) {
		configContent := `files_to_copy:
  - .env.example`

		configPath := filepath.Join(tempDir, ".workie.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tempDir, "")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.Sprite != nil {
			t.Error("Expected sprite config to be nil when not configured")
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
		if config.Hooks == nil {
			t.Fatal("Expected hooks to be set")
		}
		if len(config.Hooks.PostCreate) != 2 {
			t.Errorf("Expected 2 post_create hooks, got %d", len(config.Hooks.PostCreate))
		}
		if len(config.Hooks.PreRemove) != 1 {
			t.Errorf("Expected 1 pre_remove hook, got %d", len(config.Hooks.PreRemove))
		}
	})
}
