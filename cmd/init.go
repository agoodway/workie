package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	forceInit  bool
	outputFile string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new .workie.yaml configuration file",
	Long: `Create a new .workie.yaml configuration file in the current directory
with commented examples and best practices.

This command helps you get started with Workie by generating a comprehensive
configuration template that you can customize for your project's needs.`,
	Example: `  # Create .workie.yaml in current directory
  workie init

  # Create config file with specific name
  workie init --output custom-workie.yaml

  # Overwrite existing config file
  workie init --force

  # Create config in a specific directory
  cd /path/to/project && workie init`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := createConfigFile(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func createConfigFile() error {
	// Determine output file path
	configPath := ".workie.yaml"
	if outputFile != "" {
		configPath = outputFile
	}

	// Get absolute path for better error messages
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !forceInit {
		return fmt.Errorf("configuration file already exists: %s\n\nTo fix this:\n  • Use --force to overwrite the existing file\n  • Use --output to specify a different filename\n  • Move or rename the existing file", absPath)
	}

	// Generate configuration content
	configContent := generateConfigContent()

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing config file: %s\n\nTo fix this:\n  • Check directory permissions\n  • Ensure you have write access to the directory\n  • Try running from a different directory", absPath)
		}
		return fmt.Errorf("failed to write config file %s: %w\n\nTo fix this:\n  • Check available disk space\n  • Verify directory permissions\n  • Ensure the path is valid", absPath, err)
	}

	// Success message
	if !quiet {
		fmt.Printf("✅ Created Workie configuration file: %s\n", configPath)
		fmt.Printf("\n💡 Next steps:\n")
		fmt.Printf("  • Edit %s to customize for your project\n", configPath)
		fmt.Printf("  • Uncomment the files and directories you want to copy\n")
		fmt.Printf("  • Add project-specific files to the files_to_copy section\n")
		fmt.Printf("  • Run 'workie your-branch-name' to test the configuration\n")

		if verbose {
			fmt.Printf("\n📄 Configuration file created at: %s\n", absPath)
			fmt.Printf("📝 File size: %d bytes\n", len(configContent))
		}
	}

	return nil
}

func generateConfigContent() string {
	return `# Workie Configuration File
# =========================
# This file defines how Workie manages your development worktrees.
# Uncomment and customize the settings below for your project.

# Core Configuration - Files to Copy to New Worktrees
# ====================================================
files_to_copy:
  # Environment files (commonly needed in all worktrees)
  - .env.example
  # - .env.dev.example
  # - .env.test.example
  # - .env.local.example

  # Configuration files
  # - config/development.yaml
  # - config/testing.yaml
  # - config/staging.yaml
  # - .editorconfig
  # - .gitignore

  # Documentation
  # - README.md
  # - docs/setup.md
  # - docs/development.md
  # - CONTRIBUTING.md

  # Scripts and tools (use trailing slash for directories)
  # - scripts/
  # - tools/
  # - bin/

  # Language-specific files
  # Node.js/JavaScript
  # - package.json
  # - package-lock.json
  # - yarn.lock
  # - .eslintrc.js
  # - .prettierrc
  # - tsconfig.json
  # - jest.config.js

  # Python
  # - requirements.txt
  # - requirements-dev.txt
  # - pyproject.toml
  # - setup.py
  # - tox.ini
  # - .flake8

  # Go
  # - go.mod
  # - go.sum
  # - Makefile

  # Ruby
  # - Gemfile
  # - Gemfile.lock
  # - .ruby-version

  # Docker files
  # - Dockerfile
  # - Dockerfile.dev
  # - docker-compose.yml
  # - docker-compose.dev.yml
  # - docker-compose.test.yml
  # - .dockerignore

  # CI/CD files
  # - .github/
  # - .gitlab-ci.yml
  # - .travis.yml
  # - circle.yml

  # IDE/Editor settings (uncomment if your team uses these)
  # - .vscode/
  # - .idea/
  # - .sublime-project

# Post-creation hooks (uncomment and customize as needed)
# hooks:
#   post_create:
#     - "echo 'Setting up new worktree...'"
#     - "npm install"
#     - "make setup"
#   pre_remove:
#     - "echo 'Cleaning up worktree...'"
#     - "npm run cleanup"


# Tips for Customizing Your Configuration:
# ========================================
# 1. Start simple - uncomment just the files you need most
# 2. Use relative paths from your repository root
# 3. For directories, include the trailing slash (/)
# 4. Test your configuration with a temporary branch first
# 5. Add comments to explain project-specific choices
# 6. Consider different needs for different branch types
# 7. Keep the file under version control so your team can share it

# Common Patterns:
# ===============
# - Always copy environment examples and config files
# - Include package manager files for dependency installation
# - Copy scripts and tools that help with development
# - Include documentation that developers need to reference
# - Add IDE settings if your team standardizes on specific tools
# - Be selective with CI/CD files to avoid conflicts

# Troubleshooting:
# ===============
# - If a file doesn't exist, Workie will show a warning but continue
# - Use 'workie --verbose' to see detailed copy operations
# - Check file permissions if copies fail
# - Use 'workie --list' to see all your worktrees
# - Use 'workie finish <branch>' to clean up test worktrees

# Sprite VM Configuration (Optional)
# ===================================
# Configure tools to install on a sprite VM via ` + "`" + `workie setup-sprite` + "`" + `

# sprite:
#   name: "my-dev-vm"
#   org: "my-org"
#   pre_setup:
#     - "mkdir -p ~/go/bin"
#     - "echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc"
#     - "go install github.com/marcus/td@latest"
#     - "go install github.com/Fission-AI/OpenSpec@latest"
#   post_setup:
#     - "which td && td --version"
#     - "echo 'Setup complete'"
`
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags specific to init command
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing configuration file")
	initCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file name (default: .workie.yaml)")
}
