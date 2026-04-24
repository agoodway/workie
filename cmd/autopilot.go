package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agoodway/workie/autopilot"
	"github.com/agoodway/workie/config"
	"github.com/spf13/cobra"
)

var (
	startPhase   int
	backend      string
	model        string
	notify       bool
	autoApprove  bool
	dryRun       bool
	noCommit     bool
	showOutput   bool
	phaseTimeout string

	opencodeServerURL string
)

var autopilotCmd = &cobra.Command{
	Use:   "autopilot <change> [change...]",
	Short: "Run the full OpenSpec aviation pipeline for one or more changes",
	Long: `Autopilot orchestrates the full OpenSpec aviation pipeline by spawning
AI backend sessions for each phase:

  1. prepare-for-takeoff  — Create td issues from tasks.md
  2. takeoff              — Create branch, implement all issues
  3. pre-landing-check    — Multi-specialist code review
  4. prepare-for-landing  — Fix review findings
  5. land-the-plane       — Verify, approve, commit, open PR

After each phase (except land-the-plane), an intermediate git commit is
made via a separate AI session unless --no-commit is passed.

Each change must have a corresponding directory at openspec/changes/<name>/
with a tasks.md file.`,
	Example: `  # Run full pipeline for a single change
  workie autopilot add-user-auth

  # Run multiple changes sequentially
  workie autopilot add-user-auth improve-dashboard

  # Resume from phase 3 (e.g., after a failure)
  workie autopilot add-user-auth --start-phase 3

  # Preview what would execute without running
  workie autopilot add-user-auth --dry-run

  # Run with full output streaming
  workie autopilot add-user-auth --show-output

  # Skip intermediate commits
  workie autopilot add-user-auth --no-commit

  # Run with auto-approve (no permission prompts)
  workie autopilot add-user-auth --auto-approve

  # Set a shorter phase timeout (default: 45m)
  workie autopilot add-user-auth --phase-timeout 30m`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, err := time.ParseDuration(phaseTimeout)
		if err != nil {
			return fmt.Errorf("invalid --phase-timeout %q: %w", phaseTimeout, err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to resolve working directory: %w", err)
		}

		cfg, err := config.LoadConfig(cwd, configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		rawBackend := backend
		if !cmd.Flags().Changed("backend") {
			if cfg.Autopilot != nil && strings.TrimSpace(cfg.Autopilot.Backend) != "" {
				rawBackend = cfg.Autopilot.Backend
			} else {
				rawBackend = string(autopilot.BackendClaude)
			}
		}

		resolvedBackend, err := autopilot.ParseBackend(rawBackend)
		if err != nil {
			return fmt.Errorf("invalid backend: %w", err)
		}

		resolvedModel := model
		if !cmd.Flags().Changed("model") {
			resolvedModel = ""
			if cfg.Autopilot != nil {
				if configuredModel, ok := cfg.Autopilot.Models[string(resolvedBackend)]; ok {
					resolvedModel = strings.TrimSpace(configuredModel)
				}
			}
			if resolvedModel == "" {
				resolvedModel = autopilot.DefaultModelForBackend(resolvedBackend)
			}
		}

		resolvedOpencodeServerURL := ""
		resolvedOpencodeServerUsername := ""
		resolvedOpencodeServerPassword := ""
		if resolvedBackend == autopilot.BackendOpencode {
			resolvedOpencodeServerURL = strings.TrimSpace(opencodeServerURL)
			if !cmd.Flags().Changed("opencode-server-url") && cfg.Autopilot != nil && cfg.Autopilot.Opencode != nil {
				resolvedOpencodeServerURL = strings.TrimSpace(cfg.Autopilot.Opencode.ServerURL)
			}
			if resolvedOpencodeServerURL == "" {
				resolvedOpencodeServerURL = autopilot.DefaultOpencodeServerURL
			}

			usernameEnv := autopilot.DefaultOpencodeServerUsernameEnv
			passwordEnv := autopilot.DefaultOpencodeServerPasswordEnv
			if cfg.Autopilot != nil && cfg.Autopilot.Opencode != nil {
				if configuredUsernameEnv := strings.TrimSpace(cfg.Autopilot.Opencode.UsernameEnv); configuredUsernameEnv != "" {
					usernameEnv = configuredUsernameEnv
				}
				if configuredPasswordEnv := strings.TrimSpace(cfg.Autopilot.Opencode.PasswordEnv); configuredPasswordEnv != "" {
					passwordEnv = configuredPasswordEnv
				}
			}

			resolvedOpencodeServerUsername = strings.TrimSpace(os.Getenv(usernameEnv))
			resolvedOpencodeServerPassword = strings.TrimSpace(os.Getenv(passwordEnv))
			if resolvedOpencodeServerPassword != "" && resolvedOpencodeServerUsername == "" {
				resolvedOpencodeServerUsername = autopilot.DefaultOpencodeServerUsername
			}
		}

		opts := autopilot.PipelineOptions{
			Changes:                args,
			StartPhase:             autopilot.Phase(startPhase),
			Backend:                resolvedBackend,
			Model:                  resolvedModel,
			Notify:                 notify,
			AutoApprove:            autoApprove,
			DryRun:                 dryRun,
			NoCommit:               noCommit,
			ShowOutput:             showOutput,
			PhaseTimeout:           timeout,
			OpencodeServerURL:      resolvedOpencodeServerURL,
			OpencodeServerUsername: resolvedOpencodeServerUsername,
			OpencodeServerPassword: resolvedOpencodeServerPassword,
		}
		return autopilot.NewPipeline(opts).Run()
	},
}

func init() {
	rootCmd.AddCommand(autopilotCmd)

	autopilotCmd.Flags().IntVar(&startPhase, "start-phase", 1, "Resume from a specific phase (1-5)")
	autopilotCmd.Flags().StringVar(&backend, "backend", "claude", fmt.Sprintf("Backend to run autopilot with (%s)", autopilot.SupportedBackendsList))
	autopilotCmd.Flags().StringVar(&model, "model", "", "Model to use for the selected backend (defaults to the backend's default model)")
	autopilotCmd.Flags().BoolVar(&notify, "notify", true, "Send OS notifications during autopilot execution")
	autopilotCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip permission prompts for the selected backend")
	autopilotCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would execute without running")
	autopilotCmd.Flags().BoolVar(&noCommit, "no-commit", false, "Skip intermediate git commits between phases")
	autopilotCmd.Flags().BoolVar(&showOutput, "show-output", false, "Stream full backend output to terminal (default: log only)")
	autopilotCmd.Flags().StringVar(&phaseTimeout, "phase-timeout", "45m", "Maximum duration per phase before killing the process (e.g., 30m, 1h)")
	autopilotCmd.Flags().StringVar(&opencodeServerURL, "opencode-server-url", "", "URL for an existing OpenCode server (used when --backend opencode)")
}
