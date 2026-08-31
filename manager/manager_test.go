package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agoodway/workie/internal/teststdio"
)

func TestDetectGitRepositoryAt(t *testing.T) {
	repo := initTestRepo(t)
	wm := NewWithOptions(Options{Quiet: true})

	if err := wm.DetectGitRepositoryAt(filepath.Join(repo, "subdir")); err != nil {
		t.Fatalf("DetectGitRepositoryAt failed: %v", err)
	}
	expectedRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if wm.RepoPath != expectedRepo {
		t.Fatalf("expected repo path %q, got %q", expectedRepo, wm.RepoPath)
	}
	if wm.WorktreesDir != filepath.Join(filepath.Dir(expectedRepo), filepath.Base(expectedRepo)+"-worktrees") {
		t.Fatalf("unexpected worktrees dir %q", wm.WorktreesDir)
	}
}

func TestCreateOrReuseWorktree(t *testing.T) {
	repo := initTestRepo(t)
	wm := NewWithOptions(Options{Quiet: true})
	if err := wm.DetectGitRepositoryAt(repo); err != nil {
		t.Fatalf("DetectGitRepositoryAt failed: %v", err)
	}
	if err := wm.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatalf("CreateWorktreesDirectory failed: %v", err)
	}

	created, err := wm.CreateOrReuseWorktree("feature/test-change", "")
	if err != nil {
		t.Fatalf("CreateOrReuseWorktree create failed: %v", err)
	}
	if !created.Created || created.Reused {
		t.Fatalf("expected created result, got %#v", created)
	}
	if _, err := os.Stat(created.Path); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}

	reused, err := wm.CreateOrReuseWorktree("feature/test-change", created.Path)
	if err != nil {
		t.Fatalf("CreateOrReuseWorktree reuse failed: %v", err)
	}
	if !reused.Reused || reused.Created {
		t.Fatalf("expected reused result, got %#v", reused)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}

func writeWorkieConfig(t *testing.T, repo, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, ".workie.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	return teststdio.CaptureStdout(fn)
}

func preparedManager(t *testing.T, repo string, quiet bool) *WorktreeManager {
	t.Helper()
	wm := NewWithOptions(Options{Quiet: quiet})
	if err := wm.DetectGitRepositoryAt(repo); err != nil {
		t.Fatalf("DetectGitRepositoryAt failed: %v", err)
	}
	if err := wm.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	return wm
}

func TestSetupExistingWorktreeCopiesAndOverwrites(t *testing.T) {
	repo := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("from-source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkieConfig(t, repo, "files_to_copy:\n  - .env\n")

	dest := filepath.Join(t.TempDir(), "existing-wt")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".env"), []byte("old-dest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wm := preparedManager(t, repo, true)
	if err := wm.SetupExistingWorktree("feature/setup", dest); err != nil {
		t.Fatalf("SetupExistingWorktree failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from-source\n" {
		t.Fatalf("expected overwritten .env %q, got %q", "from-source\n", string(got))
	}
}

func TestSetupExistingWorktreeCopyWarningsDoNotAbort(t *testing.T) {
	repo := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "keep-me.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkieConfig(t, repo, `files_to_copy:
  - missing-file.txt
  - keep-me.txt
hooks:
  post_create:
    - "echo hooked >> hook.log"
`)

	dest := filepath.Join(t.TempDir(), "existing-wt")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	wm := preparedManager(t, repo, true)
	out := captureOutput(t, func() {
		if err := wm.SetupExistingWorktree("feature/setup", dest); err != nil {
			t.Errorf("SetupExistingWorktree failed: %v", err)
		}
	})
	if !strings.Contains(out, "Warning") {
		t.Fatalf("expected copy warning, got %q", out)
	}

	got, err := os.ReadFile(filepath.Join(dest, "keep-me.txt"))
	if err != nil {
		t.Fatalf("expected keep-me.txt to be copied: %v", err)
	}
	if string(got) != "keep\n" {
		t.Fatalf("unexpected keep-me.txt content %q", string(got))
	}
	if _, err := os.Stat(filepath.Join(dest, "hook.log")); err != nil {
		t.Fatalf("expected post_create hooks to run after copy warning: %v", err)
	}
}

func TestSetupExistingWorktreeHookOrderingAndWarnAndContinue(t *testing.T) {
	repo := initTestRepo(t)
	writeWorkieConfig(t, repo, `hooks:
  post_create:
    - "echo first >> hook.log"
    - "false"
    - "echo third >> hook.log"
`)
	dest := filepath.Join(t.TempDir(), "existing-wt")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	wm := preparedManager(t, repo, false)
	out := captureOutput(t, func() {
		if err := wm.SetupExistingWorktree("feature/setup", dest); err != nil {
			t.Errorf("SetupExistingWorktree failed: %v", err)
		}
	})
	if strings.Contains(out, "created") {
		t.Fatalf("hook warning must not say the worktree was created, got %q", out)
	}
	if !strings.Contains(out, "mixed results") && !strings.Contains(out, "Some post_create hooks failed") {
		t.Fatalf("expected a hook-failure warning, got %q", out)
	}

	got, err := os.ReadFile(filepath.Join(dest, "hook.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\nthird\n" {
		t.Fatalf("expected hooks to run in order around a failure, got %q", string(got))
	}
}

func TestSetupExistingWorktreeAllHooksFailedStillSucceeds(t *testing.T) {
	repo := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("copied\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkieConfig(t, repo, `files_to_copy:
  - .env
hooks:
  post_create:
    - "false"
    - "false"
`)
	dest := filepath.Join(t.TempDir(), "existing-wt")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	wm := preparedManager(t, repo, true)
	out := captureOutput(t, func() {
		if err := wm.SetupExistingWorktree("feature/setup", dest); err != nil {
			t.Errorf("SetupExistingWorktree returned error on all-hooks-failed: %v", err)
		}
	})
	if !strings.Contains(out, "⚠️  Warning: Some post_create hooks failed") {
		t.Fatalf("expected hook-failure warning, got %q", out)
	}
	if strings.Contains(out, "created") {
		t.Fatalf("hook warning must not say the worktree was created, got %q", out)
	}

	got, err := os.ReadFile(filepath.Join(dest, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "copied\n" {
		t.Fatalf("copied files should remain after hook failures, got %q", string(got))
	}
}

func TestSetupExistingWorktreeDoesNotAddGitWorktree(t *testing.T) {
	repo := initTestRepo(t)
	dest := filepath.Join(t.TempDir(), "not-a-worktree")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	wm := preparedManager(t, repo, true)
	beforeList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	beforeBranches := gitOutput(t, repo, "branch", "--list")

	if err := wm.SetupExistingWorktree("feature/setup", dest); err != nil {
		t.Fatalf("SetupExistingWorktree failed: %v", err)
	}

	afterList := gitOutput(t, repo, "worktree", "list", "--porcelain")
	afterBranches := gitOutput(t, repo, "branch", "--list")
	if beforeList != afterList {
		t.Fatalf("SetupExistingWorktree must not call git worktree add\nbefore:\n%s\nafter:\n%s", beforeList, afterList)
	}
	if beforeBranches != afterBranches {
		t.Fatalf("SetupExistingWorktree must not create a branch\nbefore:\n%s\nafter:\n%s", beforeBranches, afterBranches)
	}
}

func TestCreateOrReuseWorktreePrintsCreatedSuccess(t *testing.T) {
	repo := initTestRepo(t)
	wm := preparedManager(t, repo, true)
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatal(err)
	}

	var result WorktreeResult
	out := captureOutput(t, func() {
		var err error
		result, err = wm.CreateOrReuseWorktree("feature/created-stdout", "")
		if err != nil {
			t.Errorf("CreateOrReuseWorktree failed: %v", err)
		}
	})
	if !strings.Contains(out, "Successfully created worktree") {
		t.Fatalf("expected created-worktree success output, got %q", out)
	}
	if !strings.Contains(out, "feature/created-stdout") {
		t.Fatalf("expected branch name in success output, got %q", out)
	}
	if result.Path != "" && !strings.Contains(out, result.Path) {
		t.Fatalf("expected worktree path %q in success output, got %q", result.Path, out)
	}
}

func TestCreateOrReuseWorktreeReportsGitErrorForNonGitDir(t *testing.T) {
	repo := initTestRepo(t)
	wm := preparedManager(t, repo, true)
	if err := wm.CreateWorktreesDirectory(); err != nil {
		t.Fatal(err)
	}

	bogus := filepath.Join(wm.WorktreesDir, "feature/not-git")
	if err := os.MkdirAll(bogus, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := wm.CreateOrReuseWorktree("feature/not-git", bogus)
	if err == nil {
		t.Fatal("expected git error for a non-git directory")
	}
	if !strings.Contains(err.Error(), "failed to resolve worktree branch") {
		t.Fatalf("expected branch-resolve error, got %q", err.Error())
	}
}
