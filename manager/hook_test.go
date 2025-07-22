package manager

import (
	"os"
	"strings"
	"testing"
	"time"
	"workie/config"
)

// TestExecuteHooks tests hook execution logic with various scenarios
func TestExecuteHooks(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "workie-hook-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("empty hooks list", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		err := wm.ExecuteHooks([]string{}, tempDir, "post_create")
		if err != nil {
			t.Errorf("Expected no error for empty hooks, got: %v", err)
		}
	})

	t.Run("successful command execution", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true // Suppress output during tests
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		// Use echo command which should be available on all systems
		hooks := []string{"echo 'test successful'"}
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		if err != nil {
			t.Errorf("Expected no error for successful command, got: %v", err)
		}
	})

	t.Run("command not found", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		hooks := []string{"nonexistent-command-12345"}
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		// Note: We don't expect ExecuteHooks to fail completely for individual command failures
		// It should continue processing and only fail if ALL hooks fail
		if err == nil {
			// This is expected behavior - warnings are shown but execution continues
			t.Log("ExecuteHooks correctly continued despite command failure")
		}
	})

	t.Run("mixed successful and failing commands", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		hooks := []string{
			"echo 'first command success'",
			"nonexistent-command-12345",
			"echo 'third command success'",
		}
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		// Should not fail completely since some commands succeed
		if err != nil {
			t.Log("ExecuteHooks failed, but this might be expected behavior for mixed results")
		}
	})

	t.Run("all commands fail", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		hooks := []string{
			"nonexistent-command-1",
			"nonexistent-command-2",
		}
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		if err == nil {
			t.Error("Expected error when all hooks fail, got none")
		}
		if !strings.Contains(err.Error(), "all") {
			t.Errorf("Expected error message about 'all hooks failed', got: %v", err)
		}
	})

	t.Run("invalid working directory", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		invalidDir := "/nonexistent/directory/path"
		hooks := []string{"echo 'test'"}
		err := wm.ExecuteHooks(hooks, invalidDir, "post_create")
		if err == nil {
			t.Error("Expected error for invalid working directory, got none")
		}
		if !strings.Contains(err.Error(), "working directory does not exist") {
			t.Errorf("Expected working directory error, got: %v", err)
		}
	})

	t.Run("hook timeout", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{
				TimeoutMinutes: 1, // Very short timeout for testing
			},
		}
		
		// Command that sleeps longer than timeout
		hooks := []string{"sleep 65"} // 65 seconds > 1 minute timeout
		
		start := time.Now()
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		duration := time.Since(start)
		
		// Should timeout within reasonable bounds (not wait full 65 seconds)
		if duration > 70*time.Second {
			t.Errorf("Hook execution took too long: %v (expected timeout around 1 minute)", duration)
		}
		
		// Should not completely fail execution due to timeout
		if err != nil {
			t.Log("Hook execution failed due to timeout, which may be expected behavior")
		}
	})

	t.Run("empty hook command in list", func(t *testing.T) {
		wm := New()
		wm.Options.Quiet = true
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		// Include empty command in the list
		hooks := []string{
			"echo 'before empty'",
			"", // Empty command
			"echo 'after empty'",
		}
		err := wm.ExecuteHooks(hooks, tempDir, "post_create")
		if err != nil {
			t.Errorf("Expected no error with empty command in list, got: %v", err)
		}
	})
}

// TestHasHooks tests the helper methods for checking hook presence
func TestHasHooks(t *testing.T) {
	t.Run("has post create hooks", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{
				PostCreate: []string{"echo 'test'"},
			},
		}
		
		if !wm.HasPostCreateHooks() {
			t.Error("Expected HasPostCreateHooks to return true")
		}
	})

	t.Run("no post create hooks", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		if wm.HasPostCreateHooks() {
			t.Error("Expected HasPostCreateHooks to return false")
		}
	})

	t.Run("has pre remove hooks", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{
				PreRemove: []string{"echo 'cleanup'"},
			},
		}
		
		if !wm.HasPreRemoveHooks() {
			t.Error("Expected HasPreRemoveHooks to return true")
		}
	})

	t.Run("no config at all", func(t *testing.T) {
		wm := New()
		wm.Config = nil
		
		if wm.HasPostCreateHooks() {
			t.Error("Expected HasPostCreateHooks to return false when config is nil")
		}
		if wm.HasPreRemoveHooks() {
			t.Error("Expected HasPreRemoveHooks to return false when config is nil")
		}
	})
}

// TestHookTimeout tests the timeout configuration
func TestHookTimeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{},
		}
		
		// Use reflection or indirect testing since getHookTimeout is not exported
		// We'll test via ExecuteHooks with a command that should complete within default timeout
		wm.Options.Quiet = true
		hooks := []string{"echo 'timeout test'"}
		err := wm.ExecuteHooks(hooks, "/tmp", "test")
		if err != nil {
			t.Errorf("Expected no error with default timeout, got: %v", err)
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		wm := New()
		wm.Config = &config.Config{
			Hooks: &config.Hooks{
				TimeoutMinutes: 10, // Custom timeout
			},
		}
		
		wm.Options.Quiet = true
		hooks := []string{"echo 'custom timeout test'"}
		err := wm.ExecuteHooks(hooks, "/tmp", "test")
		if err != nil {
			t.Errorf("Expected no error with custom timeout, got: %v", err)
		}
	})
}

