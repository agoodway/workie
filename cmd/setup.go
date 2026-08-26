package cmd

import (
	"fmt"

	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

var (
	setupSourceTreePath string
	setupWorktreePath   string
)

// setupCmd runs post-create lifecycle on an existing worktree without creating one.
var setupCmd = &cobra.Command{
	Use:   "setup [branch-name]",
	Short: "Set up an existing worktree (copy files and run post_create hooks)",
	Long: `Setup copies configured files and runs post_create hooks on an existing
worktree. It does not run git worktree add or create a branch.

Use this when another tool (for example Codex) has already created the worktree
and you want Workie to apply .workie.yaml file copies and post_create hooks.

Paths are taken from --source-tree-path and --worktree-path, falling back to
CODEX_SOURCE_TREE_PATH and CODEX_WORKTREE_PATH. If no branch name is given, it
is derived from the branch checked out in the worktree (detached HEAD is
rejected). If given, the branch name must match the worktree's checked-out
branch.

This is different from setup-sprite, which configures sprite.dev VMs.`,
	Example: `  # Set up an existing worktree with explicit paths
  workie setup feature/foo --source-tree-path /repo --worktree-path /path/to/worktree

  # Derive the branch name from the worktree HEAD
  workie setup --source-tree-path /repo --worktree-path /path/to/worktree

  # Use Codex environment variables
  workie setup`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		}

		opts := manager.Options{
			ConfigFile: configFile,
			Verbose:    verbose,
			Quiet:      quiet,
		}
		wm := manager.NewWithOptions(opts)

		target, err := resolveCodexPaths(wm, setupSourceTreePath, setupWorktreePath, branchName)
		if err != nil {
			return err
		}

		if err := target.Manager.SetupExistingWorktree(target.BranchName, target.WorktreePath); err != nil {
			return err
		}

		fmt.Printf("✅ Set up worktree:\n")
		fmt.Printf("   Branch: %s\n", target.BranchName)
		fmt.Printf("   Path: %s\n", target.WorktreePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVar(&setupSourceTreePath, "source-tree-path", "", "Path to the source git repository (or CODEX_SOURCE_TREE_PATH)")
	setupCmd.Flags().StringVar(&setupWorktreePath, "worktree-path", "", "Path to the existing worktree (or CODEX_WORKTREE_PATH)")
}
