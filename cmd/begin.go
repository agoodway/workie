package cmd

import (
	"fmt"
	"os"

	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

// beginCmd represents the begin command
var beginCmd = &cobra.Command{
	Use:   "begin [branch-name]",
	Short: "Begin working on a new branch by creating a worktree",
	Long: `Begin creates a new Git worktree for isolated development on a branch.

This command will:
1. Create a new branch (or checkout an existing one)
2. Set up a worktree directory alongside your main repository
3. Copy essential files and configurations from .workie.yaml
4. Execute any post_create hooks to set up your environment
5. List all active worktrees to show your development environments

The worktree will be created in a directory named after your repository
with a "-worktrees" suffix, keeping your development organized.

Configuration is read from .workie.yaml (or workie.yaml) and can specify:
- Files and directories to copy to new worktrees
- Post-creation hooks for environment setup
- Pre-removal hooks for cleanup tasks

Use this to start working on a new feature, bugfix, or experiment without
affecting your main working directory.`,
	Example: `  # Begin work on a new feature
  workie begin feature/user-auth

  # Begin work on an AI integration feature
  workie begin feature/ai-integration

  # Begin a hotfix with custom configuration
  workie begin hotfix/security-patch --config .workie-production.yaml

  # Begin work silently for automation
  workie begin feature/ci-pipeline --quiet

  # Begin with detailed output for debugging
  workie begin feature/complex-setup --verbose`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]

		// Create manager with options
		opts := manager.Options{
			ConfigFile: configFile,
			Verbose:    verbose,
			Quiet:      quiet,
			ShowInitMessages: true,
		}
		wm := manager.NewWithOptions(opts)

		// Run the main workflow with provided branch name
		if err := wm.Run(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(beginCmd)
}