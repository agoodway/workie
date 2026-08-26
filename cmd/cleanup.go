package cmd

import (
	"errors"
	"fmt"

	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

var (
	forceCleanup          bool
	cleanupSourceTreePath string
	cleanupWorktreePath   string
)

// cleanupCmd runs pre-remove lifecycle on an existing worktree without removing it.
var cleanupCmd = &cobra.Command{
	Use:   "cleanup [branch-name]",
	Short: "Clean up an existing worktree (pre_remove hooks and dirty check, no removal)",
	Long: `Cleanup runs pre_remove hooks and checks for uncommitted changes on an
existing worktree. It does not run git worktree remove or delete branches.

Use this when another tool (for example Codex) owns worktree removal and you
want Workie to apply .workie.yaml pre_remove hooks and dirty-worktree checks.

Paths are taken from --source-tree-path and --worktree-path, falling back to
CODEX_SOURCE_TREE_PATH and CODEX_WORKTREE_PATH. If no branch name is given, it
is derived from the branch checked out in the worktree (detached HEAD is
rejected). If given, the branch name must match the worktree's checked-out
branch.

--force skips the dirty-worktree check only; it does not remove the worktree.`,
	Example: `  # Clean up an existing worktree with explicit paths
  workie cleanup feature/foo --source-tree-path /repo --worktree-path /path/to/worktree

  # Skip the dirty-worktree check
  workie cleanup feature/foo --source-tree-path /repo --worktree-path /path/to/worktree --force

  # Use Codex environment variables
  workie cleanup`,
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

		target, err := resolveCodexPaths(wm, cleanupSourceTreePath, cleanupWorktreePath, branchName)
		if err != nil {
			return err
		}

		if err := prepareWorktreeCleanup(target.Manager, target.BranchName, target.WorktreePath, forceCleanup,
			"🪝 Running pre_remove hooks...",
			"⚠️  Warning: Some pre_remove hooks failed"); err != nil {
			var statusErr *worktreeStatusError
			if errors.As(err, &statusErr) {
				return fmt.Errorf("cleanup blocked: %w\n\nTo fix this:\n  • Commit or stash your changes\n  • Use --force to continue anyway", err)
			}
			return err
		}

		fmt.Printf("✅ Cleaned up worktree:\n")
		fmt.Printf("   Branch: %s\n", target.BranchName)
		fmt.Printf("   Path: %s\n", target.WorktreePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().StringVar(&cleanupSourceTreePath, "source-tree-path", "", "Path to the source git repository (or CODEX_SOURCE_TREE_PATH)")
	cleanupCmd.Flags().StringVar(&cleanupWorktreePath, "worktree-path", "", "Path to the existing worktree (or CODEX_WORKTREE_PATH)")
	cleanupCmd.Flags().BoolVarP(&forceCleanup, "force", "f", false, "Skip uncommitted-changes check (does not remove the worktree)")
}
