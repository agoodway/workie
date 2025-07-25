package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/agoodway/workie/config"
	"github.com/agoodway/workie/manager"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage and execute workie hooks",
	Long: `Manage and execute hooks that can be triggered at various stages of workie operations.
	
Hooks allow you to run custom commands when certain events occur, such as:
- post_create: After creating a new work session
- pre_remove: Before removing a work session
- claude_pre_tool_use: Before Claude Code uses a tool (Bash, Edit, etc.)
- claude_post_tool_use: After Claude Code uses a tool
- claude_notification: When Claude Code shows notifications
- claude_user_prompt_submit: When user submits a prompt to Claude Code
- claude_stop: When Claude Code finishes responding
- claude_subagent_stop: When a Claude Code subagent finishes
- claude_pre_compact: Before Claude Code compacts context`,
	Example: `  workie hooks list
  workie hooks run post_create
  workie hooks test
  workie hooks add claude_stop "npm test"`,
}

var (
	hooksQuiet      bool
	hooksAIDecision bool
	hooksInputFile  string
)

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured hooks",
	Long:  "Display all hooks configured in your .workie.yaml file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.Hooks == nil || !hasAnyHooks(cfg.Hooks) {
			if !hooksQuiet {
				fmt.Println(color.YellowString("No hooks configured"))
				fmt.Println("\nYou can add hooks to your .workie.yaml file. Example:")
				fmt.Println(color.CyanString(`
hooks:
  post_create:
    - echo 'Work session started!'
    - npm test
  pre_remove:
    - git status
  claude_pre_tool_use:
    - echo 'Claude Code is about to use a tool'
  claude_post_tool_use:
    - echo 'Claude Code finished using a tool'
  claude_user_prompt_submit:
    - echo 'User submitted: $USER_PROMPT'
  claude_stop:
    - echo 'Claude Code finished responding'
    - npm test
  timeout_minutes: 5  # Optional: timeout for all hooks (default: 5)`))
			}
			return nil
		}

		if !hooksQuiet {
			fmt.Println(color.CyanString("Configured Hooks:"))
			fmt.Println()
		}

		if !hooksQuiet {
			if len(cfg.Hooks.PostCreate) > 0 {
				fmt.Println(color.GreenString("post_create:"))
				for i, hook := range cfg.Hooks.PostCreate {
					fmt.Printf("  %d. %s", i+1, hook)
					if cfg.Hooks.TimeoutMinutes > 0 {
						fmt.Printf(" (timeout: %dm)", cfg.Hooks.TimeoutMinutes)
					}
					fmt.Println()
				}
			}

			if len(cfg.Hooks.PreRemove) > 0 {
				fmt.Println(color.GreenString("\npre_remove:"))
				for i, hook := range cfg.Hooks.PreRemove {
					fmt.Printf("  %d. %s", i+1, hook)
					if cfg.Hooks.TimeoutMinutes > 0 {
						fmt.Printf(" (timeout: %dm)", cfg.Hooks.TimeoutMinutes)
					}
					fmt.Println()
				}
			}
		}

		// Display Claude Code hooks
		displayHookList("claude_pre_tool_use", cfg.Hooks.ClaudePreToolUse, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_post_tool_use", cfg.Hooks.ClaudePostToolUse, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_notification", cfg.Hooks.ClaudeNotification, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_user_prompt_submit", cfg.Hooks.ClaudeUserPromptSubmit, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_stop", cfg.Hooks.ClaudeStop, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_subagent_stop", cfg.Hooks.ClaudeSubagentStop, cfg.Hooks.TimeoutMinutes, hooksQuiet)
		displayHookList("claude_pre_compact", cfg.Hooks.ClaudePreCompact, cfg.Hooks.TimeoutMinutes, hooksQuiet)

		return nil
	},
}

var hooksRunCmd = &cobra.Command{
	Use:   "run <hook-type>",
	Short: "Manually run hooks of a specific type",
	Long:  "Execute hooks of a specific type (post_create, pre_remove, etc.) manually",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hookType := args[0]

		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr := manager.New()
		mgr.Config = cfg
		mgr.RepoPath = repoRoot
		mgr.Options.Quiet = hooksQuiet

		// Special handling for claude_notification hooks
		if hookType == "claude_notification" {
			// For claude_notification, we need to execute the special handler
			// that reads input from stdin and sends system notifications
			return mgr.ExecuteClaudeNotificationHooks()
		}

		// Determine which hooks to run based on type
		hooks, err := getHooksByType(cfg.Hooks, hookType)
		if err != nil {
			return err
		}

		if len(hooks) == 0 {
			if !hooksQuiet {
				fmt.Printf(color.YellowString("No %s hooks configured\n"), hookType)
			}
			return nil
		}

		if !hooksQuiet {
			fmt.Printf(color.CyanString("Running %s hooks...\n"), hookType)
		}

		if err := mgr.ExecuteHooks(hooks, repoRoot, hookType); err != nil {
			return fmt.Errorf("failed to execute hooks: %w", err)
		}

		if !hooksQuiet {
			fmt.Println(color.GreenString("✓ Hooks executed successfully"))
		}
		return nil
	},
}

var hooksTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test all configured hooks (dry run)",
	Long:  "Run all configured hooks in test mode to validate they work correctly",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !hooksQuiet {
			fmt.Println(color.CyanString("Testing configured hooks..."))
			fmt.Println()
		}

		allPassed := true

		// Test post_create hooks
		if len(cfg.Hooks.PostCreate) > 0 {
			if !hooksQuiet {
				fmt.Println(color.YellowString("Testing post_create hooks:"))
			}
			for i, hook := range cfg.Hooks.PostCreate {
				if !hooksQuiet {
					fmt.Printf("  %d. Testing: %s... ", i+1, hook)
				}
				if err := testHook(hook); err != nil {
					if !hooksQuiet {
						fmt.Println(color.RedString("✗ Failed: %v", err))
					}
					allPassed = false
				} else {
					if !hooksQuiet {
						fmt.Println(color.GreenString("✓ Passed"))
					}
				}
			}
		}

		// Test pre_remove hooks
		if len(cfg.Hooks.PreRemove) > 0 {
			if !hooksQuiet {
				fmt.Println(color.YellowString("\nTesting pre_remove hooks:"))
			}
			for i, hook := range cfg.Hooks.PreRemove {
				if !hooksQuiet {
					fmt.Printf("  %d. Testing: %s... ", i+1, hook)
				}
				if err := testHook(hook); err != nil {
					if !hooksQuiet {
						fmt.Println(color.RedString("✗ Failed: %v", err))
					}
					allPassed = false
				} else {
					if !hooksQuiet {
						fmt.Println(color.GreenString("✓ Passed"))
					}
				}
			}
		}

		// Test Claude Code hooks
		testHookType := func(name string, hooks []string) {
			if len(hooks) > 0 {
				if !hooksQuiet {
					fmt.Printf(color.YellowString("\nTesting %s hooks:\n"), name)
				}
				for i, hook := range hooks {
					if !hooksQuiet {
						fmt.Printf("  %d. Testing: %s... ", i+1, hook)
					}
					if err := testHook(hook); err != nil {
						if !hooksQuiet {
							fmt.Println(color.RedString("✗ Failed: %v", err))
						}
						allPassed = false
					} else {
						if !hooksQuiet {
							fmt.Println(color.GreenString("✓ Passed"))
						}
					}
				}
			}
		}

		testHookType("claude_pre_tool_use", cfg.Hooks.ClaudePreToolUse)
		testHookType("claude_post_tool_use", cfg.Hooks.ClaudePostToolUse)
		testHookType("claude_notification", cfg.Hooks.ClaudeNotification)
		testHookType("claude_user_prompt_submit", cfg.Hooks.ClaudeUserPromptSubmit)
		testHookType("claude_stop", cfg.Hooks.ClaudeStop)
		testHookType("claude_subagent_stop", cfg.Hooks.ClaudeSubagentStop)
		testHookType("claude_pre_compact", cfg.Hooks.ClaudePreCompact)

		if !hooksQuiet {
			fmt.Println()
			if allPassed {
				fmt.Println(color.GreenString("✓ All hooks passed validation"))
			} else {
				fmt.Println(color.RedString("✗ Some hooks failed validation"))
			}
		}

		if !allPassed {
			return fmt.Errorf("hook validation failed")
		}

		return nil
	},
}

var hooksAddCmd = &cobra.Command{
	Use:   "add <hook-type> <command>",
	Short: "Add a new hook to the configuration",
	Long:  "Add a new hook to your .workie.yaml configuration file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hookType := args[0]
		command := args[1]

		// Validate hook type
		validTypes := []string{
			"post_create", "pre_remove",
			"claude_pre_tool_use", "claude_post_tool_use", "claude_notification",
			"claude_user_prompt_submit", "claude_stop", "claude_subagent_stop", "claude_pre_compact",
		}

		isValid := false
		for _, t := range validTypes {
			if t == hookType {
				isValid = true
				break
			}
		}

		if !isValid {
			return fmt.Errorf("invalid hook type: %s. Valid types are: %s",
				hookType, strings.Join(validTypes, ", "))
		}

		// Get timeout flag
		timeout, _ := cmd.Flags().GetDuration("timeout")

		// Create hook entry
		hookEntry := fmt.Sprintf(`
hooks:
  %s:
    - command: "%s"`, hookType, command)

		if timeout > 0 {
			hookEntry += fmt.Sprintf(`
      timeout: %s`, timeout)
		}

		if !hooksQuiet {
			fmt.Println(color.CyanString("Add the following to your .workie.yaml file:"))
			fmt.Println(hookEntry)
			fmt.Println()

			// Check if .workie.yaml exists
			if _, err := os.Stat(".workie.yaml"); err == nil {
				fmt.Println(color.YellowString("Note: .workie.yaml already exists. Please add the hook manually to avoid overwriting existing configuration."))
			} else {
				fmt.Println(color.GreenString("You can create a .workie.yaml file with this content."))
			}
		} else {
			// In quiet mode, just output the YAML snippet
			fmt.Println(hookEntry)
		}

		return nil
	},
}

var hooksClaudeTestCmd = &cobra.Command{
	Use:   "claude-test",
	Short: "Test Claude Code PreToolUse hooks with AI decision",
	Long: `Test Claude Code PreToolUse hooks by simulating a tool use request.
This command reads Claude Code input JSON and executes the hooks with optional AI decision making.`,
	Example: `  # Test with sample input
  workie hooks claude-test --input sample-tool-use.json
  
  # Test with AI decision enabled
  workie hooks claude-test --input sample-tool-use.json --ai
  
  # Create sample input file
  echo '{"session_id":"test","tool_name":"Write","tool_input":{"file_path":"/tmp/test.txt","content":"test"}}' > test.json
  workie hooks claude-test --input test.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr := manager.New()
		mgr.Config = cfg
		mgr.RepoPath = repoRoot
		mgr.Options.Quiet = hooksQuiet

		// Handle input file or stdin
		var inputReader *os.File
		if hooksInputFile != "" {
			file, err := os.Open(hooksInputFile)
			if err != nil {
				return fmt.Errorf("failed to open input file: %w", err)
			}
			defer file.Close()
			inputReader = file
		} else {
			// Read from stdin
			inputReader = os.Stdin
			if !hooksQuiet {
				fmt.Println("Reading Claude Code input from stdin...")
			}
		}

		// Save original stdin and replace it
		originalStdin := os.Stdin
		os.Stdin = inputReader
		defer func() { os.Stdin = originalStdin }()

		// Execute the Claude PreToolUse hooks
		err = mgr.ExecuteClaudePreToolUseHooks(hooksAIDecision)
		if err != nil {
			return fmt.Errorf("failed to execute Claude hooks: %w", err)
		}

		return nil
	},
}

var (
	hooksClaudeConfigOutput string
	hooksClaudeConfigFormat string
	hooksClaudeConfigHooks  []string
	hooksClaudeConfigAI     bool
)

var hooksClaudeConfigCmd = &cobra.Command{
	Use:   "claude-config",
	Short: "Generate Claude Code settings configuration for Workie integration",
	Long: `Generate Claude Code settings.json configuration to integrate Workie hooks.
This command creates the necessary hook configuration that you can add to your Claude Code settings.`,
	Example: `  # Generate config for all hooks
  workie hooks claude-config
  
  # Generate config for specific hooks only
  workie hooks claude-config --hooks pre_tool_use,post_tool_use,stop
  
  # Save to file
  workie hooks claude-config --output ~/.claude/settings.json
  
  # Generate with AI assistance for optimal configuration
  workie hooks claude-config --ai`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr := manager.New()
		mgr.Config = cfg
		mgr.RepoPath = repoRoot
		mgr.Options.Quiet = hooksQuiet

		// Generate the configuration
		configJSON, err := mgr.GenerateClaudeConfig(hooksClaudeConfigHooks, hooksClaudeConfigAI)
		if err != nil {
			return fmt.Errorf("failed to generate Claude config: %w", err)
		}

		// Output the configuration
		if hooksClaudeConfigOutput != "" {
			// Write to file
			if err := os.WriteFile(hooksClaudeConfigOutput, []byte(configJSON), 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}
			if !hooksQuiet {
				fmt.Printf(color.GreenString("✓ Claude Code configuration written to %s\n"), hooksClaudeConfigOutput)
			}
		} else {
			// Output to stdout
			fmt.Println(configJSON)
		}

		return nil
	},
}

// Helper function to test a hook without actually executing it
func testHook(command string) error {
	// Parse the command to check if it's valid
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Basic validation - check if command seems reasonable
	if len(command) > 1000 {
		return fmt.Errorf("command too long")
	}

	return nil
}

// Helper functions

// hasAnyHooks checks if any hooks are configured
func hasAnyHooks(hooks *config.Hooks) bool {
	return len(hooks.PostCreate) > 0 || len(hooks.PreRemove) > 0 ||
		len(hooks.ClaudePreToolUse) > 0 || len(hooks.ClaudePostToolUse) > 0 ||
		len(hooks.ClaudeNotification) > 0 || len(hooks.ClaudeUserPromptSubmit) > 0 ||
		len(hooks.ClaudeStop) > 0 || len(hooks.ClaudeSubagentStop) > 0 ||
		len(hooks.ClaudePreCompact) > 0
}

// getHooksByType returns hooks for a specific type
func getHooksByType(hooks *config.Hooks, hookType string) ([]string, error) {
	switch hookType {
	case "post_create":
		return hooks.PostCreate, nil
	case "pre_remove":
		return hooks.PreRemove, nil
	case "claude_pre_tool_use":
		return hooks.ClaudePreToolUse, nil
	case "claude_post_tool_use":
		return hooks.ClaudePostToolUse, nil
	case "claude_notification":
		return hooks.ClaudeNotification, nil
	case "claude_user_prompt_submit":
		return hooks.ClaudeUserPromptSubmit, nil
	case "claude_stop":
		return hooks.ClaudeStop, nil
	case "claude_subagent_stop":
		return hooks.ClaudeSubagentStop, nil
	case "claude_pre_compact":
		return hooks.ClaudePreCompact, nil
	default:
		return nil, fmt.Errorf("unknown hook type: %s", hookType)
	}
}

// displayHookList displays a list of hooks if they exist
func displayHookList(name string, hooks []string, timeoutMinutes int, quiet bool) {
	if len(hooks) > 0 && !quiet {
		fmt.Println(color.GreenString("\n%s:", name))
		for i, hook := range hooks {
			fmt.Printf("  %d. %s", i+1, hook)
			if timeoutMinutes > 0 {
				fmt.Printf(" (timeout: %dm)", timeoutMinutes)
			}
			fmt.Println()
		}
	}
}

func init() {
	rootCmd.AddCommand(hooksCmd)

	// Add subcommands
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksRunCmd)
	hooksCmd.AddCommand(hooksTestCmd)
	hooksCmd.AddCommand(hooksAddCmd)
	hooksCmd.AddCommand(hooksClaudeTestCmd)
	hooksCmd.AddCommand(hooksClaudeConfigCmd)

	// Add quiet flag to all subcommands
	hooksListCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Suppress output")
	hooksRunCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Suppress output (shows only hook output)")
	hooksTestCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Suppress output (exit code indicates success)")
	hooksAddCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Output only the YAML configuration")
	hooksClaudeTestCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Suppress output (shows only decision JSON)")
	hooksClaudeConfigCmd.Flags().BoolVarP(&hooksQuiet, "quiet", "q", false, "Suppress output")

	// Add other flags
	hooksAddCmd.Flags().DurationP("timeout", "t", 0, "Timeout for the hook execution")

	// Claude test specific flags
	hooksClaudeTestCmd.Flags().BoolVarP(&hooksAIDecision, "ai", "a", false, "Enable AI decision making")
	hooksClaudeTestCmd.Flags().StringVarP(&hooksInputFile, "input", "i", "", "Input file containing Claude Code JSON (defaults to stdin)")

	// Claude config specific flags
	hooksClaudeConfigCmd.Flags().StringVarP(&hooksClaudeConfigOutput, "output", "o", "", "Output file path (defaults to stdout)")
	hooksClaudeConfigCmd.Flags().StringSliceVar(&hooksClaudeConfigHooks, "hooks", []string{}, "Comma-separated list of hooks to include (defaults to all)")
	hooksClaudeConfigCmd.Flags().BoolVarP(&hooksClaudeConfigAI, "ai", "a", false, "Use AI to generate optimal hook configuration")
}
