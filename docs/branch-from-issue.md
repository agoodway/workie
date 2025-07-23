# Creating Branches from Issues

The `workie begin` command now supports creating branches directly from issues in GitHub, Jira, or Linear. This feature automatically generates appropriate branch names based on issue type and title.

## Usage

```bash
# Create a branch from a GitHub issue
workie begin --issue github:123

# Create a branch from a Jira issue
workie begin --issue jira:PROJ-456

# Create a branch from a Linear issue
workie begin --issue linear:TEAM-789

# Short form
workie begin -i github:123
```

## How it Works

When you use the `--issue` flag:

1. Workie fetches the issue details from the configured provider
2. Displays the issue information (title, type, status, labels)
3. Generates a branch name using the provider's naming convention
4. Creates a new worktree with that branch name
5. Continues with the normal worktree setup process

## Branch Naming Conventions

Each provider generates branch names differently:

### GitHub
- Bug issues: `fix/123-issue-title`
- Feature issues: `feat/123-issue-title`
- Default: `issue/123-issue-title`

### Jira
- Bug: `bugfix/PROJ-123-issue-title`
- Story: `feature/PROJ-123-issue-title`
- Task: `task/PROJ-123-issue-title`
- Default: `jira/PROJ-123-issue-title`

### Linear
- Bug: `fix/TEAM-123-issue-title`
- Feature: `feat/TEAM-123-issue-title`
- Default: `linear/TEAM-123-issue-title`

## Configuration

To use this feature, you need to configure your issue providers in `.workie.yaml`:

```yaml
providers:
  github:
    enabled: true
    settings:
      token_env: "GITHUB_TOKEN"
      owner: "your-org"
      repo: "your-repo"
    branch_prefix:
      bug: "fix/"
      feature: "feat/"
      default: "issue/"
```

## Examples

```bash
# Fetch issue #123 from GitHub and create a worktree
$ workie begin --issue github:123

🔍 Fetching issue github:123...

📋 Creating branch from issue:
   Provider: github
   ID: 123
   Title: Add dark mode support
   Type: feature
   Status: open
   Labels: enhancement, ui

🌿 Generated branch name: feat/123-add-dark-mode-support

📝 Creating worktree for branch 'feat/123-add-dark-mode-support'...
✓ Git worktree created successfully
✓ Successfully created worktree:
   Branch: feat/123-add-dark-mode-support
   Path: /path/to/project-worktrees/feat/123-add-dark-mode-support
```

## Notes

- You cannot use both a manual branch name and the `--issue` flag at the same time
- The issue must exist and be accessible with your configured credentials
- Issue titles are sanitized to create valid Git branch names
- The generated branch name can be quite long if the issue title is verbose