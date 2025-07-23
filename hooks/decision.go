package hooks

import (
	"encoding/json"
	"fmt"
	"time"
)

// PreToolUseInput represents the input that Claude Code sends to PreToolUse hooks
type PreToolUseInput struct {
	SessionID      string                 `json:"session_id"`
	TranscriptPath string                 `json:"transcript_path"`
	CWD            string                 `json:"cwd"`
	HookEventName  string                 `json:"hook_event_name"`
	ToolName       string                 `json:"tool_name"`
	ToolInput      map[string]interface{} `json:"tool_input"`
}

// HookDecision represents the decision response for Claude Code hooks
type HookDecision struct {
	Decision string `json:"decision,omitempty"` // "approve", "block", or undefined
	Reason   string `json:"reason,omitempty"`   // Explanation for the decision
}

// ParsePreToolUseInput parses JSON input from Claude Code
func ParsePreToolUseInput(data []byte) (*PreToolUseInput, error) {
	var input PreToolUseInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse PreToolUse input: %w", err)
	}
	return &input, nil
}

// ToJSON converts the decision to JSON format expected by Claude Code
func (d *HookDecision) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// IsApprove returns true if the decision is to approve
func (d *HookDecision) IsApprove() bool {
	return d.Decision == "approve"
}

// IsBlock returns true if the decision is to block
func (d *HookDecision) IsBlock() bool {
	return d.Decision == "block"
}

// IsUndefined returns true if the decision is undefined (continue with normal flow)
func (d *HookDecision) IsUndefined() bool {
	return d.Decision == ""
}

// Validate checks if the decision is valid
func (d *HookDecision) Validate() error {
	if d.Decision != "" && d.Decision != "approve" && d.Decision != "block" {
		return fmt.Errorf("invalid decision value: %s (must be 'approve', 'block', or empty)", d.Decision)
	}
	return nil
}

// HookExecutionResult represents the result of executing a single hook
type HookExecutionResult struct {
	Index    int
	Command  string
	Success  bool
	Duration time.Duration
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
	TimedOut bool
}