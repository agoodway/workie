package provider

import (
	"testing"
)

func TestParseIssueReference(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		wantProvider string
		wantIssueID  string
		wantErr      bool
	}{
		{
			name:         "GitHub issue",
			ref:          "github:123",
			wantProvider: "github",
			wantIssueID:  "123",
			wantErr:      false,
		},
		{
			name:         "Jira issue",
			ref:          "jira:PROJ-456",
			wantProvider: "jira",
			wantIssueID:  "PROJ-456",
			wantErr:      false,
		},
		{
			name:         "Linear issue",
			ref:          "linear:TEAM-789",
			wantProvider: "linear",
			wantIssueID:  "TEAM-789",
			wantErr:      false,
		},
		{
			name:         "With spaces",
			ref:          "github: 123 ",
			wantProvider: "github",
			wantIssueID:  "123",
			wantErr:      false,
		},
		{
			name:         "Invalid format - no colon",
			ref:          "github123",
			wantProvider: "",
			wantIssueID:  "",
			wantErr:      true,
		},
		{
			name:         "Invalid format - empty provider",
			ref:          ":123",
			wantProvider: "",
			wantIssueID:  "",
			wantErr:      true,
		},
		{
			name:         "Invalid format - empty ID",
			ref:          "github:",
			wantProvider: "",
			wantIssueID:  "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, issueID, err := ParseIssueReference(tt.ref)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseIssueReference() error = nil, wantErr true")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseIssueReference() error = %v, wantErr false", err)
				return
			}

			if provider != tt.wantProvider {
				t.Errorf("ParseIssueReference() provider = %v, want %v", provider, tt.wantProvider)
			}

			if issueID != tt.wantIssueID {
				t.Errorf("ParseIssueReference() issueID = %v, want %v", issueID, tt.wantIssueID)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple title",
			input:    "Fix bug in login",
			expected: "fix-bug-in-login",
		},
		{
			name:     "Title with special characters",
			input:    "Add feature: user@email.com support!",
			expected: "add-feature-user-email-com-support",
		},
		{
			name:     "Title with multiple spaces",
			input:    "Fix   multiple   spaces",
			expected: "fix-multiple-spaces",
		},
		{
			name:     "Title with slashes",
			input:    "Fix bug/feature in module/component",
			expected: "fix-bug-feature-in-module-component",
		},
		{
			name:     "Very long title",
			input:    "This is a very long title that exceeds the maximum length allowed for branch names in git repositories",
			expected: "this-is-a-very-long-title-that-exceeds-the-maximum-length-allow",
		},
		{
			name:     "Title with brackets and quotes",
			input:    "[BUG] Fix \"critical\" issue (urgent)",
			expected: "bug-fix-critical-issue-urgent",
		},
		{
			name:     "Title ending with special characters",
			input:    "Fix trailing hyphens---",
			expected: "fix-trailing-hyphens",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only special characters",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	t.Run("Register and Get providers", func(t *testing.T) {
		registry := NewRegistry()

		// Create a mock provider
		mockProvider := &mockProvider{name: "test"}

		// Register provider
		err := registry.Register(mockProvider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		// Try to register same provider again
		err = registry.Register(mockProvider)
		if err == nil {
			t.Error("Expected error when registering duplicate provider")
		}

		// Get provider
		provider, err := registry.Get("test")
		if err != nil {
			t.Fatalf("Failed to get provider: %v", err)
		}

		if provider.Name() != "test" {
			t.Errorf("Got provider name %s, want test", provider.Name())
		}

		// Get non-existent provider
		_, err = registry.Get("nonexistent")
		if err == nil {
			t.Error("Expected error when getting non-existent provider")
		}
	})

	t.Run("List providers", func(t *testing.T) {
		registry := NewRegistry()

		// Register multiple providers
		providers := []Provider{
			&mockProvider{name: "github", configured: true},
			&mockProvider{name: "jira", configured: false},
			&mockProvider{name: "linear", configured: true},
		}

		for _, p := range providers {
			registry.Register(p)
		}

		// List all providers
		allProviders := registry.List()
		if len(allProviders) != 3 {
			t.Errorf("Expected 3 providers, got %d", len(allProviders))
		}

		// List configured providers
		configuredProviders := registry.ListConfigured()
		if len(configuredProviders) != 2 {
			t.Errorf("Expected 2 configured providers, got %d", len(configuredProviders))
		}
	})
}

// mockProvider is a test implementation of the Provider interface
type mockProvider struct {
	name       string
	configured bool
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) ListIssues(filter ListFilter) (*IssueList, error) {
	return &IssueList{Issues: []Issue{}}, nil
}

func (m *mockProvider) GetIssue(issueID string) (*Issue, error) {
	return &Issue{
		ID:       issueID,
		Title:    "Test Issue",
		Provider: m.name,
	}, nil
}

func (m *mockProvider) CreateBranchName(issue *Issue) string {
	return "test-branch"
}

func (m *mockProvider) ValidateConfig() error {
	return nil
}

func (m *mockProvider) IsConfigured() bool {
	return m.configured
}
