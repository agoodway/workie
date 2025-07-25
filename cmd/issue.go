package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/agoodway/workie/manager"
	"github.com/agoodway/workie/provider"
	"github.com/agoodway/workie/provider/github"
	"github.com/agoodway/workie/provider/jira"
	"github.com/agoodway/workie/provider/linear"
	"github.com/spf13/cobra"
)

var (
	issueProvider string
	issueStatus   string
	issueAssignee string
	issueLimit    int
	issueLabels   []string
	issueQuery    string
	issueCreate   bool
)

// issuesCmd represents the issues command
var issuesCmd = &cobra.Command{
	Use:   "issues [provider:id]",
	Short: "Work with issues from GitHub, Jira, or Linear",
	Long: `Fetch and work with issues from various issue tracking providers.

You can list issues, view issue details, or create a worktree based on an issue.

Examples:
  # List open issues from configured providers
  workie issues

  # List issues from a specific provider
  workie issues --provider github

  # List issues assigned to you
  workie issues --assignee me

  # List issues with specific status
  workie issues --status in-progress

  # View details of a specific issue
  workie issues github:123
  workie issues jira:PROJ-456
  workie issues linear:TEAM-789

  # Create a worktree from an issue
  workie issues github:123 --create
  workie issues jira:PROJ-456 -c`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIssue,
}

func init() {
	rootCmd.AddCommand(issuesCmd)

	// Add flags
	issuesCmd.Flags().StringVarP(&issueProvider, "provider", "p", "", "Filter by provider (github, jira, linear)")
	issuesCmd.Flags().StringVarP(&issueStatus, "status", "s", "", "Filter by status (open, closed, in-progress)")
	issuesCmd.Flags().StringVarP(&issueAssignee, "assignee", "a", "", "Filter by assignee (use 'me' for current user)")
	issuesCmd.Flags().IntVarP(&issueLimit, "limit", "n", 20, "Maximum number of issues to display")
	issuesCmd.Flags().StringSliceVarP(&issueLabels, "labels", "l", nil, "Filter by labels (comma-separated)")
	issuesCmd.Flags().StringVarP(&issueQuery, "query", "q", "", "Search query")
	issuesCmd.Flags().BoolVarP(&issueCreate, "create", "c", false, "Create a worktree from the issue")
}

func runIssue(cmd *cobra.Command, args []string) error {
	// Create manager with options
	opts := manager.Options{
		ConfigFile: configFile,
		Verbose:    verbose,
		Quiet:      quiet,
	}
	wm := manager.NewWithOptions(opts)

	// Detect git repository
	if err := wm.DetectGitRepository(); err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load configuration
	if err := wm.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize provider registry
	registry := provider.NewRegistry()

	// Initialize providers based on configuration
	if err := initializeProviders(wm, registry); err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	// If no providers are configured, show helpful message
	configuredProviders := registry.ListConfigured()
	if len(configuredProviders) == 0 {
		fmt.Println("No issue providers are configured.")
		fmt.Println("\nTo configure providers, add them to your .workie.yaml file:")
		fmt.Println("\nproviders:")
		fmt.Println("  github:")
		fmt.Println("    enabled: true")
		fmt.Println("    settings:")
		fmt.Println("      token_env: GITHUB_TOKEN")
		fmt.Println("      owner: your-org")
		fmt.Println("      repo: your-repo")
		fmt.Println("\nSee the documentation for more provider configuration examples.")
		return nil
	}

	// Handle specific issue reference
	if len(args) > 0 {
		return handleSpecificIssue(wm, registry, args[0])
	}

	// List issues
	return listIssues(wm, registry)
}

func initializeProviders(wm *manager.WorktreeManager, registry *provider.Registry) error {
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

func handleSpecificIssue(wm *manager.WorktreeManager, registry *provider.Registry, issueRef string) error {
	// Parse issue reference
	providerName, issueID, err := provider.ParseIssueReference(issueRef)
	if err != nil {
		// If parsing fails, check if it's just an issue ID and we have a default provider
		if wm.Config.DefaultProvider != "" && !strings.Contains(issueRef, ":") {
			providerName = wm.Config.DefaultProvider
			issueID = issueRef
		} else {
			return err
		}
	}

	// Get provider
	p, err := registry.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider '%s' not found or not configured", providerName)
	}

	// Fetch issue
	issue, err := p.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to fetch issue: %w", err)
	}

	// Display issue details
	displayIssueDetails(issue)

	// Create worktree if requested
	if issueCreate {
		branchName := p.CreateBranchName(issue)
		fmt.Printf("\n🌳 Creating worktree with branch: %s\n", branchName)

		if err := wm.CreateWorktreeBranch(branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		// TODO: Consider adding issue metadata to initial commit message
	}

	return nil
}

func listIssues(wm *manager.WorktreeManager, registry *provider.Registry) error {
	// Build filter
	filter := provider.ListFilter{
		Status:   issueStatus,
		Assignee: issueAssignee,
		Labels:   issueLabels,
		Limit:    issueLimit,
		Query:    issueQuery,
	}

	// Get list of providers to query
	var providersToQuery []string

	// If provider flag is specified, use it
	if issueProvider != "" {
		// Filter to specific provider
		found := false
		for _, p := range registry.ListConfigured() {
			if p == issueProvider {
				providersToQuery = []string{p}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("provider '%s' not found or not configured", issueProvider)
		}
	} else if wm.Config.DefaultProvider != "" {
		// Use default provider from config if no provider flag specified
		if p, err := registry.Get(wm.Config.DefaultProvider); err == nil && p.IsConfigured() {
			providersToQuery = []string{wm.Config.DefaultProvider}
		} else {
			// Fall back to all configured providers if default is not available
			providersToQuery = registry.ListConfigured()
		}
	} else {
		// No provider specified and no default, use all configured providers
		providersToQuery = registry.ListConfigured()
	}

	// Collect issues from all providers
	allIssues := make([]provider.Issue, 0)
	for _, providerName := range providersToQuery {
		p, err := registry.Get(providerName)
		if err != nil {
			continue
		}

		issueList, err := p.ListIssues(filter)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to fetch issues from %s: %v\n", providerName, err)
			}
			continue
		}

		allIssues = append(allIssues, issueList.Issues...)
	}

	// Display issues
	if len(allIssues) == 0 {
		fmt.Println("No issues found matching the criteria.")
		return nil
	}

	displayIssueList(allIssues)
	return nil
}

func displayIssueList(issues []provider.Issue) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tID\tTITLE\tSTATUS\tTYPE")
	fmt.Fprintln(w, "--------\t--\t-----\t------\t----")

	for _, issue := range issues {
		// Truncate title if too long
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			issue.Provider,
			issue.ID,
			title,
			issue.Status,
			issue.Type,
		)
	}

	w.Flush()

	fmt.Printf("\n📋 Total issues: %d\n", len(issues))
	fmt.Println("\nTo view issue details: workie issues <provider>:<id>")
	fmt.Println("To create worktree:   workie issues <provider>:<id> --create")
}

func displayIssueDetails(issue *provider.Issue) {
	fmt.Printf("📋 Issue Details\n")
	fmt.Printf("================\n\n")
	fmt.Printf("Provider:    %s\n", issue.Provider)
	fmt.Printf("ID:          %s\n", issue.ID)
	fmt.Printf("Title:       %s\n", issue.Title)
	fmt.Printf("Type:        %s\n", issue.Type)
	fmt.Printf("Status:      %s\n", issue.Status)
	fmt.Printf("URL:         %s\n", issue.URL)

	if len(issue.Labels) > 0 {
		fmt.Printf("Labels:      %s\n", strings.Join(issue.Labels, ", "))
	}

	if issue.Metadata["assignee"] != "" {
		fmt.Printf("Assignee:    %s\n", issue.Metadata["assignee"])
	}

	if issue.Metadata["created_at"] != "" {
		fmt.Printf("Created:     %s\n", issue.Metadata["created_at"])
	}

	if issue.Description != "" {
		fmt.Printf("\nDescription:\n")
		fmt.Printf("------------\n")
		// Limit description length for display
		desc := issue.Description
		if len(desc) > 500 {
			desc = desc[:497] + "..."
		}
		fmt.Printf("%s\n", desc)
	}
}
