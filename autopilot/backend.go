package autopilot

import (
	"fmt"
	"strings"
)

type Backend string

const (
	BackendClaude   Backend = "claude"
	BackendCodex    Backend = "codex"
	BackendOpencode Backend = "opencode"
)

const (
	SupportedBackendsList = "claude, codex, opencode"

	DefaultOpencodeServerURL         = "http://127.0.0.1:4096"
	DefaultOpencodeServerUsername    = "opencode"
	DefaultOpencodeServerUsernameEnv = "OPENCODE_SERVER_USERNAME"
	DefaultOpencodeServerPasswordEnv = "OPENCODE_SERVER_PASSWORD"

	defaultClaudeModel   = "opus"
	defaultCodexModel    = "gpt-5.3-codex"
	defaultOpencodeModel = "openai/gpt-5.5"
)

type SessionCommand struct {
	Binary string
	Args   []string
}

func ParseBackend(raw string) (Backend, error) {
	b := Backend(strings.ToLower(strings.TrimSpace(raw)))
	if b == "" {
		return "", fmt.Errorf("backend cannot be empty")
	}
	if !b.IsValid() {
		return "", fmt.Errorf("invalid backend %q (must be one of: %s)", raw, SupportedBackendsList)
	}
	return b, nil
}

func (b Backend) IsValid() bool {
	return b == BackendClaude || b == BackendCodex || b == BackendOpencode
}

func (b Backend) Binary() string {
	switch b {
	case BackendCodex:
		return "codex"
	case BackendOpencode:
		return "opencode"
	default:
		return "claude"
	}
}

func normalizeCodexModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if strings.Contains(trimmed, "/") {
		return trimmed
	}
	return "openai/" + trimmed
}

func DefaultModelForBackend(backend Backend) string {
	switch backend {
	case BackendCodex:
		return defaultCodexModel
	case BackendOpencode:
		return defaultOpencodeModel
	default:
		return defaultClaudeModel
	}
}

func BuildSessionCommand(backend Backend, model, prompt string, showOutput, autoApprove bool) (SessionCommand, error) {
	if !backend.IsValid() {
		return SessionCommand{}, fmt.Errorf("invalid backend %q", backend)
	}
	if backend == BackendOpencode {
		return SessionCommand{}, fmt.Errorf("backend %q uses the opencode server transport", backend)
	}

	args := []string{}
	if backend == BackendCodex {
		args = append(args,
			"exec",
			"--model", normalizeCodexModel(model),
			"--sandbox", "workspace-write",
			"--ask-for-approval", "never",
		)
		args = append(args, prompt)
		return SessionCommand{Binary: backend.Binary(), Args: args}, nil
	}

	args = append(args, "--print", "--model", model)
	if showOutput {
		args = append(args, "--verbose")
	}
	if autoApprove {
		args = append(args, "--dangerously-skip-permissions")
	}
	args = append(args, prompt)

	return SessionCommand{Binary: backend.Binary(), Args: args}, nil
}
