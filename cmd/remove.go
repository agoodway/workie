package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"workie/manager"

	"github.com/spf13/cobra"
)

var (
	forceRemove bool
	pruneBranch bool
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [branch-name]",
	Short: "Remove a worktree and optionally its branch",
	Long: `Remove a worktree when you're finished working on a branch.

This command will:
1. Execute any pre_remove hooks configured in .workie.yaml
2. Remove the worktree directory and its contents
3. Optionally delete the branch (if --prune-branch is used)
4. Clean up any Git references

Pre-remove hooks allow you to run cleanup tasks before the worktree
is removed, such as stopping services, backing up data, or stashing
changes. These hooks run in the worktree directory that will be removed.

Use this when you've finished working on a feature branch and want to
clean up your development environment.`,
	Example: `  # Remove a specific worktree (keeps the branch)
  workie remove feature/user-auth

  # Remove worktree and delete the branch
  workie remove feature/completed-feature --prune-branch

  # Force remove even with uncommitted changes
  workie remove feature/experimental --force

  # Remove worktree, delete branch, and force if needed
  workie remove hotfix/old-fix --prune-branch --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]

		// Create manager with options
		opts := manager.Options{
			ConfigFile: configFile,
			Verbose:    verbose,
			Quiet:      quiet,
		}
		wm := manager.NewWithOptions(opts)

		// Detect git repository
		if err := wm.DetectGitRepository(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Load configuration
		if err := wm.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Remove the worktree
		if err := removeWorktree(wm, branchName); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func removeWorktree(wm *manager.WorktreeManager, branchName string) error {
	// Validate branch name
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Construct expected worktree path
	worktreePath := filepath.Join(wm.WorktreesDir, branchName)

	// Check if worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s\n\nTo fix this:\n  • Check the branch name is correct\n  • Use 'workie --list' to see available worktrees\n  • Verify the worktree hasn't already been removed", worktreePath)
	}

	// Execute pre_remove hooks if configured
	if wm.Config.Hooks != nil && len(wm.Config.Hooks.PreRemove) > 0 {
		if !wm.Options.Quiet {
			fmt.Printf("🪝 Running pre_remove hooks before removal...\n")
		}
		if err := wm.ExecuteHooks(wm.Config.Hooks.PreRemove, worktreePath, "pre_remove"); err != nil {
			// Don't fail the entire operation for hook errors, just warn
			fmt.Printf("⚠️  Warning: Some pre_remove hooks failed, but worktree removal will continue\n")
			if wm.Options.Verbose {
				fmt.Printf("Hook execution details: %v\n", err)
			}
		}
	} else {
		if wm.Options.Verbose {
			fmt.Printf("🪝 No pre_remove hooks configured\n")
		}
	}

	// Check if worktree is currently active/checked out
	if err := checkWorktreeStatus(wm, worktreePath); err != nil && !forceRemove {
		return fmt.Errorf("worktree removal blocked: %w\n\nTo fix this:\n  • Commit or stash your changes\n  • Use --force to remove anyway (will lose uncommitted changes)", err)
	}

	if !wm.Options.Quiet {
		fmt.Printf("🗑️  Removing worktree: %s\n", branchName)
		if wm.Options.Verbose {
			fmt.Printf("Worktree path: %s\n", worktreePath)
		}
	}

	// Remove the worktree using git worktree remove
	if err := executeWorktreeRemove(wm, worktreePath); err != nil {
		return err
	}

	if !wm.Options.Quiet {
		fmt.Printf("✓ Worktree removed successfully\n")
	}

	// Optionally remove the branch
	if pruneBranch {
		if err := removeBranch(wm, branchName); err != nil {
			fmt.Printf("⚠️  Warning: Failed to remove branch: %v\n", err)
			fmt.Printf("You can manually remove it with: git branch -D %s\n", branchName)
		} else {
			if !wm.Options.Quiet {
				fmt.Printf("✓ Branch '%s' removed successfully\n", branchName)
			}
		}
	}

	if !wm.Options.Quiet {
		fmt.Printf("\n✅ Cleanup completed for: %s\n", branchName)
		if !pruneBranch {
			fmt.Printf("\n💡 Tip: The branch '%s' still exists. Use --prune-branch to delete it next time.\n", branchName)
		}
	}

	return nil
}

func checkWorktreeStatus(wm *manager.WorktreeManager, worktreePath string) error {
	// Check if there are uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	var stderr strings.Builder
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "not a git repository") {
			// If it's not a git repository anymore, that's fine for removal
			return nil
		}
		return fmt.Errorf("failed to check worktree status: %s", stderrStr)
	}

	// If there are uncommitted changes, warn the user
	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("worktree has uncommitted changes")
	}

	return nil
}

func executeWorktreeRemove(wm *manager.WorktreeManager, worktreePath string) error {
	args := []string{"worktree", "remove"}
	
	if forceRemove {
		args = append(args, "--force")
	}
	
	args = append(args, worktreePath)

	if wm.Options.Verbose {
		fmt.Printf("Executing: git %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = wm.RepoPath

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if _, ok := err.(*exec.ExitError); ok {
			// Parse specific git worktree remove errors
			if strings.Contains(stderrStr, "contains modified or untracked files") {
				return fmt.Errorf("git worktree remove failed: worktree contains uncommitted changes\n\nError details: %s\n\nTo fix this:\n  • Commit your changes: git add . && git commit -m 'Your message'\n  • Stash your changes: git stash\n  • Use --force to remove anyway (will lose changes)", stderrStr)
			}
			if strings.Contains(stderrStr, "is currently checked out") {
				return fmt.Errorf("git worktree remove failed: worktree is currently active\n\nError details: %s\n\nTo fix this:\n  • Switch to a different worktree or branch\n  • Use --force to remove anyway", stderrStr)
			}
			if strings.Contains(stderrStr, "not a working tree") {
				return fmt.Errorf("git worktree remove failed: path is not a valid worktree\n\nError details: %s\n\nTo fix this:\n  • Check the path is correct\n  • Use 'git worktree list' to see valid worktrees", stderrStr)
			}
			return fmt.Errorf("git worktree remove failed\n\nError details: %s\n\nTo fix this:\n  • Check git repository status\n  • Ensure the worktree path is valid\n  • Try using --force if appropriate", stderrStr)
		}
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

func removeBranch(wm *manager.WorktreeManager, branchName string) error {
	// First check if branch exists locally
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	cmd.Dir = wm.RepoPath
	
	if cmd.Run() != nil {
		// Branch doesn't exist locally, nothing to remove
		if wm.Options.Verbose {
			fmt.Printf("Branch '%s' doesn't exist locally, skipping deletion\n", branchName)
		}
		return nil
	}

	// Remove the branch
	args := []string{"branch"}
	if forceRemove {
		args = append(args, "-D") // Force delete
	} else {
		args = append(args, "-d") // Safe delete (only if merged)
	}
	args = append(args, branchName)

	if wm.Options.Verbose {
		fmt.Printf("Executing: git %s\n", strings.Join(args, " "))
	}

	cmd = exec.Command("git", args...)
	cmd.Dir = wm.RepoPath

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if _, ok := err.(*exec.ExitError); ok {
			if strings.Contains(stderrStr, "not fully merged") {
				return fmt.Errorf("branch removal failed: branch '%s' is not fully merged\n\nError details: %s\n\nTo fix this:\n  • Merge the branch first: git checkout main && git merge %s\n  • Use --force to delete anyway (will lose unmerged changes)\n  • Or remove the worktree without --prune-branch", branchName, stderrStr, branchName)
			}
			return fmt.Errorf("branch removal failed\n\nError details: %s\n\nTo fix this:\n  • Check if branch is merged\n  • Use --force to force deletion\n  • Verify branch name is correct", stderrStr)
		}
		return fmt.Errorf("failed to remove branch: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Add flags specific to remove command
	removeCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "Force removal even with uncommitted changes")
	removeCmd.Flags().BoolVarP(&pruneBranch, "prune-branch", "p", false, "Also delete the branch after removing worktree")
}
