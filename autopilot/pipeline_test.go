package autopilot

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeSessionRunner struct {
	preflightReqs []SessionRequest
	runReqs       []SessionRequest
	preflightErr  error
	runErr        error
	stdoutText    string
	stderrText    string
}

func (f *fakeSessionRunner) Preflight(_ context.Context, req SessionRequest) error {
	f.preflightReqs = append(f.preflightReqs, req)
	return f.preflightErr
}

func (f *fakeSessionRunner) Run(_ context.Context, req SessionRequest, stdout, stderr io.Writer) error {
	f.runReqs = append(f.runReqs, req)
	if f.stdoutText != "" {
		_, _ = io.WriteString(stdout, f.stdoutText)
	}
	if f.stderrText != "" {
		_, _ = io.WriteString(stderr, f.stderrText)
	}
	return f.runErr
}

type fakeNotifier struct {
	notifications []string
	err           error
}

func (f *fakeNotifier) Notify(_ context.Context, title, message string) error {
	f.notifications = append(f.notifications, title+"|"+message)
	return f.err
}

func TestValidateOptions(t *testing.T) {
	valid := PipelineOptions{
		Changes:      []string{"add-user-auth"},
		StartPhase:   1,
		Backend:      BackendOpencode,
		Model:        "openai/gpt-5.5",
		PhaseTimeout: 45 * time.Minute,
	}

	t.Run("valid", func(t *testing.T) {
		if err := validateOptions(valid); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("invalid backend", func(t *testing.T) {
		opts := valid
		opts.Backend = Backend("other")
		err := validateOptions(opts)
		if err == nil {
			t.Fatalf("expected error for invalid backend")
		}
		if !strings.Contains(err.Error(), SupportedBackendsList) {
			t.Fatalf("expected supported backend list in error, got %v", err)
		}
	})

	t.Run("empty model", func(t *testing.T) {
		opts := valid
		opts.Model = ""
		if err := validateOptions(opts); err == nil {
			t.Fatalf("expected error for empty model")
		}
	})

	t.Run("invalid phase", func(t *testing.T) {
		opts := valid
		opts.StartPhase = 6
		if err := validateOptions(opts); err == nil {
			t.Fatalf("expected error for invalid phase")
		}
	})
}

func TestNewPipelineAppliesDefaults(t *testing.T) {
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
	})

	if p.opts.Backend != BackendClaude {
		t.Fatalf("expected default backend claude, got %q", p.opts.Backend)
	}
	if p.opts.Model != "opus" {
		t.Fatalf("expected default model opus, got %q", p.opts.Model)
	}
}

func TestNewPipelineAppliesBackendDefaultModel(t *testing.T) {
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
		Backend:    BackendOpencode,
	})

	if p.opts.Model != "openai/gpt-5.5" {
		t.Fatalf("expected opencode default model openai/gpt-5.5, got %q", p.opts.Model)
	}
}

func TestPreflightCheckDelegatesToRunner(t *testing.T) {
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
		Backend:    BackendOpencode,
		Model:      "openai/gpt-5.5",
	})
	runner := &fakeSessionRunner{}
	p.runner = runner

	if err := p.preflightCheck(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(runner.preflightReqs) != 1 {
		t.Fatalf("expected one preflight request, got %d", len(runner.preflightReqs))
	}
	if runner.preflightReqs[0].Prompt != "say ok" {
		t.Fatalf("expected preflight prompt 'say ok', got %q", runner.preflightReqs[0].Prompt)
	}
}

func TestExecuteSessionWritesRunnerOutput(t *testing.T) {
	logDir := t.TempDir()
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
		Backend:    BackendOpencode,
		Model:      "openai/gpt-5.5",
		LogDir:     logDir,
	})
	runner := &fakeSessionRunner{stdoutText: "hello from runner\n"}
	p.runner = runner

	logFile := filepath.Join(logDir, "phase.log")
	if err := p.executeSession(context.Background(), "prompt body", logFile); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(runner.runReqs) != 1 {
		t.Fatalf("expected one run request, got %d", len(runner.runReqs))
	}
	if runner.runReqs[0].Prompt != "prompt body" {
		t.Fatalf("expected prompt body, got %q", runner.runReqs[0].Prompt)
	}

	contents, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
	if string(contents) != "hello from runner\n" {
		t.Fatalf("expected log file to contain runner output, got %q", string(contents))
	}
}

func TestRunChangeSendsProgressNotifications(t *testing.T) {
	logDir := t.TempDir()
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
		Backend:    BackendOpencode,
		Model:      "openai/gpt-5.5",
		LogDir:     logDir,
		NoCommit:   true,
	})
	p.projectDir = t.TempDir()
	p.runner = &fakeSessionRunner{}
	notifier := &fakeNotifier{}
	p.notifier = notifier

	result := p.runChange("add-user-auth", 1, 0)
	if !result.Success {
		t.Fatalf("expected success, got failure: %#v", result)
	}
	if len(notifier.notifications) != 7 {
		t.Fatalf("expected 7 notifications, got %d: %#v", len(notifier.notifications), notifier.notifications)
	}
	if !strings.Contains(notifier.notifications[0], "Change started|add-user-auth starting at phase 1") {
		t.Fatalf("unexpected first notification %q", notifier.notifications[0])
	}
	if !strings.Contains(notifier.notifications[1], "Phase 1 completed|add-user-auth: prepare-for-takeoff") {
		t.Fatalf("unexpected phase completion notification %q", notifier.notifications[1])
	}
	if !strings.Contains(notifier.notifications[6], "Change completed|add-user-auth finished in") {
		t.Fatalf("unexpected final notification %q", notifier.notifications[6])
	}
}

func TestRunChangeSendsFailureNotifications(t *testing.T) {
	logDir := t.TempDir()
	p := NewPipeline(PipelineOptions{
		Changes:    []string{"add-user-auth"},
		StartPhase: 1,
		Backend:    BackendOpencode,
		Model:      "openai/gpt-5.5",
		LogDir:     logDir,
		NoCommit:   true,
	})
	p.projectDir = t.TempDir()
	p.runner = &fakeSessionRunner{runErr: errors.New("boom")}
	notifier := &fakeNotifier{}
	p.notifier = notifier

	result := p.runChange("add-user-auth", 1, 0)
	if result.Success {
		t.Fatal("expected failure")
	}
	if len(notifier.notifications) != 3 {
		t.Fatalf("expected 3 notifications, got %d: %#v", len(notifier.notifications), notifier.notifications)
	}
	if !strings.Contains(notifier.notifications[1], "Phase 1 failed|add-user-auth: prepare-for-takeoff") {
		t.Fatalf("unexpected phase failure notification %q", notifier.notifications[1])
	}
	if !strings.Contains(notifier.notifications[2], "Change failed|add-user-auth failed at phase 1") {
		t.Fatalf("unexpected change failure notification %q", notifier.notifications[2])
	}
}
