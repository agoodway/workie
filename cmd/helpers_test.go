package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agoodway/workie/internal/teststdio"
	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

func initCmdTestRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "commit.gpgsign", "false")
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

func addWorktree(t *testing.T, repo, branch, path string) {
	t.Helper()
	runGit(t, repo, "worktree", "add", "-b", branch, path)
}

func preparedManager(t *testing.T, repo string) *manager.WorktreeManager {
	t.Helper()
	resetCmdState(t)
	wm := manager.NewWithOptions(manager.Options{Quiet: true})
	if err := wm.DetectGitRepositoryAt(repo); err != nil {
		t.Fatalf("DetectGitRepositoryAt failed: %v", err)
	}
	if err := wm.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	return wm
}

func applyCmdStateReset() {
	setupSourceTreePath = ""
	setupWorktreePath = ""
	cleanupSourceTreePath = ""
	cleanupWorktreePath = ""
	forceCleanup = false
	forceFinish = false
	pruneBranch = false
	configFile = ""
	verbose = false
	quiet = false
	listFlag = false
	versionFlag = false
	rootCmd.SetArgs(nil)
	resetHelpFlags(rootCmd)
}

func resetCmdState(t *testing.T) {
	t.Helper()
	applyCmdStateReset()
	t.Cleanup(applyCmdStateReset)
}

func resetHelpFlags(cmd *cobra.Command) {
	if f := cmd.Flags().Lookup("help"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := cmd.PersistentFlags().Lookup("help"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	for _, child := range cmd.Commands() {
		resetHelpFlags(child)
	}
}

func clearCodexEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CODEX_SOURCE_TREE_PATH", "")
	t.Setenv("CODEX_WORKTREE_PATH", "")
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	return teststdio.Capture(fn)
}

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	resetCmdState(t)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		rootCmd.SetArgs(args)
		_, runErr = rootCmd.ExecuteC()
	})
	return stdout, stderr, runErr
}

func absPath(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func makeDirty(t *testing.T, worktree string) {
	t.Helper()
	readme := filepath.Join(worktree, "README.md")
	existing, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readme, append(existing, []byte("dirty\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("expected %q not to contain %q", haystack, needle)
	}
}
