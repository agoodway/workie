package autopilot

import (
	"context"
	"errors"
	"testing"
)

func TestOSNotifierDarwinUsesOsaScript(t *testing.T) {
	var gotName string
	var gotArgs []string
	notifier := newOSNotifier("darwin", func(_ context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil
	})

	if err := notifier.Notify(context.Background(), "Phase 1 completed", "prepare-for-takeoff in 12s"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotName != "osascript" {
		t.Fatalf("expected osascript, got %q", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-e" {
		t.Fatalf("expected osascript -e invocation, got %#v", gotArgs)
	}
	if gotArgs[1] == "" {
		t.Fatal("expected AppleScript payload")
	}
}

func TestOSNotifierLinuxUsesNotifySend(t *testing.T) {
	var gotName string
	var gotArgs []string
	notifier := newOSNotifier("linux", func(_ context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil
	})

	if err := notifier.Notify(context.Background(), "Phase 2 failed", "takeoff after 3m"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotName != "notify-send" {
		t.Fatalf("expected notify-send, got %q", gotName)
	}
	if len(gotArgs) != 2 {
		t.Fatalf("expected notify-send args, got %#v", gotArgs)
	}
	if gotArgs[0] != "Workie Autopilot: Phase 2 failed" {
		t.Fatalf("unexpected notify-send title %q", gotArgs[0])
	}
	if gotArgs[1] != "takeoff after 3m" {
		t.Fatalf("unexpected notify-send body %q", gotArgs[1])
	}
}

func TestNoopNotifierDoesNothing(t *testing.T) {
	if err := newNotifier(false).Notify(context.Background(), "title", "message"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQuoteAppleScriptStringEscapesQuotes(t *testing.T) {
	got := quoteAppleScriptString(`say "hi"`)
	want := `"say \"hi\""`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestOSNotifierPropagatesCommandError(t *testing.T) {
	wantErr := errors.New("boom")
	notifier := newOSNotifier("darwin", func(_ context.Context, name string, args ...string) error {
		return wantErr
	})

	err := notifier.Notify(context.Background(), "title", "message")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
