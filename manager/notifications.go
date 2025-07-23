package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gen2brain/beeep"
)

// NotificationInput represents the input for notification hooks
type NotificationInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Message        string `json:"message"`
}

// SendSystemNotification sends a system notification after claude_notification hooks
func (wm *WorktreeManager) SendSystemNotification(input *NotificationInput) error {
	// Check if system notifications are enabled
	if wm.Config == nil || wm.Config.Hooks == nil || wm.Config.Hooks.SystemNotifications == nil || !wm.Config.Hooks.SystemNotifications.Enabled {
		if wm.Options.Verbose {
			wm.printf("System notifications not enabled in config\n")
		}
		return nil // Silently skip if not enabled
	}

	wm.printf("System notifications are enabled\n")

	// Prepare notification title
	title := wm.Config.Hooks.SystemNotifications.Title
	if title == "" {
		title = "Workie - Claude Code"
	}

	// Prepare notification message
	message := input.Message
	if message == "" {
		message = "Claude Code notification"
	}

	// Get icon path if configured
	iconPath := wm.Config.Hooks.SystemNotifications.Icon
	if iconPath != "" && !filepath.IsAbs(iconPath) {
		// Make relative paths absolute based on repo path
		iconPath = filepath.Join(wm.RepoPath, iconPath)
	}

	// Validate icon exists if specified
	if iconPath != "" {
		if _, err := os.Stat(iconPath); os.IsNotExist(err) {
			// Icon doesn't exist, use default
			iconPath = ""
		}
	}

	// Use default icon based on platform if none specified
	if iconPath == "" {
		iconPath = getDefaultIcon()
	}

	// Debug output
	if wm.Options.Verbose {
		wm.printf("Attempting to send notification - Title: %s, Message: %s\n", title, message)
	}

	// On macOS, prefer osascript for better reliability
	if runtime.GOOS == "darwin" {
		// Escape quotes in the message and title
		escapedMessage := strings.ReplaceAll(message, `"`, `\"`)
		escapedTitle := strings.ReplaceAll(title, `"`, `\"`)

		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, escapedMessage, escapedTitle)
		cmd := exec.Command("osascript", "-e", script)

		output, err := cmd.CombinedOutput()
		if err != nil {
			wm.printf("Warning: osascript failed: %v (output: %s)\n", err, string(output))
			// Fall back to beeep
			if err := beeep.Notify(title, message, iconPath); err != nil {
				wm.printf("Warning: beeep also failed: %v\n", err)
				return nil
			}
			wm.printf("✓ System notification sent via beeep fallback\n")
			return nil
		}
		wm.printf("✓ System notification sent via osascript\n")
		return nil
	}

	// For other platforms, use beeep
	err := beeep.Notify(title, message, iconPath)
	if err != nil {
		wm.printf("Warning: Failed to send system notification: %v\n", err)
		return nil
	}

	wm.printf("✓ System notification sent via beeep\n")
	return nil
}

// getDefaultIcon returns a default icon path based on the platform
func getDefaultIcon() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: Try to use Terminal app icon
		return "/System/Library/CoreServices/Terminal.app/Contents/Resources/Terminal.icns"
	case "windows":
		// Windows: Use information icon
		return "info"
	default:
		// Linux and others: Use information icon
		return "info"
	}
}

// ExecuteClaudeNotificationHooks executes notification hooks and sends system notification
func (wm *WorktreeManager) ExecuteClaudeNotificationHooks() error {
	// Read input from stdin
	var input NotificationInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to decode Notification input: %w", err)
	}

	// Set environment variable for hook scripts
	os.Setenv("MESSAGE", input.Message)
	defer os.Unsetenv("MESSAGE")

	// Get configured hooks
	if wm.Config != nil && wm.Config.Hooks != nil && len(wm.Config.Hooks.ClaudeNotification) > 0 {
		// Execute the notification hooks
		if err := wm.ExecuteHooks(wm.Config.Hooks.ClaudeNotification, input.CWD, "claude_notification"); err != nil {
			// Log but don't fail
			if wm.Options.Verbose {
				wm.printf("Warning: Some notification hooks failed: %v\n", err)
			}
		}
	}

	// Send system notification after hooks
	if err := wm.SendSystemNotification(&input); err != nil {
		return fmt.Errorf("failed to send system notification: %w", err)
	}

	return nil
}

// ParseNotificationMessage extracts key information from Claude notification messages
func ParseNotificationMessage(message string) (string, string) {
	// Common Claude notification patterns
	if strings.Contains(message, "needs your permission") {
		return "Permission Required", message
	}
	if strings.Contains(message, "waiting for your input") {
		return "Input Required", message
	}
	if strings.Contains(message, "Task completed") {
		return "Task Complete", message
	}
	if strings.Contains(message, "Error occurred") {
		return "Error", message
	}

	// Default case
	return "Claude Code", message
}
