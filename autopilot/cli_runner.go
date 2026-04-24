package autopilot

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type cliSessionRunner struct {
	backend    Backend
	showOutput bool
}

func newCLIRunner(opts PipelineOptions) SessionRunner {
	return &cliSessionRunner{backend: opts.Backend, showOutput: opts.ShowOutput}
}

func (r *cliSessionRunner) Preflight(ctx context.Context, req SessionRequest) error {
	if _, err := exec.LookPath(r.backend.Binary()); err != nil {
		return fmt.Errorf("preflight check failed: %s CLI not found in PATH", r.backend.Binary())
	}

	probe := req
	if probe.Prompt == "" {
		probe.Prompt = "say ok"
	}
	if err := r.Run(ctx, probe, io.Discard, io.Discard); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			if r.backend == BackendClaude {
				return fmt.Errorf("preflight check failed: claude did not respond within 30s - an MCP server may be hanging")
			}
			return fmt.Errorf("preflight check failed: %s did not respond within 30s", r.backend)
		}
		if r.backend == BackendCodex {
			return fmt.Errorf("preflight check failed: %w (if this is an auth error, run 'codex login')", err)
		}
		return fmt.Errorf("preflight check failed: %w", err)
	}

	return nil
}

func (r *cliSessionRunner) Run(ctx context.Context, req SessionRequest, stdout, stderr io.Writer) error {
	execStart := time.Now()
	cmdSpec, err := BuildSessionCommand(r.backend, req.Model, req.Prompt, r.showOutput, req.AutoApprove)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, cmdSpec.Binary, cmdSpec.Args...)
	cmd.Dir = req.WorkingDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s session failed to start: %w", r.backend, err)
	}

	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			elapsed := time.Since(execStart)
			return fmt.Errorf("phase timed out after %s (pid=%d killed)", formatDuration(elapsed), pid)
		}
		return fmt.Errorf("%s session failed: %w", r.backend, err)
	}

	return nil
}
