package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agoodway/workie/manager"
)

func TestSetupHelpHasNoIssueFlag(t *testing.T) {
	if setupCmd.Flags().Lookup("issue") != nil {
		t.Fatal("setup must not define --issue")
	}
	stdout, _, err := runCLI(t, "setup", "--help")
	if err != nil {
		t.Fatalf("setup --help failed: %v", err)
	}
	if strings.Contains(stdout, "--issue") {
		t.Fatalf("setup --help must not list --issue:\n%s", stdout)
	}
}

func TestCleanupHelpHasNoPruneBranchFlag(t *testing.T) {
	if cleanupCmd.Flags().Lookup("prune-branch") != nil {
		t.Fatal("cleanup must not define --prune-branch")
	}
	stdout, _, err := runCLI(t, "cleanup", "--help")
	if err != nil {
		t.Fatalf("cleanup --help failed: %v", err)
	}
	if strings.Contains(stdout, "--prune-branch") {
		t.Fatalf("cleanup --help must not list --prune-branch:\n%s", stdout)
	}
}

func TestSetupCopiesAndOverwritesWithoutWorktreeAdd(t *testing.T) {
	repo := initCmdTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("from-source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkieConfig(t, repo, `files_to_copy:
  - .env
hooks:
  post_create:
    - "echo hooked >> hook.log"
`)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/setup", wt)
	if err := os.WriteFile(filepath.Join(wt, ".env"), []byte("old-dest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	beforeList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	beforeBranches := gitOutput(t, repo, "branch", "--list")

	outside := t.TempDir()
	t.Chdir(outside)

	stdout, _, err := runCLI(t, "setup", "feature/setup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("setup failed: %v\n%s", err, stdout)
	}
	assertContains(t, stdout, "feature/setup")
	assertContains(t, stdout, absPath(t, wt))
	assertNotContains(t, stdout, "created")
	assertNotContains(t, stdout, "Successfully created worktree")

	got, err := os.ReadFile(filepath.Join(wt, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from-source\n" {
		t.Fatalf("expected overwritten .env, got %q", string(got))
	}
	if _, err := os.Stat(filepath.Join(wt, "hook.log")); err != nil {
		t.Fatalf("expected post_create hook to run: %v", err)
	}

	afterList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	afterBranches := gitOutput(t, repo, "branch", "--list")
	if beforeList != afterList {
		t.Fatalf("setup must not call git worktree add\nbefore:\n%s\nafter:\n%s", beforeList, afterList)
	}
	if beforeBranches != afterBranches {
		t.Fatalf("setup must not create a branch\nbefore:\n%s\nafter:\n%s", beforeBranches, afterBranches)
	}
}

func TestSetupUsesEnvAndFlagsOverrideEnv(t *testing.T) {
	repo := initCmdTestRepo(t)
	writeWorkieConfig(t, repo, `hooks:
  post_create:
    - "echo ran > SETUP_RAN"
`)
	wtEnv := filepath.Join(filepath.Dir(repo), "wt-env")
	wtFlag := filepath.Join(filepath.Dir(repo), "wt-flag")
	addWorktree(t, repo, "feature/env", wtEnv)
	addWorktree(t, repo, "feature/flag", wtFlag)

	t.Setenv("CODEX_SOURCE_TREE_PATH", repo)
	t.Setenv("CODEX_WORKTREE_PATH", wtEnv)
	stdout, _, err := runCLI(t, "setup")
	if err != nil {
		t.Fatalf("setup via env failed: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(filepath.Join(wtEnv, "SETUP_RAN")); err != nil {
		t.Fatalf("env worktree should have been set up: %v", err)
	}

	t.Setenv("CODEX_SOURCE_TREE_PATH", repo)
	t.Setenv("CODEX_WORKTREE_PATH", wtEnv)
	stdout, _, err = runCLI(t, "setup", "feature/flag", "--source-tree-path", repo, "--worktree-path", wtFlag)
	if err != nil {
		t.Fatalf("setup with flags failed: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(filepath.Join(wtFlag, "SETUP_RAN")); err != nil {
		t.Fatalf("flag worktree should have been set up: %v", err)
	}
}

func TestSetupHookWarnAndContinue(t *testing.T) {
	repo := initCmdTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("copied\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkieConfig(t, repo, `files_to_copy:
  - .env
hooks:
  post_create:
    - "echo first >> hook.log"
    - "false"
    - "echo third >> hook.log"
`)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/hooks", wt)

	stdout, _, err := runCLI(t, "setup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("setup should warn and continue when hooks fail: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "mixed results") && !strings.Contains(stdout, "Some post_create hooks failed") {
		t.Fatalf("expected a hook-failure warning, got %q", stdout)
	}
	assertNotContains(t, stdout, "created")

	got, err := os.ReadFile(filepath.Join(wt, "hook.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\nthird\n" {
		t.Fatalf("expected hook order first/third, got %q", string(got))
	}
}

func TestCleanupDoesNotRemoveWorktree(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	marker := filepath.Join(filepath.Dir(repo), "pre-remove-ran")
	writeWorkieConfig(t, repo, "hooks:\n  pre_remove:\n    - \"echo cleaned > "+marker+"\"\n")
	addWorktree(t, repo, "feature/cleanup", wt)

	beforeList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	stdout, _, err := runCLI(t, "cleanup", "feature/cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("cleanup failed: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("cleanup must not remove the worktree: %v", err)
	}
	afterList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	if beforeList != afterList {
		t.Fatalf("cleanup must not change git worktrees\nbefore:\n%s\nafter:\n%s", beforeList, afterList)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected pre_remove hook to run: %v", err)
	}
	assertNotContains(t, stdout, "removed")
	assertNotContains(t, stdout, "worktree removal")
}

func TestCleanupDirtyWithoutForceFailsNonRemovalWording(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	marker := filepath.Join(filepath.Dir(repo), "pre-remove-ran")
	writeWorkieConfig(t, repo, "hooks:\n  pre_remove:\n    - \"echo cleaned > "+marker+"\"\n")
	addWorktree(t, repo, "feature/dirty", wt)
	makeDirty(t, wt)

	stdout, stderr, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err == nil {
		t.Fatal("expected cleanup to fail on dirty worktree")
	}
	combined := stdout + stderr + err.Error()
	assertContains(t, combined, "uncommitted changes")
	assertNotContains(t, combined, "worktree removal blocked")
	assertNotContains(t, combined, "worktree removal")
	assertNotContains(t, combined, "will lose uncommitted changes")
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("dirty cleanup must not remove the worktree: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("pre_remove hooks should run before the dirty check: %v", err)
	}
}

func TestCleanupForceSkipsDirtyCheckOnly(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	marker := filepath.Join(filepath.Dir(repo), "pre-remove-ran")
	writeWorkieConfig(t, repo, "hooks:\n  pre_remove:\n    - \"echo cleaned > "+marker+"\"\n")
	addWorktree(t, repo, "feature/force", wt)
	makeDirty(t, wt)

	stdout, _, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt, "--force")
	if err != nil {
		t.Fatalf("cleanup --force should skip dirty check: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("cleanup --force must not remove the worktree: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("--force must still run pre_remove hooks: %v", err)
	}
	assertNotContains(t, stdout, "removed")
}

func TestCleanupHookWarnAndContinue(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	logPath := filepath.Join(filepath.Dir(repo), "hook.log")
	writeWorkieConfig(t, repo, "hooks:\n  pre_remove:\n    - \"echo first >> "+logPath+"\"\n    - \"false\"\n    - \"echo third >> "+logPath+"\"\n")
	addWorktree(t, repo, "feature/hooks", wt)

	stdout, _, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("cleanup should warn and continue when hooks fail: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "mixed results") && !strings.Contains(stdout, "Some pre_remove hooks failed") {
		t.Fatalf("expected a hook-failure warning, got %q", stdout)
	}
	assertNotContains(t, stdout, "worktree removal will continue")
	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\nthird\n" {
		t.Fatalf("expected hook order first/third, got %q", string(got))
	}
}

func TestCleanupUsesEnvVars(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/env", wt)

	t.Setenv("CODEX_SOURCE_TREE_PATH", repo)
	t.Setenv("CODEX_WORKTREE_PATH", wt)
	stdout, _, err := runCLI(t, "cleanup")
	if err != nil {
		t.Fatalf("cleanup via env failed: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareWorktreeCleanupDirtyAndForce(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/preflight", wt)
	makeDirty(t, wt)

	wm := preparedManager(t, repo)
	err := prepareWorktreeCleanup(wm, "feature/preflight", wt, false, "", "")
	if err == nil {
		t.Fatal("expected dirty worktree error")
	}
	var statusErr *worktreeStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected worktreeStatusError, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Fatalf("expected uncommitted changes, got %q", err.Error())
	}

	if err := prepareWorktreeCleanup(wm, "feature/preflight", wt, true, "", ""); err != nil {
		t.Fatalf("force should skip dirty check: %v", err)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("preflight must not remove the worktree: %v", err)
	}
}

func TestPrepareWorktreeCleanupHookWarningIsCallerMessage(t *testing.T) {
	resetCmdState(t)
	repo := initCmdTestRepo(t)
	writeWorkieConfig(t, repo, `hooks:
  pre_remove:
    - "false"
    - "false"
`)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/hooks", wt)
	wm := manager.NewWithOptions(manager.Options{})
	if err := wm.DetectGitRepositoryAt(repo); err != nil {
		t.Fatal(err)
	}
	if err := wm.LoadConfig(); err != nil {
		t.Fatal(err)
	}

	out, _ := captureOutput(t, func() {
		if err := prepareWorktreeCleanup(wm, "feature/hooks", wt, false,
			"running-hooks-intro",
			"caller-continue-message"); err != nil {
			t.Errorf("prepareWorktreeCleanup failed: %v", err)
		}
	})
	assertContains(t, out, "running-hooks-intro")
	assertContains(t, out, "caller-continue-message")
	assertNotContains(t, out, "worktree removal will continue")
}

func TestBeginStillPrintsCreatedWorktree(t *testing.T) {
	repo := initCmdTestRepo(t)
	t.Chdir(repo)

	stdout, _, err := runCLI(t, "begin", "feature/begin-regression")
	if err != nil {
		t.Fatalf("begin failed: %v\n%s", err, stdout)
	}
	assertContains(t, stdout, "Successfully created worktree")
	assertContains(t, stdout, "feature/begin-regression")
}

func TestFinishStillUsesRemovalWording(t *testing.T) {
	repo := initCmdTestRepo(t)
	writeWorkieConfig(t, repo, `hooks:
  pre_remove:
    - "false"
    - "false"
`)
	wm := preparedManager(t, repo)
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatal(err)
	}
	branch := "feature/finish-hooks"
	wt := filepath.Join(wm.WorktreesDir, branch)
	addWorktree(t, repo, branch, wt)

	forceFinish = false
	t.Cleanup(func() { forceFinish = false })
	out, _ := captureOutput(t, func() {
		if err := finishWorktree(wm, branch); err != nil {
			t.Errorf("finishWorktree failed: %v", err)
		}
	})
	assertContains(t, out, "worktree removal will continue")
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("finish should remove the worktree, stat err=%v", err)
	}
}

func TestFinishDirtyBlocksWithRemovalWording(t *testing.T) {
	repo := initCmdTestRepo(t)
	wm := preparedManager(t, repo)
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatal(err)
	}
	branch := "feature/finish-dirty"
	wt := filepath.Join(wm.WorktreesDir, branch)
	addWorktree(t, repo, branch, wt)
	makeDirty(t, wt)

	forceFinish = false
	err := finishWorktree(wm, branch)
	if err == nil {
		t.Fatal("expected finish to block on dirty worktree")
	}
	msg := err.Error()
	assertContains(t, msg, "worktree removal blocked")
	assertContains(t, msg, "will lose uncommitted changes")
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("blocked finish must not remove the worktree: %v", err)
	}
}

func TestFinishForceStillRemoves(t *testing.T) {
	repo := initCmdTestRepo(t)
	wm := preparedManager(t, repo)
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatal(err)
	}
	branch := "feature/finish-force"
	wt := filepath.Join(wm.WorktreesDir, branch)
	addWorktree(t, repo, branch, wt)
	makeDirty(t, wt)

	forceFinish = true
	t.Cleanup(func() { forceFinish = false })
	if err := finishWorktree(wm, branch); err != nil {
		t.Fatalf("finish --force should remove dirty worktree: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("expected worktree to be removed, stat err=%v", err)
	}
}

func TestSetupMissingPathsError(t *testing.T) {
	clearCodexEnv(t)
	_, _, err := runCLI(t, "setup")
	if err == nil {
		t.Fatal("expected setup to fail without paths")
	}
	msg := err.Error()
	assertContains(t, msg, "--source-tree-path")
	assertContains(t, msg, "CODEX_SOURCE_TREE_PATH")
}

func TestCleanupDetachedHEADRejected(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/detach", wt)
	runGit(t, wt, "checkout", "--detach")

	_, _, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err == nil {
		t.Fatal("expected cleanup to reject detached HEAD")
	}
	assertContains(t, err.Error(), "branch must be checked out")
}

func TestSetupDoesNotCreateWorktreesDirectory(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/nodir", wt)

	wmPeek := manager.NewWithOptions(manager.Options{Quiet: true})
	if err := wmPeek.DetectGitRepositoryAt(repo); err != nil {
		t.Fatal(err)
	}
	worktreesDir := wmPeek.WorktreesDir

	_, _, err := runCLI(t, "setup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if _, err := os.Stat(worktreesDir); !os.IsNotExist(err) {
		t.Fatalf("setup must not create %s", worktreesDir)
	}
}

func TestCleanupMissingPathsError(t *testing.T) {
	clearCodexEnv(t)
	_, _, err := runCLI(t, "cleanup")
	if err == nil {
		t.Fatal("expected cleanup to fail without paths")
	}
	msg := err.Error()
	assertContains(t, msg, "--source-tree-path")
	assertContains(t, msg, "CODEX_SOURCE_TREE_PATH")
}

func TestCleanupFlagsOverrideEnv(t *testing.T) {
	repo := initCmdTestRepo(t)
	parent := filepath.Dir(repo)
	cwdFile := filepath.Join(parent, "cleanup-cwd")
	writeWorkieConfig(t, repo, "hooks:\n  pre_remove:\n    - \"pwd > "+cwdFile+"\"\n")
	wtEnv := filepath.Join(parent, "wt-env")
	wtFlag := filepath.Join(parent, "wt-flag")
	addWorktree(t, repo, "feature/env", wtEnv)
	addWorktree(t, repo, "feature/flag", wtFlag)

	t.Setenv("CODEX_SOURCE_TREE_PATH", repo)
	t.Setenv("CODEX_WORKTREE_PATH", wtEnv)
	stdout, _, err := runCLI(t, "cleanup", "feature/flag", "--source-tree-path", repo, "--worktree-path", wtFlag)
	if err != nil {
		t.Fatalf("cleanup with flags failed: %v\n%s", err, stdout)
	}
	got, err := os.ReadFile(cwdFile)
	if err != nil {
		t.Fatalf("flag worktree should have been cleaned up: %v", err)
	}
	if !strings.Contains(string(got), "wt-flag") {
		t.Fatalf("expected flag worktree cwd, got %q", got)
	}
	if strings.Contains(string(got), "wt-env") {
		t.Fatalf("env worktree should not have been cleaned up when flags override, got %q", got)
	}
}

func TestCleanupDoesNotCreateWorktreesDirectory(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/nodir", wt)

	wmPeek := manager.NewWithOptions(manager.Options{Quiet: true})
	if err := wmPeek.DetectGitRepositoryAt(repo); err != nil {
		t.Fatal(err)
	}
	worktreesDir := wmPeek.WorktreesDir

	_, _, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if _, err := os.Stat(worktreesDir); !os.IsNotExist(err) {
		t.Fatalf("cleanup must not create %s", worktreesDir)
	}
}

func TestCleanupDoesNotDeleteBranches(t *testing.T) {
	repo := initCmdTestRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/keep-branch", wt)
	before := gitOutput(t, repo, "branch", "--list")
	stdout, _, err := runCLI(t, "cleanup", "--source-tree-path", repo, "--worktree-path", wt)
	if err != nil {
		t.Fatalf("cleanup failed: %v\n%s", err, stdout)
	}
	after := gitOutput(t, repo, "branch", "--list")
	if before != after {
		t.Fatalf("cleanup must not delete branches\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestSetupDetachedHEADDoesNotRunHooks(t *testing.T) {
	repo := initCmdTestRepo(t)
	marker := filepath.Join(filepath.Dir(repo), "setup-hook")
	writeWorkieConfig(t, repo, "hooks:\n  post_create:\n    - \"echo ran > "+marker+"\"\n")
	wt := filepath.Join(filepath.Dir(repo), "wt")
	addWorktree(t, repo, "feature/detach", wt)
	runGit(t, wt, "checkout", "--detach")

	_, _, err := runCLI(t, "setup", "--source-tree-path", repo, "--worktree-path", wt)
	if err == nil {
		t.Fatal("expected setup to reject detached HEAD")
	}
	assertContains(t, err.Error(), "branch must be checked out")
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("setup must not run hooks before validation")
	}
}

func TestSetupUnregisteredWorktreeDoesNotRunHooks(t *testing.T) {
	repo := initCmdTestRepo(t)
	marker := filepath.Join(filepath.Dir(repo), "setup-hook")
	writeWorkieConfig(t, repo, "hooks:\n  post_create:\n    - \"echo ran > "+marker+"\"\n")
	other := filepath.Join(filepath.Dir(repo), "not-a-worktree")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := runCLI(t, "setup", "--source-tree-path", repo, "--worktree-path", other)
	if err == nil {
		t.Fatal("expected setup to reject unregistered worktree")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("setup must not run hooks before validation")
	}
}
