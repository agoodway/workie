package manager

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agoodway/workie/config"
)

// Options holds configuration options for the WorktreeManager
type Options struct {
	ConfigFile string // Path to custom config file
	Verbose    bool   // Enable verbose output
	Quiet      bool   // Enable quiet mode
}

// WorktreeManager handles git worktree operations
type WorktreeManager struct {
	RepoPath     string
	RepoName     string
	WorktreesDir string
	Config       *config.Config
	Options      Options
}

// New creates a new WorktreeManager instance with default options
func New() *WorktreeManager {
	return &WorktreeManager{}
}

// NewWithOptions creates a new WorktreeManager instance with the specified options
func NewWithOptions(opts Options) *WorktreeManager {
	return &WorktreeManager{
		Options: opts,
	}
}

// DetectGitRepository detects the current git repository and sets up paths
func (wm *WorktreeManager) DetectGitRepository() error {
	// First check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git command not found: Please install git and ensure it's in your PATH")
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// More specific error message based on git output
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not a git repository") {
				return fmt.Errorf("not in a git repository: Please run this command from within a git repository")
			}
			return fmt.Errorf("git command failed: %s", stderr)
		}
		return fmt.Errorf("failed to detect git repository: %w", err)
	}

	wm.RepoPath = strings.TrimSpace(string(output))
	if wm.RepoPath == "" {
		return fmt.Errorf("could not determine git repository path")
	}

	// Verify the repository path exists and is accessible
	if info, err := os.Stat(wm.RepoPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("git repository path does not exist: %s", wm.RepoPath)
		}
		return fmt.Errorf("cannot access git repository path %s: %w", wm.RepoPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("git repository path is not a directory: %s", wm.RepoPath)
	}

	wm.RepoName = filepath.Base(wm.RepoPath)
	if wm.RepoName == "" || wm.RepoName == "." || wm.RepoName == "/" {
		return fmt.Errorf("could not determine repository name from path: %s", wm.RepoPath)
	}

	// Create worktrees directory path alongside the repository
	parentDir := filepath.Dir(wm.RepoPath)
	wm.WorktreesDir = filepath.Join(parentDir, fmt.Sprintf("%s-worktrees", wm.RepoName))

	wm.printf("✓ Detected git repository: %s\n", wm.RepoPath)
	if wm.Options.Verbose {
		wm.printf("✓ Repository name: %s\n", wm.RepoName)
		wm.printf("✓ Parent directory: %s\n", filepath.Dir(wm.RepoPath))
	}
	wm.printf("✓ Worktrees directory: %s\n", wm.WorktreesDir)

	return nil
}

// LoadConfig loads the YAML configuration file
func (wm *WorktreeManager) LoadConfig() error {
	var err error
	wm.Config, err = config.LoadConfig(wm.RepoPath, wm.Options.ConfigFile)
	if err != nil {
		// Provide more specific error messages based on the error type
		if strings.Contains(err.Error(), "custom config file not found") {
			return fmt.Errorf("configuration file error: %w\n\nTo fix this:\n  • Check that the file path is correct\n  • Use --config flag with a valid YAML file\n  • Or remove the --config flag to use default configuration", err)
		}
		if strings.Contains(err.Error(), "failed to parse YAML") {
			return fmt.Errorf("configuration file syntax error: %w\n\nTo fix this:\n  • Check YAML syntax and indentation\n  • Ensure the file uses proper YAML format\n  • Example valid config:\n    files_to_copy:\n      - .env.example\n      - config/\n      - scripts/setup.sh", err)
		}
		if strings.Contains(err.Error(), "failed to read config file") {
			return fmt.Errorf("configuration file access error: %w\n\nTo fix this:\n  • Check file permissions (should be readable)\n  • Ensure the file is not corrupted\n  • Verify the file path is accessible", err)
		}
		return fmt.Errorf("configuration error: %w", err)
	}

	// Validate configuration content
	if wm.Config == nil {
		return fmt.Errorf("configuration loading failed: received nil configuration")
	}

	// Print config loading info based on output mode
	if wm.Config.LoadedFrom != "" && !wm.Options.Quiet {
		wm.printf("✓ Loaded configuration from: %s\n", wm.Config.LoadedFrom)
		if len(wm.Config.FilesToCopy) > 0 {
			wm.printf("✓ Files to copy: %d entries\n", len(wm.Config.FilesToCopy))
		}
	} else if wm.Config.LoadedFrom == "" && !wm.Options.Quiet {
		wm.printf("✓ No configuration file found - using default settings\n")
	}

	return nil
}

// CreateWorktreesDirectory creates the worktrees directory if it doesn't exist
func (wm *WorktreeManager) CreateWorktreesDirectory() error {
	// Check if directory already exists
	if info, err := os.Stat(wm.WorktreesDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("worktrees path already exists but is not a directory: %s\n\nTo fix this:\n  • Remove the file at this path\n  • Or choose a different location for worktrees", wm.WorktreesDir)
		}
		wm.printf("✓ Using existing worktrees directory: %s\n", wm.WorktreesDir)
		return nil
	}

	// Try to create the directory
	if err := os.MkdirAll(wm.WorktreesDir, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied creating worktrees directory: %s\n\nTo fix this:\n  • Check directory permissions in parent directory\n  • Ensure you have write access to: %s\n  • Consider running with appropriate permissions", wm.WorktreesDir, filepath.Dir(wm.WorktreesDir))
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("parent directory does not exist: %s\n\nTo fix this:\n  • Ensure the parent directory exists\n  • Create the parent directory first", filepath.Dir(wm.WorktreesDir))
		}
		return fmt.Errorf("failed to create worktrees directory %s: %w\n\nTo fix this:\n  • Check available disk space\n  • Verify directory permissions\n  • Ensure the path is valid", wm.WorktreesDir, err)
	}

	wm.printf("✓ Created worktrees directory: %s\n", wm.WorktreesDir)
	return nil
}

// GenerateBranchName generates a branch name based on current timestamp
func (wm *WorktreeManager) GenerateBranchName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("feature/work-%s", timestamp)
}

// BranchExists checks if a branch already exists locally or remotely
func (wm *WorktreeManager) BranchExists(branchName string) bool {
	// Check local branches
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	cmd.Dir = wm.RepoPath
	if cmd.Run() == nil {
		return true
	}

	// Check remote branches
	cmd = exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/remotes/origin/%s", branchName))
	cmd.Dir = wm.RepoPath
	return cmd.Run() == nil
}

// copyFile copies a file from src to dst with comprehensive error handling
func (wm *WorktreeManager) copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source file does not exist: %s", src)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied reading source file: %s", src)
		}
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied creating directory: %s", destDir)
		}
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied creating destination file: %s", dst)
		}
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy content from %s to %s: %w", src, dst, err)
	}

	return nil
}

// copyDirectory recursively copies a directory from src to dst with detailed error handling
func (wm *WorktreeManager) copyDirectory(src, dst string) error {
	// Verify source directory exists
	if info, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source directory does not exist: %s", src)
		}
		return fmt.Errorf("cannot access source directory %s: %w", src, err)
	} else if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", src)
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("permission denied accessing: %s", path)
			}
			return fmt.Errorf("error accessing %s: %w", path, err)
		}

		// Calculate the relative path from src
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path for %s: %w", path, err)
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				if os.IsPermission(err) {
					return fmt.Errorf("permission denied creating directory: %s", dstPath)
				}
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}
			return nil
		}

		if err := wm.copyFile(path, dstPath); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", path, dstPath, err)
		}
		return nil
	})
}

// copyConfiguredFiles copies files/directories specified in the configuration
func (wm *WorktreeManager) copyConfiguredFiles(worktreePath string) error {
	if !wm.Config.HasFilesToCopy() {
		wm.printf("📂 No files configured to copy\n")
		return nil
	}

	wm.printf("📂 Copying configured files to worktree...\n")

	var copyErrors []string
	successCount := 0

	for _, item := range wm.Config.FilesToCopy {
		// Validate item name
		if strings.TrimSpace(item) == "" {
			fmt.Printf("⚠️  Warning: Skipping empty file/directory name in configuration\n")
			continue
		}

		srcPath := filepath.Join(wm.RepoPath, item)
		dstPath := filepath.Join(worktreePath, item)

		// Check if source exists
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				errorMsg := fmt.Sprintf("Source file/directory not found: %s → Expected at: %s", item, srcPath)
				fmt.Printf("⚠️  Warning: %s\n", errorMsg)
				copyErrors = append(copyErrors, errorMsg)
			} else {
				errorMsg := fmt.Sprintf("Cannot access source %s at %s: %v", item, srcPath, err)
				fmt.Printf("⚠️  Warning: %s\n", errorMsg)
				copyErrors = append(copyErrors, errorMsg)
			}
			continue
		}

		if srcInfo.IsDir() {
			wm.printf("   📁 Copying directory: %s\n", item)
			if wm.Options.Verbose {
				wm.printf("     From → To: %s → %s\n", srcPath, dstPath)
			}
			if err := wm.copyDirectory(srcPath, dstPath); err != nil {
				errorMsg := fmt.Sprintf("Failed to copy directory %s from %s to %s: %v", item, srcPath, dstPath, err)
				fmt.Printf("❌ Error: %s\n", errorMsg)
				copyErrors = append(copyErrors, errorMsg)
			} else {
				successCount++
				wm.printf("     ✓ Directory copied successfully\n")
			}
		} else {
			wm.printf("   📄 Copying file: %s\n", item)
			if wm.Options.Verbose {
				wm.printf("     From → To: %s → %s\n", srcPath, dstPath)
			}
			if err := wm.copyFile(srcPath, dstPath); err != nil {
				errorMsg := fmt.Sprintf("Failed to copy file %s from %s to %s: %v", item, srcPath, dstPath, err)
				fmt.Printf("❌ Error: %s\n", errorMsg)
				copyErrors = append(copyErrors, errorMsg)
			} else {
				successCount++
				wm.printf("     ✓ File copied successfully\n")
			}
		}
	}

	// Show summary
	totalItems := len(wm.Config.FilesToCopy)
	if successCount == totalItems {
		wm.printf("✓ Successfully copied all %d configured items\n", successCount)
	} else if successCount > 0 {
		wm.printf("⚠️  Copied %d out of %d configured items (%d failed)\n", successCount, totalItems, len(copyErrors))
	} else {
		wm.printf("❌ Failed to copy any configured items\n")
	}

	// If there were copy errors, provide helpful information
	if len(copyErrors) > 0 && wm.Options.Verbose {
		fmt.Printf("\nCopy error summary:\n")
		for i, err := range copyErrors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
		fmt.Printf("\nTo fix copy issues:\n")
		fmt.Printf("  • Verify files/directories exist in the source repository\n")
		fmt.Printf("  • Check file permissions\n")
		fmt.Printf("  • Update your configuration file if paths have changed\n")
	}

	return nil
}

// CreateWorktreeBranch creates a new worktree with the specified branch name
func (wm *WorktreeManager) CreateWorktreeBranch(branchName string) error {
	// Validate branch name
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for invalid characters in branch name
	if strings.ContainsAny(branchName, " \t\n\r~^:?*[\\@{}") {
		return fmt.Errorf("invalid branch name '%s': contains invalid characters\n\nBranch names cannot contain: spaces, ~, ^, :, ?, *, [, \\, @, {, }\nTry using: feature/my-branch, bugfix/issue-123, etc.", branchName)
	}

	if wm.BranchExists(branchName) {
		return fmt.Errorf("branch '%s' already exists\n\nTo fix this:\n  • Use a different branch name\n  • Or delete the existing branch if no longer needed\n  • Use: git branch -D %s (to delete locally)\n  • Use: git push origin --delete %s (to delete remotely)", branchName, branchName, branchName)
	}

	worktreePath := filepath.Join(wm.WorktreesDir, branchName)

	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s\n\nTo fix this:\n  • Choose a different branch name\n  • Remove the existing directory: rm -rf %s\n  • Or use: git worktree remove %s", worktreePath, worktreePath, worktreePath)
	}

	// Create new worktree with new branch
	wm.printf("📝 Creating worktree for branch '%s'...\n", branchName)
	if wm.Options.Verbose {
		wm.printf("Executing: git worktree add -b %s %s\n", branchName, worktreePath)
	}

	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = wm.RepoPath

	// Capture both stdout and stderr for better error reporting
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if _, ok := err.(*exec.ExitError); ok {
			// Parse specific git worktree errors
			if strings.Contains(stderrStr, "already exists") {
				return fmt.Errorf("git worktree creation failed: path already exists\n\nError details: %s\n\nTo fix this:\n  • Remove the existing directory\n  • Use a different branch name\n  • Clean up with: git worktree prune", stderrStr)
			}
			if strings.Contains(stderrStr, "is already checked out") {
				return fmt.Errorf("git worktree creation failed: branch already checked out\n\nError details: %s\n\nTo fix this:\n  • Use a different branch name\n  • Switch to a different branch in existing worktree\n  • Remove the existing worktree first", stderrStr)
			}
			if strings.Contains(stderrStr, "not a valid object name") {
				return fmt.Errorf("git worktree creation failed: invalid reference\n\nError details: %s\n\nTo fix this:\n  • Ensure you're in a valid git repository\n  • Check that HEAD points to a valid commit\n  • Try: git status to check repository state", stderrStr)
			}
			return fmt.Errorf("git worktree creation failed\n\nError details: %s\n\nTo fix this:\n  • Check git repository status: git status\n  • Ensure working directory is clean\n  • Verify branch name is valid\n  • Check available disk space", stderrStr)
		}
		return fmt.Errorf("failed to create worktree: %w\n\nCommand: git worktree add -b %s %s\nWorking directory: %s", err, branchName, worktreePath, wm.RepoPath)
	}

	wm.printf("✓ Git worktree created successfully\n")

	// Copy configured files to the new worktree
	if err := wm.copyConfiguredFiles(worktreePath); err != nil {
		return fmt.Errorf("failed to copy configured files: %w", err)
	}

	// Execute post_create hooks if configured
	if wm.HasPostCreateHooks() {
		if err := wm.ExecuteHooks(wm.Config.Hooks.PostCreate, worktreePath, "post_create"); err != nil {
			// Don't fail the entire operation for hook errors, just warn
			fmt.Printf("⚠️  Warning: Some post_create hooks failed, but worktree was created successfully\n")
			if wm.Options.Verbose {
				fmt.Printf("Hook execution details: %v\n", err)
			}
		}
	} else {
		wm.printf("🪝 No post_create hooks configured\n")
	}

	// Always show success and path info, even in quiet mode (essential info)
	fmt.Printf("✅ Successfully created worktree:\n")
	fmt.Printf("   Branch: %s\n", branchName)
	fmt.Printf("   Path: %s\n", worktreePath)

	// Show file copy summary
	if wm.Config.HasFilesToCopy() {
		totalConfiguredFiles := len(wm.Config.FilesToCopy)
		fmt.Printf("   Files copied to worktree: %d configured item(s)\n", totalConfiguredFiles)
		if wm.Options.Verbose {
			fmt.Printf("   From repository → To worktree: %s → %s\n", wm.RepoPath, worktreePath)
		}
	} else {
		fmt.Printf("   Files copied to worktree: None (no files configured)\n")
	}

	// Show next steps in non-quiet mode
	if !wm.Options.Quiet {
		fmt.Printf("\n🚀 To start working:\n")
		fmt.Printf("   cd %s\n", worktreePath)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("   • Make your changes\n")
		fmt.Printf("   • Commit your work: git add . && git commit -m 'Your message'\n")
		fmt.Printf("   • Push when ready: git push -u origin %s\n", branchName)
	}

	// For quiet mode, just output the worktree path
	if wm.Options.Quiet {
		fmt.Println(worktreePath)
	}

	return nil
}

// ListWorktrees lists all existing worktrees
func (wm *WorktreeManager) ListWorktrees() error {
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = wm.RepoPath

	var stderr strings.Builder
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		stderrStr := stderr.String()
		if _, ok := err.(*exec.ExitError); ok {
			if strings.Contains(stderrStr, "not a git repository") {
				return fmt.Errorf("cannot list worktrees: not in a git repository\n\nTo fix this:\n  • Navigate to a git repository\n  • Initialize a git repository: git init")
			}
			return fmt.Errorf("git worktree list failed\n\nError details: %s\n\nTo fix this:\n  • Ensure you're in a valid git repository\n  • Check git installation: git --version\n  • Verify repository status: git status", stderrStr)
		}
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if wm.Options.Verbose {
		wm.printf("Executing: git worktree list\n")
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		wm.printf("\n📋 No worktrees found\n")
		if !wm.Options.Quiet {
			fmt.Printf("Only the main repository is currently available.\n")
			fmt.Printf("Create a new worktree with: %s <branch-name>\n", os.Args[0])
		}
		return nil
	}

	wm.printf("\n📋 Existing worktrees:\n")
	if !wm.Options.Quiet {
		fmt.Printf("%s\n", outputStr)

		// Count worktrees for additional info
		lines := strings.Split(outputStr, "\n")
		worktreeCount := 0
		mainRepo := ""

		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				worktreeCount++
				if strings.Contains(line, "(bare)") || strings.Contains(line, "["+wm.RepoName+"]") {
					mainRepo = strings.Fields(line)[0]
				}
			}
		}

		if wm.Options.Verbose {
			fmt.Printf("\nSummary: Found %d worktree(s)\n", worktreeCount)
			if mainRepo != "" {
				fmt.Printf("Main repository: %s\n", mainRepo)
			}
		}
	}

	return nil
}

// HookExecutionResult represents the result of executing a single hook
type HookExecutionResult struct {
	Index    int
	Command  string
	Success  bool
	Duration time.Duration
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
	TimedOut bool
}

// HookSummary represents the overall execution summary
type HookSummary struct {
	HookType      string
	TotalHooks    int
	SuccessCount  int
	FailedCount   int
	SkippedCount  int
	TotalDuration time.Duration
	Results       []HookExecutionResult
	WorkingDir    string
}

// ExecuteHooks executes a slice of command strings in sequence within the specified working directory
// It provides comprehensive error handling, progress indication, and detailed feedback
func (wm *WorktreeManager) ExecuteHooks(hooks []string, workDir string, hookType string) error {
	if len(hooks) == 0 {
		wm.printf("🪝 No %s hooks configured\n", hookType)
		return nil
	}

	// Show progress indicator and initial status
	wm.printf("🪝 Executing %s hooks (%d commands)...\n", hookType, len(hooks))
	if wm.Options.Verbose {
		wm.printf("Working directory: %s\n", workDir)
		wm.printf("Hook timeout: %v\n", wm.getHookTimeout())
	}

	// Validate working directory
	if _, err := os.Stat(workDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("hook execution failed: working directory does not exist: %s", workDir)
		}
		return fmt.Errorf("hook execution failed: cannot access working directory %s: %w", workDir, err)
	}

	// Initialize execution summary
	summary := HookSummary{
		HookType:   hookType,
		TotalHooks: len(hooks),
		Results:    make([]HookExecutionResult, 0, len(hooks)),
		WorkingDir: workDir,
	}

	// Show progress indicator for longer operations
	if !wm.Options.Quiet && len(hooks) > 3 {
		fmt.Printf("\n")
		wm.showProgressIndicator("Initializing hooks...")
	}

	overallStart := time.Now()

	for i, hookCommand := range hooks {
		// Validate hook command
		hookCommand = strings.TrimSpace(hookCommand)
		if hookCommand == "" {
			wm.printf("   ⚠️  Warning: Skipping empty hook command at position %d\n", i+1)
			summary.SkippedCount++
			continue
		}

		// Show current progress
		wm.printf("\n   [%d/%d] 🔄 Running: %s\n", i+1, len(hooks), hookCommand)

		// In verbose mode, show exact command being executed
		if wm.Options.Verbose {
			wm.printf("      Directory: %s\n", workDir)
			wm.printf("      Timeout: %v\n", wm.getHookTimeout())
		}

		// Execute the hook with comprehensive error handling
		result := wm.executeHookCommand(hookCommand, workDir, i+1)
		summary.Results = append(summary.Results, result)

		// Update counters
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailedCount++
		}

		// Show result with appropriate formatting
		wm.displayHookResult(result)

		// In non-verbose mode, show a simple progress indicator
		if !wm.Options.Verbose && !wm.Options.Quiet {
			wm.updateProgress(i+1, len(hooks))
		}
	}

	summary.TotalDuration = time.Since(overallStart)

	// Display comprehensive execution summary
	wm.displayHookSummary(summary)

	// Return error only if all hooks failed, otherwise return nil to continue workflow
	if summary.FailedCount > 0 && summary.SuccessCount == 0 {
		return fmt.Errorf("all %s hooks failed to execute - see above for details", hookType)
	}

	return nil
}

// printf is a helper function that considers the verbose and quiet flags
func (wm *WorktreeManager) printf(format string, a ...interface{}) {
	if !wm.Options.Quiet {
		if wm.Options.Verbose {
			fmt.Printf("VERBOSE: "+format, a...)
		} else {
			fmt.Printf(format, a...)
		}
	}
}

// HasPostCreateHooks checks if post_create hooks are configured
func (wm *WorktreeManager) HasPostCreateHooks() bool {
	return wm.Config != nil && wm.Config.Hooks != nil && len(wm.Config.Hooks.PostCreate) > 0
}

// HasPreRemoveHooks checks if pre_remove hooks are configured
func (wm *WorktreeManager) HasPreRemoveHooks() bool {
	return wm.Config != nil && wm.Config.Hooks != nil && len(wm.Config.Hooks.PreRemove) > 0
}


// parseCommand splits command strings into executable parts
// It handles shell-style commands with pipes, redirects, etc.
func parseCommand(command string) ([]*exec.Cmd, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Check if command contains shell operators that require shell execution
	needsShell := strings.ContainsAny(command, "|&;<>()$`{}*?[]~")
	needsShell = needsShell || strings.Contains(command, ">>")
	needsShell = needsShell || strings.Contains(command, "<<")
	needsShell = needsShell || strings.Contains(command, "&&")
	needsShell = needsShell || strings.Contains(command, "||")

	var cmds []*exec.Cmd

	if needsShell {
		// Use shell for complex commands
		cmd := exec.Command("sh", "-c", command)
		cmds = append(cmds, cmd)
	} else {
		// Simple command - split by whitespace
		cmdParts := strings.Fields(command)
		if len(cmdParts) == 0 {
			return nil, fmt.Errorf("no command parts found")
		}

		var cmd *exec.Cmd
		if len(cmdParts) == 1 {
			cmd = exec.Command(cmdParts[0])
		} else {
			cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
		}
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

// getHookTimeout returns the configured timeout for hook execution
func (wm *WorktreeManager) getHookTimeout() time.Duration {
	// Use configured timeout if available
	if wm.Config != nil && wm.Config.Hooks != nil && wm.Config.Hooks.TimeoutMinutes > 0 {
		return time.Duration(wm.Config.Hooks.TimeoutMinutes) * time.Minute
	}
	// Default timeout of 5 minutes
	return 5 * time.Minute
}

// showProgressIndicator shows a spinning progress indicator
func (wm *WorktreeManager) showProgressIndicator(message string) {
	if wm.Options.Quiet {
		return
	}
	fmt.Printf("\r%s ", message)
}

// updateProgress shows progress as a percentage
func (wm *WorktreeManager) updateProgress(current, total int) {
	if wm.Options.Quiet {
		return
	}
	percent := (current * 100) / total
	fmt.Printf("\r   Progress: [")
	bars := percent / 5 // Each bar represents 5%
	for i := 0; i < 20; i++ {
		if i < bars {
			fmt.Printf("█")
		} else {
			fmt.Printf("░")
		}
	}
	fmt.Printf("] %d%% (%d/%d)", percent, current, total)
	if current == total {
		fmt.Printf("\n")
	}
}

// executeHookCommand executes a single hook command with timeout and comprehensive error handling
func (wm *WorktreeManager) executeHookCommand(command, workDir string, index int) HookExecutionResult {
	result := HookExecutionResult{
		Index:   index,
		Command: command,
		Success: false,
	}

	// Parse command using helper method that handles shell operators
	cmds, err := parseCommand(command)
	if err != nil {
		result.Error = fmt.Errorf("command parsing failed: %w", err)
		return result
	}

	// For now, we'll only execute the first command in the parsed list
	cmd := cmds[0]
	cmd.Dir = workDir

	// Capture output for verbose mode or error reporting
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set up command execution with timeout
	start := time.Now()
	timeout := wm.getHookTimeout()

	// Create a channel to signal completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for either completion or timeout
	var execErr error
	select {
	case execErr = <-done:
		// Command completed within timeout
		result.Duration = time.Since(start)
	case <-time.After(timeout):
		// Command timed out
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				// Log but don't fail on kill error
				if wm.Options.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to kill timed-out process: %v\n", err)
				}
			}
		}
		result.TimedOut = true
		result.Duration = timeout
		execErr = fmt.Errorf("command timed out after %v", timeout)
	}

	// Capture output
	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())

	// Handle command results
	if execErr != nil {
		result.Error = execErr
		result.Success = false

		// Try to extract exit code
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			if ws := exitErr.ProcessState.Sys(); ws != nil {
				if status, ok := ws.(interface{ ExitStatus() int }); ok {
					result.ExitCode = status.ExitStatus()
				}
			}
		} else if result.TimedOut {
			result.ExitCode = 124 // Standard timeout exit code
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

// displayHookResult displays the result of a single hook execution
func (wm *WorktreeManager) displayHookResult(result HookExecutionResult) {
	if result.Success {
		wm.printf("      ✅ Success (duration: %v)\n", result.Duration)

		// Show stdout in verbose mode
		if wm.Options.Verbose && result.Stdout != "" {
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					wm.printf("         stdout: %s\n", line)
				}
			}
		}
	} else {
		// Show error with detailed information
		errorIcon := "❌"
		if result.TimedOut {
			errorIcon = "⏰"
		}

		if result.TimedOut {
			wm.printf("      %s Timed out after %v\n", errorIcon, result.Duration)
		} else {
			wm.printf("      %s Failed (exit code: %d, duration: %v)\n", errorIcon, result.ExitCode, result.Duration)
		}

		// Show error details
		if result.Error != nil && wm.Options.Verbose {
			wm.printf("         error: %s\n", result.Error.Error())
		}

		// Show stderr if available
		if result.Stderr != "" {
			if wm.Options.Verbose {
				// In verbose mode, show all stderr lines
				lines := strings.Split(result.Stderr, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						wm.printf("         stderr: %s\n", line)
					}
				}
			} else {
				// In non-verbose mode, show first meaningful line
				lines := strings.Split(result.Stderr, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						wm.printf("         error: %s\n", strings.TrimSpace(line))
						break
					}
				}
			}
		}

		// Show helpful debugging hints for common errors
		if !wm.Options.Verbose {
			wm.showDebuggingHints(result)
		}
	}
}

// showDebuggingHints provides helpful hints for common hook execution errors
func (wm *WorktreeManager) showDebuggingHints(result HookExecutionResult) {
	if result.Success {
		return
	}

	// Check for common error patterns and provide hints
	if result.ExitCode == 127 || (result.Stderr != "" && strings.Contains(strings.ToLower(result.Stderr), "command not found")) {
		wm.printf("         💡 Hint: Command not found. Check if it's installed and in PATH\n")
	} else if result.ExitCode == 126 || (result.Stderr != "" && strings.Contains(strings.ToLower(result.Stderr), "permission denied")) {
		wm.printf("         💡 Hint: Permission denied. Check file permissions (chmod +x)\n")
	} else if result.TimedOut {
		wm.printf("         💡 Hint: Command timed out. Consider breaking into smaller steps\n")
	} else if result.ExitCode == 1 && strings.Contains(strings.ToLower(result.Command), "npm") {
		wm.printf("         💡 Hint: NPM error. Try 'npm install' or check package.json\n")
	} else if result.ExitCode != 0 && strings.Contains(strings.ToLower(result.Command), "docker") {
		wm.printf("         💡 Hint: Docker error. Check if Docker is running\n")
	}
}

// displayHookSummary displays a comprehensive summary of hook execution
func (wm *WorktreeManager) displayHookSummary(summary HookSummary) {
	if wm.Options.Quiet {
		return
	}

	fmt.Printf("\n")
	wm.printf("📊 Hook Execution Summary for %s:\n", summary.HookType)
	wm.printf("   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Overall statistics
	totalExecuted := summary.SuccessCount + summary.FailedCount
	successRate := 0
	if totalExecuted > 0 {
		successRate = (summary.SuccessCount * 100) / totalExecuted
	}

	wm.printf("   📈 Statistics:\n")
	wm.printf("      • Total hooks: %d\n", summary.TotalHooks)
	wm.printf("      • Executed: %d\n", totalExecuted)
	if summary.SkippedCount > 0 {
		wm.printf("      • Skipped: %d\n", summary.SkippedCount)
	}
	wm.printf("      • Successful: %d (✅)\n", summary.SuccessCount)
	if summary.FailedCount > 0 {
		wm.printf("      • Failed: %d (❌)\n", summary.FailedCount)
	}
	wm.printf("      • Success rate: %d%%\n", successRate)
	wm.printf("      • Total duration: %v\n", summary.TotalDuration)

	// Show overall result
	fmt.Printf("\n")
	if summary.FailedCount == 0 && summary.SuccessCount > 0 {
		wm.printf("✅ All %s hooks executed successfully!\n", summary.HookType)
	} else if summary.SuccessCount > 0 && summary.FailedCount > 0 {
		wm.printf("⚠️  %s hooks completed with mixed results\n", summary.HookType)
	} else if summary.FailedCount > 0 {
		wm.printf("❌ All executed %s hooks failed\n", summary.HookType)
	}

	// Show detailed results in verbose mode
	if wm.Options.Verbose && len(summary.Results) > 0 {
		fmt.Printf("\n")
		wm.printf("📝 Detailed Results:\n")
		for _, result := range summary.Results {
			status := "✅"
			if !result.Success {
				if result.TimedOut {
					status = "⏰"
				} else {
					status = "❌"
				}
			}
			wm.printf("   [%d] %s %s (duration: %v)\n", result.Index, status, result.Command, result.Duration)
			if !result.Success && result.Error != nil {
				wm.printf("       Error: %s\n", result.Error.Error())
			}
		}
	}

	// Show troubleshooting section for failures
	if summary.FailedCount > 0 {
		fmt.Printf("\n")
		wm.printf("🔧 Troubleshooting Failed Hooks:\n")
		wm.printf("   • Run with --verbose to see detailed error messages\n")
		wm.printf("   • Test commands manually in: %s\n", summary.WorkingDir)
		wm.printf("   • Check that all required tools are installed\n")
		wm.printf("   • Verify file permissions for scripts\n")
		wm.printf("   • Use absolute paths if relative paths fail\n")
		wm.printf("   • Consider increasing timeout for long-running commands\n")
	}

	fmt.Printf("\n")
}

// Run executes the main workflow
func (wm *WorktreeManager) Run(branchName string) error {
	wm.printf("🌳 Workie\n")
	wm.printf("==============================================\n")

	// Step 1: Detect git repository
	if err := wm.DetectGitRepository(); err != nil {
		return err
	}

	// Step 2: Load configuration
	if err := wm.LoadConfig(); err != nil {
		return err
	}

	// Step 3: Create worktrees directory
	if err := wm.CreateWorktreesDirectory(); err != nil {
		return err
	}

	// Step 4: Generate branch name if not provided
	if branchName == "" {
		branchName = wm.GenerateBranchName()
		wm.printf("🔄 Auto-generated branch name: %s\n", branchName)
	}

	// Step 5: Create worktree
	if err := wm.CreateWorktreeBranch(branchName); err != nil {
		return err
	}

	// Step 6: List all worktrees
	return wm.ListWorktrees()
}
