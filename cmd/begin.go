package cmd

import (
	"fmt"
	"strings"

	"github.com/agoodway/workie/config"
	"github.com/agoodway/workie/manager"
	"github.com/agoodway/workie/provider"
	"github.com/agoodway/workie/provider/github"
	"github.com/agoodway/workie/provider/jira"
	"github.com/agoodway/workie/provider/linear"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms/ollama"
)

var (
	issueRef string // Issue reference for creating branch from issue
	useAI    bool   // Use AI to generate branch names
)

// beginCmd represents the begin command
var beginCmd = &cobra.Command{
	Use:   "begin [branch-name]",
	Short: "Begin working on a new branch by creating a worktree",
	Long: `Begin creates a new Git worktree for isolated development on a branch.

This command will:
1. Create a new branch (or checkout an existing one)
2. Set up a worktree directory alongside your main repository
3. Copy essential files and configurations from .workie.yaml
4. Execute any post_create hooks to set up your environment
5. List all active worktrees to show your development environments

The worktree will be created in a directory named after your repository
with a "-worktrees" suffix, keeping your development organized.

Branch Creation Options:
- Provide a branch name directly: workie begin feature/my-feature
- Auto-generate a timestamp-based name: workie begin
- Create from an issue: workie begin --issue github:123
- Use AI for better branch names: workie begin --issue github:123 --ai

When using --issue, the command will:
- Fetch issue details from the configured provider
- Generate an appropriate branch name based on issue type and title
- Display issue information before creating the worktree

When using --ai with --issue:
- Uses AI to analyze the issue and generate more descriptive branch names
- Creates concise names that capture the essence of the work
- Falls back to standard generation if AI is unavailable

Configuration is read from .workie.yaml (or workie.yaml) and can specify:
- Files and directories to copy to new worktrees
- Post-creation hooks for environment setup
- Pre-removal hooks for cleanup tasks
- Issue provider settings (GitHub, Jira, Linear)

Use this to start working on a new feature, bugfix, or experiment without
affecting your main working directory.`,
	Example: `  # Begin work on a new feature
  workie begin feature/user-auth

  # Begin work on an AI integration feature
  workie begin feature/ai-integration

  # Begin work from a GitHub issue
  workie begin --issue github:123

  # Begin work from a Jira issue
  workie begin --issue jira:PROJ-456

  # Begin work from issue (uses default/only configured provider)
  workie begin --issue 123

  # Begin work with AI-generated branch name
  workie begin --issue github:123 --ai

  # Begin a hotfix with custom configuration
  workie begin hotfix/security-patch --config .workie-production.yaml

  # Begin work silently for automation
  workie begin feature/ci-pipeline --quiet

  # Begin with detailed output for debugging
  workie begin feature/complex-setup --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string

		// Check if both branch name and issue flag are provided
		if len(args) > 0 && issueRef != "" {
			return fmt.Errorf("cannot specify both branch name and --issue flag")
		}

		// Check if --ai is used without --issue
		if useAI && issueRef == "" {
			return fmt.Errorf("--ai flag requires --issue flag")
		}

		// Get branch name from args if provided
		if len(args) > 0 {
			branchName = args[0]
		}

		// Create manager with options
		opts := manager.Options{
			ConfigFile:       configFile,
			Verbose:          verbose,
			Quiet:            quiet,
			ShowInitMessages: true,
		}
		wm := manager.NewWithOptions(opts)

		// If issue flag is provided, get branch name from issue
		if issueRef != "" {
			// Detect git repository first
			if err := wm.DetectGitRepository(); err != nil {
				return fmt.Errorf("not in a git repository: %w", err)
			}

			// Load configuration to get providers
			if err := wm.LoadConfig(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Get branch name from issue
			name, err := getBranchNameFromIssue(wm, issueRef)
			if err != nil {
				return fmt.Errorf("failed to create branch from issue: %w", err)
			}
			branchName = name
		}

		// Run the main workflow with the branch name
		if err := wm.Run(branchName); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(beginCmd)

	// Add flags
	beginCmd.Flags().StringVarP(&issueRef, "issue", "i", "", "Create branch from issue reference (e.g., github:123, jira:PROJ-456, or just 123 if only one provider is configured)")
	beginCmd.Flags().BoolVar(&useAI, "ai", false, "Use AI to generate more descriptive branch names (requires --issue)")
}

// getBranchNameFromIssue fetches an issue and generates a branch name from it
func getBranchNameFromIssue(wm *manager.WorktreeManager, issueRef string) (string, error) {
	// Initialize provider registry
	registry := provider.NewRegistry()

	// Initialize providers based on configuration
	if err := initializeBeginProviders(wm, registry); err != nil {
		return "", fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Check if any providers are configured
	configuredProviders := registry.ListConfigured()
	if len(configuredProviders) == 0 {
		return "", fmt.Errorf("no issue providers are configured. Please configure providers in your .workie.yaml file")
	}

	// Parse issue reference
	providerName, issueID, err := provider.ParseIssueReference(issueRef)
	if err != nil {
		// If parsing fails, check if it's just an issue ID (no provider specified)
		if !strings.Contains(issueRef, ":") {
			// Try to use default provider
			if wm.Config.DefaultProvider != "" {
				providerName = wm.Config.DefaultProvider
				issueID = issueRef
			} else if len(configuredProviders) == 1 {
				// If only one provider is configured, use it as default
				providerName = configuredProviders[0]
				issueID = issueRef
				if verbose {
					fmt.Printf("Using %s as default provider (only configured provider)\n", providerName)
				}
			} else if len(configuredProviders) > 1 {
				// Multiple providers configured but no default specified
				return "", fmt.Errorf("multiple providers configured but no default specified. Use format 'provider:id' or set 'default_provider' in config")
			} else {
				return "", err
			}
		} else {
			return "", err
		}
	}

	// Get provider
	p, err := registry.Get(providerName)
	if err != nil {
		return "", fmt.Errorf("provider '%s' not found or not configured", providerName)
	}

	// Fetch issue
	fmt.Printf("🔍 Fetching issue %s:%s...\n", providerName, issueID)
	issue, err := p.GetIssue(issueID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch issue: %w", err)
	}

	// Display issue details
	fmt.Printf("\n📋 Creating branch from issue:\n")
	fmt.Printf("   Provider: %s\n", issue.Provider)
	fmt.Printf("   ID: %s\n", issue.ID)
	fmt.Printf("   Title: %s\n", issue.Title)
	fmt.Printf("   Type: %s\n", issue.Type)
	fmt.Printf("   Status: %s\n", issue.Status)
	if len(issue.Labels) > 0 {
		fmt.Printf("   Labels: %s\n", strings.Join(issue.Labels, ", "))
	}

	// Generate branch name
	var branchName string

	if useAI {
		// Use AI to generate branch name
		aiName, err := generateAIBranchName(wm, p, issue)
		if err != nil {
			// Fall back to standard generation if AI fails
			if verbose {
				fmt.Printf("⚠️  AI branch name generation failed: %v\n", err)
				fmt.Printf("   Falling back to standard generation...\n")
			}
			branchName = p.CreateBranchName(issue)
		} else {
			branchName = aiName
			fmt.Printf("\n🤖 AI-generated branch name: %s\n", branchName)
		}
	} else {
		// Use standard branch name generation
		branchName = p.CreateBranchName(issue)
		fmt.Printf("\n🌿 Generated branch name: %s\n", branchName)
	}

	return branchName, nil
}

// initializeBeginProviders initializes issue providers based on configuration
func initializeBeginProviders(wm *manager.WorktreeManager, registry *provider.Registry) error {
	// Get providers configuration
	providersConfig := wm.Config.Providers
	if providersConfig == nil {
		// No providers configured
		return nil
	}

	for name, config := range providersConfig {
		configMap, ok := config.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if provider is enabled
		if enabled, ok := configMap["enabled"].(bool); !ok || !enabled {
			continue
		}

		// Create provider based on type
		var p provider.Provider
		var err error

		switch name {
		case "github":
			p, err = github.NewProvider(configMap)
		case "jira":
			p, err = jira.NewProvider(configMap)
		case "linear":
			p, err = linear.NewProvider(configMap)
		default:
			if verbose {
				fmt.Printf("Unknown provider type: %s\n", name)
			}
			continue
		}

		if err != nil {
			return fmt.Errorf("failed to create %s provider: %w", name, err)
		}

		// Register provider if it's configured
		if p.IsConfigured() {
			if err := registry.Register(p); err != nil {
				return fmt.Errorf("failed to register %s provider: %w", name, err)
			}
		} else if verbose {
			fmt.Printf("Provider %s is not fully configured\n", name)
		}
	}

	return nil
}

// generateAIBranchName generates a branch name using AI
func generateAIBranchName(wm *manager.WorktreeManager, p provider.Provider, issue *provider.Issue) (string, error) {
	// Load AI configuration
	configFile := wm.Options.ConfigFile
	if configFile == "" {
		configFile = ".workie.yaml"
	}
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if !cfg.AI.Enabled {
		return "", fmt.Errorf("AI features are not enabled in configuration")
	}

	// Create Ollama client
	ollamaOpts := []ollama.Option{
		ollama.WithModel(cfg.AI.Model.Name),
	}

	if cfg.AI.Ollama.BaseURL != "" {
		ollamaOpts = append(ollamaOpts, ollama.WithServerURL(cfg.AI.Ollama.BaseURL))
	}

	llm, err := ollama.New(ollamaOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to create AI client: %w", err)
	}

	// Get the branch prefix from provider
	prefix := ""
	switch strings.ToLower(issue.Type) {
	case "bug":
		if wm.Config.Providers != nil {
			if provConfig, ok := wm.Config.Providers[p.Name()].(map[string]interface{}); ok {
				if branchPrefix, ok := provConfig["branch_prefix"].(map[string]interface{}); ok {
					if bugPrefix, ok := branchPrefix["bug"].(string); ok {
						prefix = bugPrefix
					}
				}
			}
		}
		if prefix == "" {
			prefix = "fix/"
		}
	case "feature", "enhancement", "story":
		if wm.Config.Providers != nil {
			if provConfig, ok := wm.Config.Providers[p.Name()].(map[string]interface{}); ok {
				if branchPrefix, ok := provConfig["branch_prefix"].(map[string]interface{}); ok {
					if featPrefix, ok := branchPrefix["feature"].(string); ok {
						prefix = featPrefix
					}
				}
			}
		}
		if prefix == "" {
			prefix = "feat/"
		}
	default:
		if wm.Config.Providers != nil {
			if provConfig, ok := wm.Config.Providers[p.Name()].(map[string]interface{}); ok {
				if branchPrefix, ok := provConfig["branch_prefix"].(map[string]interface{}); ok {
					if defaultPrefix, ok := branchPrefix["default"].(string); ok {
						prefix = defaultPrefix
					}
				}
			}
		}
		if prefix == "" {
			prefix = "issue/"
		}
	}

	// Create AI branch name generator
	generator := provider.NewAIBranchNameGenerator(llm)

	// Generate the branch name
	return generator.GenerateBranchName(issue, prefix)
}
