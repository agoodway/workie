package manager

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/agoodway/workie/config"
)

func TestParseNotificationMessage(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		expectedTitle string
		expectedMsg   string
	}{
		{
			name:          "Permission required",
			message:       "Claude needs your permission to use Bash",
			expectedTitle: "Permission Required",
			expectedMsg:   "Claude needs your permission to use Bash",
		},
		{
			name:          "Input required",
			message:       "Claude is waiting for your input",
			expectedTitle: "Input Required",
			expectedMsg:   "Claude is waiting for your input",
		},
		{
			name:          "Task completed",
			message:       "Task completed successfully",
			expectedTitle: "Task Complete",
			expectedMsg:   "Task completed successfully",
		},
		{
			name:          "Error occurred",
			message:       "Error occurred while processing",
			expectedTitle: "Error",
			expectedMsg:   "Error occurred while processing",
		},
		{
			name:          "Generic message",
			message:       "Some other notification",
			expectedTitle: "Claude Code",
			expectedMsg:   "Some other notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, msg := ParseNotificationMessage(tt.message)
			if title != tt.expectedTitle {
				t.Errorf("ParseNotificationMessage() title = %v, want %v", title, tt.expectedTitle)
			}
			if msg != tt.expectedMsg {
				t.Errorf("ParseNotificationMessage() msg = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}
}

func TestSendSystemNotification(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "workie-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test cases
	tests := []struct {
		name       string
		config     *config.Config
		input      *NotificationInput
		shouldSend bool
	}{
		{
			name: "Notifications disabled",
			config: &config.Config{
				Hooks: &config.Hooks{
					SystemNotifications: &config.SystemNotificationConfig{
						Enabled: false,
					},
				},
			},
			input: &NotificationInput{
				Message: "Test message",
			},
			shouldSend: false,
		},
		{
			name: "Notifications enabled with default title",
			config: &config.Config{
				Hooks: &config.Hooks{
					SystemNotifications: &config.SystemNotificationConfig{
						Enabled: true,
					},
				},
			},
			input: &NotificationInput{
				Message: "Test message",
			},
			shouldSend: true,
		},
		{
			name: "Notifications enabled with custom title",
			config: &config.Config{
				Hooks: &config.Hooks{
					SystemNotifications: &config.SystemNotificationConfig{
						Enabled: true,
						Title:   "Custom Title",
					},
				},
			},
			input: &NotificationInput{
				Message: "Test message",
			},
			shouldSend: true,
		},
		{
			name: "Notifications with icon",
			config: &config.Config{
				Hooks: &config.Hooks{
					SystemNotifications: &config.SystemNotificationConfig{
						Enabled: true,
						Title:   "Custom Title",
						Icon:    "test-icon.png",
					},
				},
			},
			input: &NotificationInput{
				Message: "Test message",
			},
			shouldSend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := &WorktreeManager{
				Config:   tt.config,
				RepoPath: tmpDir,
				Options:  Options{Verbose: false},
			}

			// We can't actually test the notification sending without
			// triggering real system notifications, so we just ensure
			// the function doesn't panic
			err := wm.SendSystemNotification(tt.input)
			if err != nil {
				t.Errorf("SendSystemNotification() error = %v", err)
			}
		})
	}
}

func TestNotificationInputJSON(t *testing.T) {
	// Test JSON parsing of notification input
	jsonData := `{
		"session_id": "test-session",
		"transcript_path": "/path/to/transcript.jsonl",
		"cwd": "/working/directory",
		"hook_event_name": "Notification",
		"message": "Claude needs your permission to use Bash"
	}`

	var input NotificationInput
	err := json.Unmarshal([]byte(jsonData), &input)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if input.SessionID != "test-session" {
		t.Errorf("Expected session_id 'test-session', got '%s'", input.SessionID)
	}
	if input.Message != "Claude needs your permission to use Bash" {
		t.Errorf("Expected message 'Claude needs your permission to use Bash', got '%s'", input.Message)
	}
	if input.HookEventName != "Notification" {
		t.Errorf("Expected hook_event_name 'Notification', got '%s'", input.HookEventName)
	}
}

func TestGetDefaultIcon(t *testing.T) {
	// Test that getDefaultIcon returns a non-empty string
	icon := getDefaultIcon()
	if icon == "" {
		t.Error("getDefaultIcon() returned empty string")
	}
}
