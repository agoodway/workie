# Workie - Agentic Coding Assistant CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/agoodway/workie)](https://goreportcard.com/report/github.com/agoodway/workie)
[![GitHub release](https://img.shields.io/github/release/agoodway/workie.svg)](https://github.com/agoodway/workie/releases/latest)

A comprehensive developer assistant that streamlines your coding workflow. Starting with advanced Git worktree management, Workie is evolving into an intelligent coding companion powered by AI agents to automate common development tasks and boost your productivity.

## Vision

Workie is transforming from a simple Git worktree manager into a comprehensive agentic coding assistant that understands your development workflow. Our vision is to create an intelligent CLI companion that:

- **Learns your patterns** - Understands how you work and adapts to your coding style
- **Automates repetitive tasks** - From branch creation to code generation and testing
- **Provides intelligent suggestions** - Recommends optimal workflows based on your project context
- **Integrates seamlessly** - Works with your existing tools and development environment
- **Evolves with AI** - Leverages the latest AI technologies to enhance developer productivity

The future of Workie includes AI-powered code analysis, automated testing generation, intelligent refactoring suggestions, and context-aware development assistance.

## Features

### 🌳 Git Worktree Management (Current)

- **Smart worktree creation** - Create git worktrees with new branches effortlessly
- **Intelligent branch naming** - Auto-generate branch names based on timestamp or patterns
- **Configuration-driven setup** - YAML configuration support for copying files/directories to new worktrees
- **Worktree discovery** - List and manage existing worktrees
- **Cross-platform support** - Works seamlessly on Linux, macOS, and Windows

### 🪝 Hooks System

- **Lifecycle hooks** - Run custom commands during worktree creation and removal
- **Claude Code integration** - Hook into Claude Code events for automation
- **Event-driven automation** - Execute scripts on tool use, prompts, and completions
- **Flexible configuration** - Define hooks for 9 different event types
- **Testing and validation** - Built-in commands to test and validate hook configurations

### 🤖 AI-Powered Features

#### Current AI Capabilities

- **AI-Powered Assistant** - Ask questions about your codebase and get intelligent responses
- **Smart Branch Names** - Generate descriptive branch names from issue details using AI
- **Tool Integration** - AI can execute git commands, analyze files, and suggest improvements
- **Commit Message Generation** - Create meaningful commit messages based on your changes

#### Roadmap

- **Code analysis and suggestions** - Intelligent code review and improvement recommendations
- **Automated testing generation** - Generate unit tests based on your code patterns
- **Context-aware assistance** - Understand your project structure and provide relevant help
- **Workflow automation** - Learn and automate your common development patterns

## Installation

### From Source

```bash
# Clone or navigate to the project directory
cd workie

# Build the binary
go build -o workie .

# Optionally, install to your PATH
go install .
```

### Direct Installation

```bash
go install github.com/agoodway/workie@latest
```

## Usage

### Basic Commands

```bash
# Show help and available commands
workie

# Show version information
workie --version

# Initialize configuration file in your project
workie init

# Begin work on a new branch with worktree
workie begin feature/new-ui
workie begin bugfix/issue-123

# Begin work and change directory to it (using quiet mode)
cd $(workie begin -q feature/new-feature)

# List existing worktrees
workie --list
workie -l

# Finish working on a branch (remove worktree)
workie finish feature/completed-work
workie finish feature/old-branch --prune-branch
workie finish feature/experimental --force
```

### Hook Commands

```bash
# List all configured hooks
workie hooks list

# Run hooks manually (for testing)
workie hooks run post_create
workie hooks run claude_pre_tool_use

# Test all hooks (dry run validation)
workie hooks test

# Generate hook configuration
workie hooks add claude_stop "npm test" --timeout 2m
```

### AI-Powered Commands

```bash
# Ask AI about your codebase
workie ask "What does the main function do?"
workie ask "How do I add a new provider?"

# Use AI with tools (function calling)
workie ask -t "Create a commit message for my changes"
workie ask -t "What are the recent commits?"
workie ask -t "List all test files"

# Create branches from issues with AI-generated names
workie begin --issue 123 --ai
workie begin --issue github:456 --ai
workie begin --issue jira:PROJ-789 --ai

# When only one provider is configured, omit the provider prefix
workie begin --issue 123  # Uses the only configured provider
```

### Help and Information

```bash
# Show help and available commands
workie --help

# Display version, commit, and build information
workie --version
```

### Issue Provider Integration

Workie can integrate with GitHub, Jira, and Linear to streamline your issue-based workflow:

```bash
# List issues from configured providers
workie issues

# List issues from a specific provider
workie issues --provider github

# View issue details
workie issues github:123
workie issues jira:PROJ-456
workie issues linear:TEAM-789

# Create a worktree from an issue
workie issues github:123 --create
workie issues jira:PROJ-456 -c

# Filter issues
workie issues --assignee me --status open
workie issues --labels bug,urgent --limit 10
```

## Configuration

The tool supports YAML configuration files that specify files and directories to automatically copy to new worktrees. This is useful for environment files, configuration files, and setup scripts that should be available in each worktree.

### Quick Start with Init Command

The easiest way to get started is to use the `init` command to generate a configuration file:

```bash
# Create .workie.yaml with comprehensive examples
workie init

# Create with custom name
workie init --output my-config.yaml

# Overwrite existing file
workie init --force
```

This creates a well-documented configuration file with:
- **Commented examples** for all common file types
- **Language-specific sections** (Node.js, Python, Go, Ruby, etc.)
- **Best practices** and usage tips
- **Future feature previews** with placeholder configurations

### Configuration File Location

Place one of these files in your repository root:
- `.workie.yaml` (preferred, hidden file)
- `workie.yaml` (alternative)

### Configuration Format

```yaml
files_to_copy:
  - .env.example
  - .env.dev.example
  - config/development.yaml
  - scripts/
  - docs/setup.md
  - docker-compose.dev.yml
```

### Configuration Features

- **Files**: Specify individual files to copy
- **Directories**: Specify directories to copy recursively (end with `/` for clarity)
- **Relative paths**: All paths are relative to the repository root
- **Automatic creation**: Destination directories are created automatically
- **Error handling**: Missing files/directories show warnings but don't stop the process

## 🚀 File Copying

The file copying feature is one of Workie's most powerful capabilities, automatically copying essential files and directories whenever a new worktree is created. This ensures consistent setup across all your branches.

### How File Copying Works

1. **Configuration**: Define files and directories in `.workie.yaml`
2. **Automatic Detection**: When creating a worktree, Workie reads the configuration
3. **Smart Copying**: Files are copied with proper directory structure
4. **Error Handling**: Missing files generate warnings but don't stop the process

### Visual Directory Structure

**Before** (main repository):
```
your-project/
├── .workie.yaml
├── .env.example
├── scripts/
│   └── setup.sh
└── config/
    └── development.yaml
```

**After** creating a new worktree:
```
your-project-worktrees/
└── feature-new-ui/
    ├── .env.example         # ✓ Copied
    ├── scripts/             # ✓ Copied recursively
    │   └── setup.sh
    └── config/              # ✓ Copied recursively
        └── development.yaml
```

### Detailed Configuration Examples

#### Example 1: Development Environment Setup
```yaml
files_to_copy:
  - .env.example              # Environment variables template
  - .env.dev.example          # Development-specific environment
  - scripts/                  # All utility scripts
  - config/development.yaml   # Development configuration
  - docker-compose.dev.yml    # Development Docker setup
```

#### Example 2: Language-Specific Configurations
```yaml
# Node.js project
files_to_copy:
  - .nvmrc                    # Node version specification
  - .env.example
  - scripts/
  - jest.config.js            # Testing configuration
  - .eslintrc.json           # Linting rules
```

```yaml
# Python project
files_to_copy:
  - .python-version           # Python version specification
  - .env.example
  - requirements-dev.txt      # Development dependencies
  - pytest.ini               # Test configuration
  - setup.cfg                 # Tool configurations
```

### Advanced File Copying Features

#### Directory Handling
- Directories are copied **recursively** with full structure
- Empty directories are created if needed
- Nested subdirectories maintain their hierarchy

#### Error Resilience
- **Missing source files**: Shows warning, continues processing
- **Permission issues**: Shows detailed error, continues with other files
- **Path conflicts**: Overwrites existing files in destination

#### Path Resolution
- All paths are **relative to repository root**
- Supports both files and directories
- Automatically creates destination directory structure

### Troubleshooting File Copying

#### Common Issues and Solutions

**❌ File not found warning**
```
Warning: Failed to copy file 'missing-file.txt': file does not exist
```
**Solution**: Verify the file path is correct and relative to repository root

**❌ Permission denied**
```
Error: Failed to copy file '.env.example': permission denied
```
**Solution**: Check file permissions and ensure write access to destination

**❌ Configuration not loaded**
```
No configuration file found
```
**Solution**: Ensure `.workie.yaml` exists in repository root and is valid YAML

#### Debugging Tips

1. **Use verbose mode** to see detailed copying logs:
   ```bash
   workie begin feature/new-branch --verbose
   ```

2. **Validate your YAML configuration**:
   ```bash
   # Test YAML syntax
   python -c "import yaml; yaml.safe_load(open('.workie.yaml'))"
   ```

3. **Check file paths** from repository root:
   ```bash
   # Verify files exist
   ls -la .env.example config/
   ```

### Best Practices for File Copying

#### ✅ Recommended Practices

- **Development configs**: Include development-specific configurations only
- **Script organization**: Group related scripts in directories for easy copying
- **Version control**: Always version control your `.workie.yaml` configuration

#### ❌ What to Avoid

- **Large binary files**: Avoid copying large assets or compiled binaries
- **Sensitive data**: Never copy files containing secrets or credentials
- **Generated files**: Don't copy build artifacts or generated content
- **OS-specific files**: Avoid copying system-specific configurations

#### 🎯 Optimization Tips

- **Minimal file set**: Only copy files essential for initial development
- **Directory grouping**: Organize related files in directories for cleaner config
- **Environment separation**: Use different configs for different environments
- **Documentation**: Comment your `.workie.yaml` to explain why files are copied

## How It Works

1. **Repository Detection**: Detects the current git repository using `git rev-parse --show-toplevel`
2. **Configuration Loading**: Loads YAML configuration if present
3. **Directory Setup**: Creates a `<repo-name>-worktrees` directory alongside your repository
4. **Branch Creation**: Creates a new git worktree with a new branch
5. **File Copying**: Copies configured files/directories to the new worktree
6. **Summary**: Shows the new worktree location and lists all worktrees

## Directory Structure

```
your-project/                    # Your main repository
your-project-worktrees/          # Worktrees directory (created automatically)
├── feature-new-ui/              # Worktree for feature/new-ui branch
├── bugfix-issue-123/            # Worktree for bugfix/issue-123 branch
└── feature-work-20240120-143022/ # Auto-generated branch name
```

## Hooks

Hooks allow you to execute commands at different stages in the lifecycle of a worktree:

- **post_create**: Commands to run after creating a new worktree.
- **pre_remove**: Commands to run before removing a worktree.

### Configuration Example

Here's how you can set hooks in your configuration file:

```yaml
hooks:
  timeout_minutes: 5  # Optional: Timeout in minutes for each hook command

  # Workie lifecycle hooks
  post_create:
    - "echo 'Welcome to your new environment!'"
    - "npm install"
  pre_remove:
    - "echo 'Cleaning up...'"
    - "git status"

  # Claude Code integration hooks
  claude_pre_tool_use:
    - 'echo "Tool being used: $TOOL_NAME"'
  claude_post_tool_use:
    - 'test "$TOOL_NAME" = "Edit" && npm run lint'
  claude_user_prompt_submit:
    - 'echo "Processing prompt..."'
  claude_stop:
    - "npm test"
    - "echo 'Session complete'"
```

### Common Use Cases

- **Setting up development environments** with `post_create`, ensuring all dependencies are installed
- **Cleanup tasks** using `pre_remove` to tidy up temporary files
- **Automated testing** with `claude_stop` hook to run tests after Claude Code finishes
- **Tool monitoring** with `claude_pre_tool_use` and `claude_post_tool_use` for logging and validation
- **Session tracking** with `claude_user_prompt_submit` and `claude_stop` for analytics

For detailed documentation on all available hooks and their usage, see [docs/hooks.md](docs/hooks.md).

### Security Considerations

- Avoid destructive commands like `rm -rf /` in hooks.
- Validate all inputs and paths to prevent injection attacks.
- Limit the number of hooks and complexity to maintain performance.

## Claude Code Hooks Integration ⚠️

> ### 🚨 **EXPERIMENTAL FEATURE - USE AT YOUR OWN RISK** 🚨
>
> **⚠️ CRITICAL WARNING ⚠️**
>
> The Claude Code hooks integration is an **EXPERIMENTAL** and **UNOFFICIAL** feature that interfaces with Claude Code's hook system.
>
> **THIS INTEGRATION:**
> - ❌ Is **NOT** officially supported or endorsed by Anthropic
> - ❌ May **BREAK** without warning when Claude Code updates
> - ❌ Executes **ARBITRARY SHELL COMMANDS** based on AI decisions
> - ❌ Could **INTERFERE** with Claude Code's normal operation
> - ❌ Has **NOT** been extensively tested in production environments
> - ❌ May cause **DATA LOSS** or **SECURITY VULNERABILITIES** if misconfigured
>
> **BY USING THIS FEATURE, YOU EXPLICITLY ACKNOWLEDGE AND ACCEPT THAT:**
> - ✋ You understand **ALL RISKS** involved
> - ✋ You take **FULL RESPONSIBILITY** for any consequences
> - ✋ You will **NOT** hold Workie maintainers or contributors liable
> - ✋ You will implement proper **SECURITY MEASURES** and **TESTING**
> - ✋ You are using this in a **SAFE, ISOLATED ENVIRONMENT**
> - ✋ You have **BACKUPS** of all important data
>
> **⚡ PROCEED WITH EXTREME CAUTION ⚡**

### Claude Code Hook Types

Workie supports all Claude Code hook events:

```yaml
hooks:
  # Before Claude uses any tool (Bash, Edit, Read, etc.)
  claude_pre_tool_use:
    - 'echo "[$(date)] Tool: $TOOL_NAME" >> ~/.workie/claude.log'
    - 'security-check.sh "$TOOL_NAME"'

  # After Claude successfully uses a tool
  claude_post_tool_use:
    - 'test "$TOOL_NAME" = "Edit" && npm run lint || true'

  # When user submits a prompt
  claude_user_prompt_submit:
    - 'echo "New prompt received" | notify-send'

  # When Claude finishes responding
  claude_stop:
    - 'npm test --silent'
    - 'git diff --stat'

  # Other supported hooks
  claude_notification:        # On Claude notifications
  claude_subagent_stop:      # When subagent finishes
  claude_pre_compact:        # Before context compaction
```

### AI-Powered Tool Use Decisions 🤖

Workie can use AI to analyze hook outputs and decide whether to approve or block tool usage:

```yaml
hooks:
  claude_pre_tool_use:
    # Security scripts that check tool usage
    - 'check-file-paths.sh'
    - 'validate-tool-params.sh'
    - 'policy-enforcer.sh'

  # Enable AI decision making
  ai_decision:
    enabled: true
    model: "llama3.2"      # Optional: override default
    strict_mode: false     # If true, any hook failure = block
```

#### How AI Decisions Work

1. **Hook Execution**: Your security scripts run and produce output
2. **AI Analysis**: The LLM analyzes:
   - Tool name and parameters
   - Script outputs (stdout/stderr)
   - Exit codes and warnings
   - Security implications
3. **Decision**: Returns JSON to Claude Code:
   ```json
   {
     "decision": "block",
     "reason": "Security policy violation detected"
   }
   ```

### Testing Claude Code Hooks

```bash
# Create a test scenario
cat > test-write.json << EOF
{
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/etc/sensitive.conf",
    "content": "test data"
  }
}
EOF

# Test your hooks with AI decision
workie hooks claude-test --input test-write.json --ai

# Test without AI (rule-based only)
workie hooks claude-test --input test-write.json
```

### Example: Security Policy Enforcement

```yaml
hooks:
  claude_pre_tool_use:
    - |
      #!/bin/bash
      # Inline security check
      case "$TOOL_NAME" in
        Write|Edit)
          if [[ "$1" =~ ^/etc/|^/sys/|^/root/ ]]; then
            echo "BLOCKED: System file modification attempt" >&2
            exit 1
          fi
          ;;
        Bash)
          echo "WARNING: Shell execution requested" >&2
          ;;
      esac
    - 'audit-log.sh "$TOOL_NAME" "$@"'
```

### Example: Development Workflow Automation

```yaml
hooks:
  # Auto-format on edit
  claude_post_tool_use:
    - 'test "$TOOL_NAME" = "Edit" && prettier --write . || true'

  # Run tests after Claude finishes
  claude_stop:
    - 'npm test'
    - 'echo "✅ Session complete. Test results above."'

  # Track Claude's activity
  claude_pre_tool_use:
    - 'echo "[$(date)] $TOOL_NAME" >> ~/.claude-activity.log'
```

### Configuring Claude Code to Use Workie Hooks

To integrate Workie with Claude Code's native hook system, you need to modify your Claude settings file. Here's how:

1. **Locate your Claude settings file**:
   - User settings: `~/.claude/settings.json`
   - Project settings: `.claude/settings.json` (in your project root)
   - Local settings: `.claude/settings.local.json` (not committed to git)

2. **Add Workie hook commands to your Claude settings**:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "workie hooks run claude_pre_tool_use"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "workie hooks run claude_post_tool_use"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "workie hooks run claude_user_prompt_submit"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "workie hooks run claude_stop"
          }
        ]
      }
    ]
  }
}
```

3. **Configure your Workie hooks** in `.workie.yaml`:

```yaml
hooks:
  claude_pre_tool_use:
    - 'echo "Tool: $TOOL_NAME" >> ~/.workie/claude-activity.log'
    - 'security-check.sh "$TOOL_NAME"'

  claude_post_tool_use:
    - 'test "$TOOL_NAME" = "Edit" && npm run lint || true'

  claude_stop:
    - 'npm test'
    - 'git status --short'

  # Enable AI decision making for security
  ai_decision:
    enabled: true
    model: "zephyr"
```

4. **Test the integration**:

```bash
# Test that Claude Code can call Workie hooks
workie hooks test

# Manually test a specific hook type
workie hooks run claude_pre_tool_use

# Test with Claude Code input simulation
echo '{"tool_name":"Write","tool_input":{"file_path":"/tmp/test.txt"}}' | workie hooks claude-test --ai
```

This setup allows you to:
- Use Workie's configuration management for your Claude Code hooks
- Leverage Workie's AI decision-making capabilities for tool approval/blocking
- Maintain hook configurations in version control with your project
- Test hooks independently before using them with Claude Code

### Best Practices for Claude Code Hooks

1. **Test Thoroughly**: Always test hooks in a safe environment first
2. **Fail Gracefully**: Use `|| true` to prevent blocking on non-critical failures
3. **Log Everything**: Keep audit logs of tool usage and decisions
4. **Performance**: Keep hooks fast to avoid slowing down Claude Code
5. **Security First**: Implement defense in depth with multiple validation layers

### Troubleshooting

- **Hooks not triggering**: Ensure Claude Code is configured to use your hooks
- **AI decisions failing**: Check that Ollama is running and the model is available
- **Performance issues**: Reduce hook complexity or increase timeout settings
- **False positives**: Tune your security scripts and AI prompts

Remember: This integration is experimental. Always have backups and test in isolated environments!

## AI Configuration

Workie integrates with Ollama for local AI capabilities. Configure AI features in your `.workie.yaml`:

```yaml
ai:
  enabled: true
  model:
    provider: "ollama"
    name: "llama3.2"         # or any Ollama model you have installed
    temperature: 0.7
    max_tokens: 2048
  ollama:
    base_url: "http://localhost:11434"  # Default Ollama URL
    keep_alive: "5m"
```

### Setting Up AI Features

1. **Install Ollama**: Download from https://ollama.com
2. **Pull a model**: Run `ollama pull llama3.2` or your preferred model
3. **Enable in config**: Set `ai.enabled: true` in your `.workie.yaml`

### AI Use Cases

#### Smart Branch Names from Issues
When creating branches from issues, AI analyzes the issue context to generate descriptive names:

```bash
# Standard branch name: fix/123-update-user-authentication-to-support-oauth2
workie begin --issue 123

# AI-generated name: fix/123-oauth2-auth
workie begin --issue 123 --ai
```

#### Intelligent Code Assistant
Ask questions about your codebase and get context-aware answers:

```bash
# Understanding code
workie ask "What does the WorktreeManager do?"

# Finding files
workie ask -t "Find all test files related to authentication"

# Generating code
workie ask -t "Create a unit test for the parseIssueReference function"

# Git operations
workie ask -t "Show me the last 5 commits with their messages"
```

## Issue Provider Configuration

Workie can connect to GitHub, Jira, and Linear to fetch issues and create worktrees based on them. Configure providers in your `.workie.yaml`:

### Setting a Default Provider

You can set a default provider to use when no provider is specified in issue commands:

```yaml
default_provider: github
```

With this setting, you can use simplified commands:
- `workie issues 123` instead of `workie issues github:123`
- `workie issues 123 --create` instead of `workie issues github:123 --create`

### GitHub Provider

```yaml
providers:
  github:
    enabled: true
    settings:
      token_env: "GITHUB_TOKEN"  # Environment variable with your GitHub token
      owner: "your-org"          # Repository owner or organization
      repo: "your-repo"          # Repository name
    branch_prefix:
      bug: "fix/"
      feature: "feat/"
      default: "issue/"
```

### Jira Provider

```yaml
providers:
  jira:
    enabled: true
    settings:
      base_url: "https://your-company.atlassian.net"
      email_env: "JIRA_EMAIL"      # Environment variable with your Jira email
      api_token_env: "JIRA_TOKEN"  # Environment variable with your Jira API token
      project: "PROJ"              # Default project key
    branch_prefix:
      bug: "bugfix/"
      story: "feature/"
      task: "task/"
      default: "jira/"
```

### Linear Provider

```yaml
providers:
  linear:
    enabled: true
    settings:
      api_key_env: "LINEAR_API_KEY"  # Environment variable with your Linear API key
      team_id: "TEAM"                # Optional: filter by team
    branch_prefix:
      bug: "fix/"
      feature: "feat/"
      default: "linear/"
```

### Setting Up Authentication

1. **GitHub**: Create a personal access token at https://github.com/settings/tokens
2. **Jira**: Create an API token at https://id.atlassian.com/manage-profile/security/api-tokens
3. **Linear**: Create an API key at https://linear.app/settings/api

Store these tokens in environment variables:

```bash
export GITHUB_TOKEN="your-github-token"
export JIRA_EMAIL="your-email@company.com"
export JIRA_TOKEN="your-jira-api-token"
export LINEAR_API_KEY="your-linear-api-key"
```

## Error Handling

- **Not in Git Repository**: Shows clear error message
- **Branch Already Exists**: Prevents duplicate branches
- **Missing Configuration Files**: Shows warnings but continues
- **File Copy Errors**: Shows warnings for individual files but continues

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details on:

- How to set up your development environment
- Our code style and standards
- The pull request process
- How to report issues

For major changes, please open an issue first to discuss what you would like to change.

### Quick Start for Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

See our [Contributing Guide](CONTRIBUTING.md) for detailed instructions.
