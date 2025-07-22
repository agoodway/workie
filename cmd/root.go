package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

// Version information - updated during build
var (
	Version   = "dev"     // Will be set during build with ldflags
	Commit    = "unknown" // Will be set during build with ldflags
	BuildDate = "unknown" // Will be set during build with ldflags
)

var (
	listFlag    bool
	configFile  string
	verbose     bool
	quiet       bool
	versionFlag bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "workie [branch-name]",
	Short: "Workie - An agentic coding assistant CLI with Git worktree management",
	Long: `Workie is an agentic coding assistant CLI that streamlines development workflows
through intelligent Git worktree management. It creates isolated development
environments for each branch while maintaining your productivity.

Workie automates the creation of new branches and sets up worktree directories
alongside your main repository, copying essential files and configurations to
keep your development environment consistent across branches.

It supports YAML configuration files (.workie.yaml or workie.yaml) that can
specify files and directories to copy to new worktrees, enabling reproducible
development environments.

Configuration example:
  files_to_copy:
    - .env.example
    - config/
    - scripts/setup.sh`,
	Example: `  # Show help (when no arguments are provided)
  workie

  # Initialize a new project with configuration
  workie init

  # Create isolated development environments for different features
  workie feature/user-auth
  workie feature/ai-integration
  workie hotfix/security-patch

  # List all active development environments
  workie --list

  # Remove worktree when finished with a branch
  workie remove feature/completed-feature
  workie remove feature/old-work --prune-branch --force

  # Use custom configuration for specific workflows
  workie --config .workie-production.yaml feature/deployment

  # Work silently for automated scripts
  workie --quiet feature/ci-pipeline

  # Debug environment setup with detailed output
  workie --verbose feature/complex-setup`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle version flag
		if versionFlag {
			printVersion()
			return
		}

		// Validate conflicting flags
		if verbose && quiet {
			fmt.Fprintf(os.Stderr, "❌ Error: cannot use both --verbose and --quiet flags together\n")
			fmt.Fprintf(os.Stderr, "\nUsage tips:\n")
			fmt.Fprintf(os.Stderr, "  • Use --verbose for detailed output\n")
			fmt.Fprintf(os.Stderr, "  • Use --quiet for minimal output\n")
			fmt.Fprintf(os.Stderr, "  • Use neither for normal output\n")
			os.Exit(1)
		}

		// Validate custom config file exists if specified
		if configFile != "" {
			if err := validateConfigFile(configFile); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Configuration file error: %v\n", err)
				os.Exit(1)
			}
		}

		// Create manager with options
		opts := manager.Options{
			ConfigFile: configFile,
			Verbose:    verbose,
			Quiet:      quiet,
		}
		wm := manager.NewWithOptions(opts)

		// Handle list flag
		if listFlag {
			if err := wm.DetectGitRepository(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}
			if err := wm.ListWorktrees(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// If no arguments are provided, show help instead of creating worktree
		if len(args) == 0 {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			}
			return
		}

		// Run the main workflow with provided branch name
		branchName := args[0]
		if err := wm.Run(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// validateConfigFile performs early validation of the config file path
// to provide better error messages before attempting to create worktrees
func validateConfigFile(configPath string) error {
	if configPath == "" {
		return nil // No config file specified, this is fine
	}

	// Check if the path exists
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to provide helpful suggestions
			absPath, _ := filepath.Abs(configPath)
			return fmt.Errorf("config file not found: %s\n\nResolved to: %s\n\nTo fix this:\n  • Check that the file path is correct\n  • Ensure the file exists\n  • Use a relative path from the current directory\n  • Use an absolute path if needed", configPath, absPath)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing config file: %s\n\nTo fix this:\n  • Check file permissions\n  • Ensure you have read access to the file\n  • Consider using a different config file location", configPath)
		}
		return fmt.Errorf("cannot access config file %s: %w\n\nTo fix this:\n  • Verify the file path is correct\n  • Check file permissions\n  • Ensure the file system is accessible", configPath, err)
	}

	// Check if it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("config path is a directory, not a file: %s\n\nTo fix this:\n  • Specify a file path, not a directory\n  • Use a .yaml or .yml file\n  • Example: --config path/to/config.yaml", configPath)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(configPath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("config file should have .yaml or .yml extension: %s\n\nCurrent extension: %s\n\nTo fix this:\n  • Rename the file to have .yaml or .yml extension\n  • Ensure the file contains valid YAML content", configPath, ext)
	}

	// Check file size - warn if suspiciously large
	if info.Size() > 1024*1024 {
		return fmt.Errorf("config file is unusually large (%d bytes): %s\n\nTo fix this:\n  • Verify this is the correct config file\n  • Config files should typically be small\n  • Check for accidental binary content", info.Size(), configPath)
	}

	// Check if file is empty
	if info.Size() == 0 {
		return fmt.Errorf("config file is empty: %s\n\nTo fix this:\n  • Add configuration content to the file\n  • Example content:\n    files_to_copy:\n      - .env.example\n      - config/\n  • Or remove the --config flag to use defaults", configPath)
	}

	return nil
}

// printVersion displays version information in a clean, readable format
func printVersion() {
	fmt.Printf("Workie - Agentic Coding Assistant CLI\n")
	fmt.Printf("Version: %s\n", Version)
	if Commit != "unknown" {
		fmt.Printf("Commit: %s\n", Commit)
	}
	if BuildDate != "unknown" {
		fmt.Printf("Built: %s\n", BuildDate)
	}
	fmt.Printf("\n")
	fmt.Printf("An intelligent Git worktree manager evolving into a comprehensive coding assistant.\n")
	fmt.Printf("Learn more: https://github.com/agoodway/workie\n")
}

func init() {
	// Add flags
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Show version information and exit")
	rootCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List existing worktrees and exit")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to custom configuration file (default: .workie.yaml or workie.yaml)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output with detailed information")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Enable quiet mode with minimal output")

	// Mark config flag as accepting a filename
	if err := rootCmd.MarkFlagFilename("config", "yaml", "yml"); err != nil {
		// This is non-critical, so we just log it
		fmt.Fprintf(os.Stderr, "Warning: failed to mark config flag as filename: %v\n", err)
	}
}
