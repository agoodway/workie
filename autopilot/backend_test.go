package autopilot

import "testing"

func TestParseBackend(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Backend
		wantErr bool
	}{
		{name: "claude", input: "claude", want: BackendClaude},
		{name: "codex", input: "codex", want: BackendCodex},
		{name: "opencode", input: "opencode", want: BackendOpencode},
		{name: "normalized", input: "  CoDeX  ", want: BackendCodex},
		{name: "invalid", input: "other", wantErr: true},
		{name: "empty", input: "  ", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBackend(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestDefaultModelForBackend(t *testing.T) {
	if got := DefaultModelForBackend(BackendClaude); got != "opus" {
		t.Fatalf("expected claude default model opus, got %q", got)
	}
	if got := DefaultModelForBackend(BackendCodex); got != "gpt-5.3-codex" {
		t.Fatalf("expected codex default model gpt-5.3-codex, got %q", got)
	}
	if got := DefaultModelForBackend(BackendOpencode); got != "openai/gpt-5.5" {
		t.Fatalf("expected opencode default model openai/gpt-5.5, got %q", got)
	}
}

func TestBuildSessionCommand(t *testing.T) {
	t.Run("claude args", func(t *testing.T) {
		cmd, err := BuildSessionCommand(BackendClaude, "opus", "prompt", true, true)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cmd.Binary != "claude" {
			t.Fatalf("expected binary claude, got %q", cmd.Binary)
		}
		want := []string{"--print", "--model", "opus", "--verbose", "--dangerously-skip-permissions", "prompt"}
		assertArgsEqual(t, cmd.Args, want)
	})

	t.Run("codex args", func(t *testing.T) {
		cmd, err := BuildSessionCommand(BackendCodex, "gpt-5.3-codex", "prompt", true, true)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cmd.Binary != "codex" {
			t.Fatalf("expected binary codex, got %q", cmd.Binary)
		}
		want := []string{"exec", "--model", "openai/gpt-5.3-codex", "--sandbox", "workspace-write", "--ask-for-approval", "never", "prompt"}
		assertArgsEqual(t, cmd.Args, want)
	})

	t.Run("codex model passthrough provider", func(t *testing.T) {
		cmd, err := BuildSessionCommand(BackendCodex, "opencode/gpt-5.3-codex", "prompt", false, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		want := []string{"exec", "--model", "opencode/gpt-5.3-codex", "--sandbox", "workspace-write", "--ask-for-approval", "never", "prompt"}
		assertArgsEqual(t, cmd.Args, want)
	})

	t.Run("opencode uses server transport", func(t *testing.T) {
		_, err := BuildSessionCommand(BackendOpencode, "openai/gpt-5.5", "prompt", true, true)
		if err == nil {
			t.Fatalf("expected error for opencode server transport")
		}
	})

	t.Run("invalid backend", func(t *testing.T) {
		_, err := BuildSessionCommand(Backend("bogus"), "model", "prompt", false, false)
		if err == nil {
			t.Fatalf("expected error for invalid backend")
		}
	})
}

func assertArgsEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d args, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}
