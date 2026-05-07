package cmd

import (
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

If no branch name is provided, a timestamp-based name is generated.

Configuration is read from .workie.yaml (or workie.yaml) and can specify:
- Files and directories to copy to new worktrees
- Post-creation hooks for environment setup
- Pre-removal hooks for cleanup tasks`,
	Example: `  # Begin work on a new feature
  workie begin feature/user-auth

  # Begin a hotfix with custom configuration
  workie begin hotfix/security-patch --config .workie-production.yaml

  # Begin work silently for automation
  workie begin feature/ci-pipeline --quiet

  # Begin with detailed output for debugging
  workie begin feature/complex-setup --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string

		if len(args) > 0 {
			branchName = args[0]
		}

		opts := manager.Options{
			ConfigFile:       configFile,
			Verbose:          verbose,
			Quiet:            quiet,
			ShowInitMessages: true,
		}
		wm := manager.NewWithOptions(opts)

		return wm.Run(branchName)
	},
}

func init() {
	rootCmd.AddCommand(beginCmd)
}
