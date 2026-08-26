package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agoodway/workie/manager"
)

func TestResolveCodexPathsFlagPrecedenceOverEnv(t *testing.T) {
	repo := initCmdTestRepo(t)
	wtFlag := filepath.Join(filepath.Dir(repo), "wt-flag")
	wtEnv := filepath.Join(filepath.Dir(repo), "wt-env")
	addWorktree(t, repo, "feature/flag", wtFlag)
	addWorktree(t, repo, "feature/env", wtEnv)

	t.Setenv("CODEX_SOURCE_TREE_PATH", filepath.Join(t.TempDir(), "missing-source"))
	t.Setenv("CODEX_WORKTREE_PATH", wtEnv)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	target, err := resolveCodexPaths(wm, repo, wtFlag, "feature/flag")
	if err != nil {
		t.Fatalf("resolveCodexPaths failed: %v", err)
	}
	if target.WorktreePath != absPath(t, wtFlag) {
		t.Fatalf("expected flag worktree %q, got %q", absPath(t, wtFlag), target.WorktreePath)
	}
	if target.BranchName != "feature/flag" {
		t.Fatalf("expected branch feature/flag, got %q", target.BranchName)
	}
}

func TestResolveCodexPathsEnvFallback(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt-env")
	addWorktree(t, repo, "feature/env", wt)

	t.Setenv("CODEX_SOURCE_TREE_PATH", repo)
	t.Setenv("CODEX_WORKTREE_PATH", wt)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	target, err := resolveCodexPaths(wm, "", "", "")
	if err != nil {
		t.Fatalf("resolveCodexPaths failed: %v", err)
	}
	if target.WorktreePath != absPath(t, wt) {
		t.Fatalf("expected env worktree %q, got %q", absPath(t, wt), target.WorktreePath)
	}
	if target.BranchName != "feature/env" {
		t.Fatalf("expected derived branch feature/env, got %q", target.BranchName)
	}
}

func TestResolveCodexPathsMissingPathsError(t *testing.T) {
	t.Setenv("CODEX_SOURCE_TREE_PATH", "")
	t.Setenv("CODEX_WORKTREE_PATH", "")

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, "", "", "")
	if err == nil {
		t.Fatal("expected error when paths are missing")
	}
	msg := err.Error()
	for _, needle := range []string{"--source-tree-path", "--worktree-path", "CODEX_SOURCE_TREE_PATH", "CODEX_WORKTREE_PATH"} {
		if !strings.Contains(msg, needle) {
			t.Fatalf("missing-path error should mention %q, got %q", needle, msg)
		}
	}
}

func TestResolveCodexPathsSymlinkCanonicalization(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt-link")
	addWorktree(t, repo, "feature/link", wt)

	linkDir := t.TempDir()
	repoLink := filepath.Join(linkDir, "repo-link")
	wtLink := filepath.Join(linkDir, "wt-link")
	if err := os.Symlink(repo, repoLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wt, wtLink); err != nil {
		t.Fatal(err)
	}

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	target, err := resolveCodexPaths(wm, repoLink, wtLink, "feature/link")
	if err != nil {
		t.Fatalf("resolveCodexPaths failed with symlinked paths: %v", err)
	}
	if target.SourceTreePath != absPath(t, repo) {
		t.Fatalf("expected canonical repo %q, got %q", absPath(t, repo), target.SourceTreePath)
	}
	if target.WorktreePath != absPath(t, wt) {
		t.Fatalf("expected canonical worktree %q, got %q", absPath(t, wt), target.WorktreePath)
	}
}

func TestResolveCodexPathsRejectsRepoRoot(t *testing.T) {
	repo := initCmdTestRepo(t)
	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, repo, "main")
	if err == nil {
		t.Fatal("expected error when worktree path equals repo root")
	}
	if !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("expected repo-root error, got %q", err.Error())
	}
}

func TestResolveCodexPathsSourceTreeMismatch(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/mismatch", wt)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, filepath.Join(repo, "subdir"), wt, "feature/mismatch")
	if err == nil {
		t.Fatal("expected source-tree mismatch error")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected mismatch error, got %q", err.Error())
	}
}

func TestResolveCodexPathsUnregisteredWorktree(t *testing.T) {
	repo := initCmdTestRepo(t)
	other := initCmdTestRepo(t)
	otherWT := filepath.Join(filepath.Dir(other), "other-wt")
	addWorktree(t, other, "feature/other", otherWT)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, otherWT, "feature/other")
	if err == nil {
		t.Fatal("expected unregistered worktree error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not a registered worktree") {
		t.Fatalf("expected unregistered worktree error, got %q", err.Error())
	}
}

func TestResolveCodexPathsWorktreeMissing(t *testing.T) {
	repo := initCmdTestRepo(t)
	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, filepath.Join(filepath.Dir(repo), "no-such-wt"), "feature/x")
	if err == nil {
		t.Fatal("expected missing worktree error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected does-not-exist error, got %q", err.Error())
	}
}

func TestResolveCodexPathsWorktreeNotDirectory(t *testing.T) {
	repo := initCmdTestRepo(t)
	filePath := filepath.Join(filepath.Dir(repo), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("nope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, filePath, "feature/x")
	if err == nil {
		t.Fatal("expected error when worktree path is a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %q", err.Error())
	}
}

func TestResolveCodexPathsBranchMismatch(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/actual", wt)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, wt, "feature/other")
	if err == nil {
		t.Fatal("expected branch mismatch error")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected branch mismatch error, got %q", err.Error())
	}
}

func TestResolveCodexPathsDetachedHEAD(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/detach", wt)
	runGit(t, wt, "checkout", "--detach")

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, repo, wt, "")
	if err == nil {
		t.Fatal("expected detached HEAD error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "detached HEAD") {
		t.Fatalf("expected detached HEAD error, got %q", msg)
	}
	if !strings.Contains(msg, "branch must be checked out") {
		t.Fatalf("expected error to say a branch must be checked out, got %q", msg)
	}

	_, err = resolveCodexPaths(wm, repo, wt, "HEAD")
	if err == nil {
		t.Fatal("explicit HEAD must not match a detached worktree")
	}
}

func TestResolveCodexPathsDerivesBranchFromHEAD(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/derived", wt)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	target, err := resolveCodexPaths(wm, repo, wt, "")
	if err != nil {
		t.Fatalf("resolveCodexPaths failed: %v", err)
	}
	if target.BranchName != "feature/derived" {
		t.Fatalf("expected derived branch feature/derived, got %q", target.BranchName)
	}
}

func TestResolveCodexPathsDoesNotUseCwd(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/cwd", wt)

	outside := t.TempDir()
	t.Chdir(outside)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	target, err := resolveCodexPaths(wm, repo, wt, "feature/cwd")
	if err != nil {
		t.Fatalf("resolveCodexPaths should detect from source-tree-path, not cwd: %v", err)
	}
	if target.SourceTreePath != absPath(t, repo) {
		t.Fatalf("expected repo %q, got %q", absPath(t, repo), target.SourceTreePath)
	}
}

func TestResolveCodexPathsIgnoresCwdConfig(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/cwd-config", wt)

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, ".workie.yaml"), []byte("hooks:\n  post_create:\n    - \"echo from-cwd\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(outside)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	if _, err := resolveCodexPaths(wm, repo, wt, "feature/cwd-config"); err != nil {
		t.Fatalf("resolveCodexPaths failed: %v", err)
	}
	if wm.Config == nil || wm.Config.Hooks != nil {
		t.Fatalf("expected empty repo config, not cwd config, got %#v", wm.Config)
	}
}

func TestResolveCodexPathsRejectsPrimaryWorkingTreeFromLinkedSource(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/linked", wt)

	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	_, err := resolveCodexPaths(wm, wt, repo, "main")
	if err == nil {
		t.Fatal("expected error when worktree path is the primary working tree")
	}
	if !strings.Contains(err.Error(), "primary working tree") && !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("expected primary/root rejection, got %q", err.Error())
	}
}
