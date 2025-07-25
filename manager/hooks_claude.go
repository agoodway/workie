package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agoodway/workie/ai"
	"github.com/agoodway/workie/hooks"
)

// ExecuteClaudePreToolUseHooks executes PreToolUse hooks with AI decision support
// It reads the hook input from stdin, executes hooks, and returns the decision as JSON
func (wm *WorktreeManager) ExecuteClaudePreToolUseHooks(enableAI bool) error {
	// Read input from stdin
	var input hooks.PreToolUseInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to decode PreToolUse input: %w", err)
	}

	// Get configured hooks
	if wm.Config == nil || wm.Config.Hooks == nil || len(wm.Config.Hooks.ClaudePreToolUse) == 0 {
		// No hooks configured, return undefined decision
		decision := &hooks.HookDecision{}
		return wm.outputDecision(decision)
	}

	// Execute the hooks
	hookResults := wm.executeHooksForDecision(wm.Config.Hooks.ClaudePreToolUse, input.CWD)

	var decision *hooks.HookDecision

	if enableAI && wm.Config.IsAIEnabled() {
		// Use AI to make the decision
		aiService, err := ai.NewService(wm.Config)
		if err != nil {
			wm.printf("Warning: Failed to create AI service: %v\n", err)
			// Fall back to rule-based decision
			decision = wm.makeRuleBasedDecision(hookResults)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			decision, err = aiService.AnalyzeToolUse(ctx, &input, hookResults)
			if err != nil {
				wm.printf("Warning: AI analysis failed: %v\n", err)
				// Fall back to rule-based decision
				decision = wm.makeRuleBasedDecision(hookResults)
			}
		}
	} else {
		// Make rule-based decision without AI
		decision = wm.makeRuleBasedDecision(hookResults)
	}

	// Validate and output the decision
	if err := decision.Validate(); err != nil {
		return fmt.Errorf("invalid decision: %w", err)
	}

	return wm.outputDecision(decision)
}

// executeHooksForDecision executes hooks and collects results for decision making
func (wm *WorktreeManager) executeHooksForDecision(hookCommands []string, workDir string) []hooks.HookExecutionResult {
	results := make([]hooks.HookExecutionResult, 0, len(hookCommands))

	for i, hookCommand := range hookCommands {
		managerResult := wm.executeHookCommand(hookCommand, workDir, i+1)
		// Convert to hooks.HookExecutionResult
		result := hooks.HookExecutionResult{
			Index:    managerResult.Index,
			Command:  managerResult.Command,
			Success:  managerResult.Success,
			Duration: managerResult.Duration,
			ExitCode: managerResult.ExitCode,
			Stdout:   managerResult.Stdout,
			Stderr:   managerResult.Stderr,
			Error:    managerResult.Error,
			TimedOut: managerResult.TimedOut,
		}
		results = append(results, result)
	}

	return results
}

// makeRuleBasedDecision makes a decision based on hook results without AI
func (wm *WorktreeManager) makeRuleBasedDecision(hookResults []hooks.HookExecutionResult) *hooks.HookDecision {
	decision := &hooks.HookDecision{}

	// Check if any hooks failed
	hasFailures := false
	var failureReasons []string

	for _, result := range hookResults {
		if result.ExitCode != 0 || result.Error != nil {
			hasFailures = true
			if result.Error != nil {
				failureReasons = append(failureReasons, fmt.Sprintf("Hook failed: %v", result.Error))
			} else {
				failureReasons = append(failureReasons, fmt.Sprintf("Hook exited with code %d", result.ExitCode))
			}
		}

		// Check for explicit block signals in output
		if containsBlockSignal(result.Stdout) || containsBlockSignal(result.Stderr) {
			decision.Decision = "block"
			decision.Reason = "Hook output indicates tool should be blocked"
			return decision
		}
	}

	// If hooks failed, block by default
	if hasFailures {
		decision.Decision = "block"
		if len(failureReasons) > 0 {
			decision.Reason = failureReasons[0]
		} else {
			decision.Reason = "One or more hooks failed"
		}
		return decision
	}

	// If all hooks passed, let normal flow continue (undefined decision)
	return decision
}

// containsBlockSignal checks if the output contains explicit block signals
func containsBlockSignal(output string) bool {
	blockSignals := []string{
		"BLOCK",
		"DENY",
		"FORBIDDEN",
		"NOT ALLOWED",
		"SECURITY VIOLATION",
		"POLICY VIOLATION",
	}

	upperOutput := strings.ToUpper(output)
	for _, signal := range blockSignals {
		if strings.Contains(upperOutput, signal) {
			return true
		}
	}
	return false
}

// outputDecision outputs the decision as JSON to stdout
func (wm *WorktreeManager) outputDecision(decision *hooks.HookDecision) error {
	data, err := decision.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal decision: %w", err)
	}

	// Write to stdout for Claude Code
	fmt.Println(string(data))
	return nil
}
