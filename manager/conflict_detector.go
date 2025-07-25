package manager

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ConflictInfo represents information about a potential rebase conflict
type ConflictInfo struct {
	Branch        string    `json:"branch"`
	WorktreePath  string    `json:"worktree_path"`
	ConflictFiles []string  `json:"conflict_files"`
	LastChecked   time.Time `json:"last_checked"`
	Error         string    `json:"error,omitempty"`
}

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// GetWorktrees retrieves all worktrees for the repository
func (wm *WorktreeManager) GetWorktrees() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = wm.RepoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(string(output), "\n")

	var current WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(current.Branch, "refs/heads/")
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// GetMainBranch determines the main/master branch name
func (wm *WorktreeManager) GetMainBranch() (string, error) {
	// Try common main branch names
	branches := []string{"main", "master"}

	for _, branch := range branches {
		cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branch))
		cmd.Dir = wm.RepoPath

		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	// If none found, try to get the default branch from origin
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = wm.RepoPath

	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		return branch, nil
	}

	return "main", nil // Default to main if nothing else works
}

// CheckRebaseConflicts checks all worktree branches for potential rebase conflicts
func (wm *WorktreeManager) CheckRebaseConflicts() ([]ConflictInfo, error) {
	// First, fetch latest changes
	if !wm.Options.Quiet {
		wm.printf("🔄 Fetching latest changes from origin...\n")
	}

	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = wm.RepoPath
	if err := cmd.Run(); err != nil {
		// Non-fatal, continue checking with local state
		if !wm.Options.Quiet {
			wm.printf("⚠️  Warning: Failed to fetch from origin: %v\n", err)
		}
	}

	// Get main branch
	mainBranch, err := wm.GetMainBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to determine main branch: %w", err)
	}

	// Get all worktrees
	worktrees, err := wm.GetWorktrees()
	if err != nil {
		return nil, err
	}

	var conflicts []ConflictInfo
	checkTime := time.Now()

	for _, wt := range worktrees {
		// Skip if no branch (detached HEAD) or if it's the main branch
		if wt.Branch == "" || wt.Branch == mainBranch {
			continue
		}

		if !wm.Options.Quiet {
			wm.printf("🔍 Checking branch '%s' for conflicts...\n", wt.Branch)
		}

		// Check for conflicts
		conflictInfo := wm.checkBranchConflicts(wt, mainBranch, checkTime)
		if conflictInfo != nil {
			conflicts = append(conflicts, *conflictInfo)
		}
	}

	return conflicts, nil
}

// checkBranchConflicts checks a specific branch for rebase conflicts
func (wm *WorktreeManager) checkBranchConflicts(wt WorktreeInfo, mainBranch string, checkTime time.Time) *ConflictInfo {
	// Use merge-tree to detect conflicts without modifying working tree
	cmd := exec.Command("git", "merge-tree", "--write-tree", "--no-messages",
		fmt.Sprintf("origin/%s", mainBranch), wt.Branch)
	cmd.Dir = wt.Path

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		// Check if the error is due to conflicts
		stderrStr := stderr.String()
		outputStr := string(output)

		if strings.Contains(stderrStr, "conflict") || strings.Contains(outputStr, "conflict") {
			// Parse conflict files from output
			conflictFiles := parseConflictFiles(outputStr + "\n" + stderrStr)

			return &ConflictInfo{
				Branch:        wt.Branch,
				WorktreePath:  wt.Path,
				ConflictFiles: conflictFiles,
				LastChecked:   checkTime,
			}
		}

		// If it's not a conflict error, record it
		return &ConflictInfo{
			Branch:       wt.Branch,
			WorktreePath: wt.Path,
			LastChecked:  checkTime,
			Error:        fmt.Sprintf("failed to check conflicts: %v", err),
		}
	}

	// No conflicts
	return nil
}

// parseConflictFiles extracts file paths from conflict output
func parseConflictFiles(output string) []string {
	files := []string{}
	seen := make(map[string]bool)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for common conflict markers
		if strings.Contains(line, "CONFLICT") {
			// Extract file path from CONFLICT messages
			// Example: "CONFLICT (content): Merge conflict in file.txt"
			parts := strings.Split(line, " in ")
			if len(parts) > 1 {
				file := strings.TrimSpace(parts[1])
				if file != "" && !seen[file] {
					files = append(files, file)
					seen[file] = true
				}
			}
		}
	}

	return files
}

// HasNewConflicts checks if the given conflicts are new compared to a previous check
func HasNewConflicts(oldConflicts, newConflicts []ConflictInfo) bool {
	oldMap := make(map[string]bool)
	for _, c := range oldConflicts {
		if len(c.ConflictFiles) > 0 {
			oldMap[c.Branch] = true
		}
	}

	for _, c := range newConflicts {
		if len(c.ConflictFiles) > 0 && !oldMap[c.Branch] {
			return true
		}
	}

	return false
}
