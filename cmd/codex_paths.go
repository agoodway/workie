package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agoodway/workie/manager"
)

const (
	codexSourceTreeEnv = "CODEX_SOURCE_TREE_PATH"
	codexWorktreeEnv   = "CODEX_WORKTREE_PATH"
)

// codexTarget is a resolved source tree, worktree, and branch for setup/cleanup.
type codexTarget struct {
	SourceTreePath string
	WorktreePath   string
	BranchName     string
	Manager        *manager.WorktreeManager
}

func resolveCodexPaths(wm *manager.WorktreeManager, sourceTreeFlag, worktreeFlag, branchArg string) (*codexTarget, error) {
	sourceTreePath := firstNonEmpty(sourceTreeFlag, os.Getenv(codexSourceTreeEnv))
	worktreePath := firstNonEmpty(worktreeFlag, os.Getenv(codexWorktreeEnv))
	if sourceTreePath == "" || worktreePath == "" {
		return nil, fmt.Errorf("source tree path and worktree path are required: set --source-tree-path and --worktree-path, or environment variables %s and %s", codexSourceTreeEnv, codexWorktreeEnv)
	}

	if err := wm.DetectGitRepositoryAt(sourceTreePath); err != nil {
		return nil, err
	}
	if err := wm.LoadConfigFromRepo(); err != nil {
		return nil, err
	}

	canonicalSource, err := canonicalizePath(sourceTreePath)
	if err != nil {
		return nil, fmt.Errorf("invalid source tree path: %w", err)
	}
	canonicalRepo, err := canonicalizePath(wm.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}
	if canonicalSource != canonicalRepo {
		return nil, fmt.Errorf("source tree path %s does not match the detected repository root %s", sourceTreePath, wm.RepoPath)
	}

	info, err := os.Stat(worktreePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("worktree path does not exist: %s", worktreePath)
		}
		return nil, fmt.Errorf("cannot access worktree path %s: %w", worktreePath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("worktree path is not a directory: %s", worktreePath)
	}

	canonicalWorktree, err := canonicalizePath(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("invalid worktree path: %w", err)
	}
	if canonicalWorktree == canonicalRepo {
		return nil, fmt.Errorf("worktree path %s is the repository root; provide a worktree directory, not the source tree", worktreePath)
	}

	primary, err := isPrimaryWorkingTree(canonicalWorktree)
	if err != nil {
		return nil, err
	}
	if primary {
		return nil, fmt.Errorf("worktree path %s is the primary working tree; provide a linked worktree directory, not the source tree", worktreePath)
	}

	sameRepo, err := sameGitRepository(canonicalRepo, canonicalWorktree)
	if err != nil {
		return nil, err
	}
	if !sameRepo {
		return nil, fmt.Errorf("worktree path %s is not a registered worktree of %s", worktreePath, wm.RepoPath)
	}

	registered, err := worktreeIsRegistered(canonicalRepo, canonicalWorktree)
	if err != nil {
		return nil, err
	}
	if !registered {
		return nil, fmt.Errorf("worktree path %s is not a registered worktree of %s", worktreePath, wm.RepoPath)
	}

	checkedOutBranch, err := checkedOutBranch(canonicalWorktree)
	if err != nil {
		return nil, err
	}
	if checkedOutBranch == "" || checkedOutBranch == "HEAD" {
		return nil, fmt.Errorf("worktree has a detached HEAD; a branch must be checked out")
	}

	branchName := strings.TrimSpace(branchArg)
	if branchName == "" {
		branchName = checkedOutBranch
	} else if branchName != checkedOutBranch {
		return nil, fmt.Errorf("branch name %q does not match the worktree's checked-out branch %q", branchName, checkedOutBranch)
	}

	return &codexTarget{
		SourceTreePath: canonicalRepo,
		WorktreePath:   canonicalWorktree,
		BranchName:     branchName,
		Manager:        wm,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func canonicalizePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func gitCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = scrubGitEnv(os.Environ())
	return cmd
}

func scrubGitEnv(env []string) []string {
	drop := map[string]struct{}{
		"GIT_DIR":                          {},
		"GIT_WORK_TREE":                    {},
		"GIT_COMMON_DIR":                   {},
		"GIT_OBJECT_DIRECTORY":             {},
		"GIT_INDEX_FILE":                   {},
		"GIT_NAMESPACE":                    {},
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
	}
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if _, skip := drop[key]; skip {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func gitRevParse(dir string, args ...string) (string, error) {
	cmd := gitCommand(dir, append([]string{"rev-parse"}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func canonicalizeGitDir(dir, gitPath string) (string, error) {
	if gitPath == "" {
		return "", fmt.Errorf("empty git path in %s", dir)
	}
	if !filepath.IsAbs(gitPath) {
		gitPath = filepath.Join(dir, gitPath)
	}
	return canonicalizePath(gitPath)
}

func isPrimaryWorkingTree(worktreePath string) (bool, error) {
	gitDir, err := gitRevParse(worktreePath, "--absolute-git-dir")
	if err != nil {
		return false, err
	}
	commonDir, err := gitRevParse(worktreePath, "--git-common-dir")
	if err != nil {
		return false, err
	}
	canonicalGitDir, err := canonicalizeGitDir(worktreePath, gitDir)
	if err != nil {
		return false, err
	}
	canonicalCommon, err := canonicalizeGitDir(worktreePath, commonDir)
	if err != nil {
		return false, err
	}
	return canonicalGitDir == canonicalCommon, nil
}

func sameGitRepository(repoPath, worktreePath string) (bool, error) {
	repoCommon, err := gitRevParse(repoPath, "--git-common-dir")
	if err != nil {
		return false, err
	}
	worktreeCommon, err := gitRevParse(worktreePath, "--git-common-dir")
	if err != nil {
		return false, err
	}
	canonicalRepo, err := canonicalizeGitDir(repoPath, repoCommon)
	if err != nil {
		return false, err
	}
	canonicalWorktree, err := canonicalizeGitDir(worktreePath, worktreeCommon)
	if err != nil {
		return false, err
	}
	return canonicalRepo == canonicalWorktree, nil
}

func worktreeIsRegistered(repoPath, canonicalWorktreePath string) (bool, error) {
	cmd := gitCommand(repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list worktrees for %s: %w", repoPath, err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		entry := strings.TrimPrefix(line, "worktree ")
		canonicalEntry, err := canonicalizePath(entry)
		if err != nil {
			continue
		}
		if canonicalEntry == canonicalWorktreePath {
			return true, nil
		}
	}
	return false, nil
}

func checkedOutBranch(worktreePath string) (string, error) {
	output, err := gitRevParse(worktreePath, "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to resolve checked-out branch in %s: %w", worktreePath, err)
	}
	return output, nil
}
