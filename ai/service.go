package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agoodway/workie/config"
	"github.com/agoodway/workie/hooks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// Service provides AI-powered decision making for hooks
type Service struct {
	llm    llms.Model
	config *config.Config
}

// NewService creates a new AI service
func NewService(cfg *config.Config) (*Service, error) {
	if cfg == nil || !cfg.IsAIEnabled() {
		return nil, fmt.Errorf("AI is not enabled in configuration")
	}

	// Create Ollama client
	opts := []ollama.Option{
		ollama.WithModel(cfg.AI.Model.Name),
	}

	if cfg.AI.Ollama.BaseURL != "" {
		opts = append(opts, ollama.WithServerURL(cfg.AI.Ollama.BaseURL))
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &Service{
		llm:    llm,
		config: cfg,
	}, nil
}

// AnalyzeToolUse analyzes a tool use request and hook outputs to make a decision
func (s *Service) AnalyzeToolUse(ctx context.Context, input *hooks.PreToolUseInput, hookResults []hooks.HookExecutionResult) (*hooks.HookDecision, error) {
	// Build the prompt for the LLM
	prompt := s.buildDecisionPrompt(input, hookResults)

	// Call the LLM
	response, err := s.llm.Call(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	// Parse the LLM response into a decision
	decision := s.parseDecision(response, hookResults)

	return decision, nil
}

// CallLLM directly calls the LLM with a prompt
func (s *Service) CallLLM(ctx context.Context, prompt string) (string, error) {
	return s.llm.Call(ctx, prompt)
}

// buildDecisionPrompt creates the prompt for the LLM to analyze the tool use
func (s *Service) buildDecisionPrompt(input *hooks.PreToolUseInput, hookResults []hooks.HookExecutionResult) string {
	var prompt strings.Builder

	prompt.WriteString("You are a security policy enforcer for Claude Code. ")
	prompt.WriteString("Analyze the following tool use request and hook script outputs to decide if it should be allowed.\n\n")

	// Tool information
	prompt.WriteString(fmt.Sprintf("Tool Name: %s\n", input.ToolName))
	prompt.WriteString("Tool Parameters:\n")

	// Pretty print tool input
	if inputJSON, err := json.MarshalIndent(input.ToolInput, "  ", "  "); err == nil {
		prompt.WriteString(string(inputJSON))
	} else {
		prompt.WriteString(fmt.Sprintf("  %v", input.ToolInput))
	}
	prompt.WriteString("\n\n")

	// Hook outputs
	prompt.WriteString("Hook Script Outputs:\n")
	prompt.WriteString(strings.Repeat("=", 50) + "\n")

	for i, result := range hookResults {
		prompt.WriteString(fmt.Sprintf("\nHook %d: %s\n", i+1, result.Command))
		prompt.WriteString(fmt.Sprintf("Exit Code: %d\n", result.ExitCode))

		if result.Stdout != "" {
			prompt.WriteString("Standard Output:\n")
			prompt.WriteString(result.Stdout)
			if !strings.HasSuffix(result.Stdout, "\n") {
				prompt.WriteString("\n")
			}
		}

		if result.Stderr != "" {
			prompt.WriteString("Standard Error:\n")
			prompt.WriteString(result.Stderr)
			if !strings.HasSuffix(result.Stderr, "\n") {
				prompt.WriteString("\n")
			}
		}

		if result.Error != nil {
			prompt.WriteString(fmt.Sprintf("Execution Error: %v\n", result.Error))
		}

		if result.TimedOut {
			prompt.WriteString("⚠️  Hook timed out\n")
		}

		prompt.WriteString(strings.Repeat("-", 30) + "\n")
	}

	// Decision instructions
	prompt.WriteString("\nBased on the hook outputs and tool parameters, decide whether to:\n")
	prompt.WriteString("- APPROVE: Allow the tool to execute (if hooks indicate it's safe)\n")
	prompt.WriteString("- BLOCK: Prevent the tool from executing (if hooks indicate risks)\n\n")

	prompt.WriteString("Consider:\n")
	prompt.WriteString("1. Any security warnings or errors from hook scripts\n")
	prompt.WriteString("2. Exit codes (non-zero usually indicates problems)\n")
	prompt.WriteString("3. Explicit warnings in stdout/stderr\n")
	prompt.WriteString("4. The sensitivity of the tool operation\n")
	prompt.WriteString("5. File paths and their potential impact\n\n")

	prompt.WriteString("Respond with either APPROVE or BLOCK, followed by a brief explanation.\n")
	prompt.WriteString("Format: [DECISION] Explanation\n")
	prompt.WriteString("Example: BLOCK The security scan detected potential issues with the file path.\n")

	return prompt.String()
}

// parseDecision parses the LLM response into a HookDecision
func (s *Service) parseDecision(response string, hookResults []hooks.HookExecutionResult) *hooks.HookDecision {
	response = strings.TrimSpace(response)

	// Default decision if parsing fails
	decision := &hooks.HookDecision{}

	// Check if any hooks failed with non-zero exit code
	hasFailures := false
	for _, result := range hookResults {
		if result.ExitCode != 0 || result.Error != nil {
			hasFailures = true
			break
		}
	}

	// Parse the response
	upperResponse := strings.ToUpper(response)

	if strings.HasPrefix(upperResponse, "APPROVE") || strings.HasPrefix(upperResponse, "[APPROVE]") {
		decision.Decision = "approve"
		// Extract reason after APPROVE
		if idx := strings.Index(upperResponse, "APPROVE"); idx != -1 {
			reason := strings.TrimSpace(response[idx+7:])
			reason = strings.TrimPrefix(reason, "]")
			reason = strings.TrimSpace(reason)
			if reason != "" {
				decision.Reason = reason
			}
		}
	} else if strings.HasPrefix(upperResponse, "BLOCK") || strings.HasPrefix(upperResponse, "[BLOCK]") {
		decision.Decision = "block"
		// Extract reason after BLOCK
		if idx := strings.Index(upperResponse, "BLOCK"); idx != -1 {
			reason := strings.TrimSpace(response[idx+5:])
			reason = strings.TrimPrefix(reason, "]")
			reason = strings.TrimSpace(reason)
			if reason != "" {
				decision.Reason = reason
			} else {
				decision.Reason = "Tool use blocked based on hook analysis"
			}
		}
	} else {
		// If we can't parse the decision clearly, and there were hook failures,
		// err on the side of caution
		if hasFailures {
			decision.Decision = "block"
			decision.Reason = "Hook scripts reported failures or warnings"
		}
		// Otherwise, let the default permission flow continue (undefined decision)
	}

	return decision
}
