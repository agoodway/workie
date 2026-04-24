package autopilot

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Notifier interface {
	Notify(ctx context.Context, title, message string) error
}

type noopNotifier struct{}

func (noopNotifier) Notify(context.Context, string, string) error {
	return nil
}

type commandRunner func(ctx context.Context, name string, args ...string) error

type osNotifier struct {
	goos string
	run  commandRunner
}

func newNotifier(enabled bool) Notifier {
	if !enabled {
		return noopNotifier{}
	}

	return newOSNotifier(runtime.GOOS, nil)
}

func newOSNotifier(goos string, run commandRunner) Notifier {
	if run == nil {
		run = runCommand
	}

	return &osNotifier{goos: goos, run: run}
}

func runCommand(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func (n *osNotifier) Notify(ctx context.Context, title, message string) error {
	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	if title == "" && message == "" {
		return nil
	}

	switch n.goos {
	case "darwin":
		script := fmt.Sprintf(
			"display notification %s with title %s subtitle %s",
			quoteAppleScriptString(message),
			quoteAppleScriptString("Workie Autopilot"),
			quoteAppleScriptString(title),
		)
		return n.run(ctx, "osascript", "-e", script)
	case "linux":
		linuxTitle := strings.TrimSpace("Workie Autopilot: " + title)
		if linuxTitle == "Workie Autopilot:" {
			linuxTitle = "Workie Autopilot"
		}
		return n.run(ctx, "notify-send", linuxTitle, message)
	default:
		return nil
	}
}

func quoteAppleScriptString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}
