# Workie - Agentic Coding Assistant CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/agoodway/workie)](https://goreportcard.com/report/github.com/agoodway/workie)
[![GitHub release](https://img.shields.io/github/release/agoodway/workie.svg)](https://github.com/agoodway/workie/releases/latest)

A comprehensive developer assistant that streamlines your coding workflow with advanced Git worktree management, AI-powered features, and Claude Code integration.

## Table of Contents

- [Quick Start](#quick-start)
- [Features](#features)
- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Configuration](#configuration)
- [AI Features](#ai-features)
- [Issue Provider Integration](#issue-provider-integration)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

```bash
# Install
go install github.com/agoodway/workie@latest

# Initialize configuration
workie init

# Create a new worktree branch
workie begin feature/new-ui

# List worktrees
workie --list
```

## Features

### Core Features

- 🌳 **Smart Git Worktree Management** - Create and manage git worktrees effortlessly
- 🤖 **AI-Powered Assistant** - Generate branch names and commit messages from issue details
- 📋 **Issue Provider Integration** - Connect with GitHub, Jira, and Linear
- 🔔 **System Notifications** - Get alerts for important events
- 📁 **Smart File Copying** - Automatically copy essential files to new worktrees

### AI Capabilities

- Generate descriptive branch names from issue details
- Create meaningful commit messages
- Execute git commands with context awareness
- AI-powered security decisions for tool usage

## Installation

### Using Go Install (Recommended)

```bash
go install github.com/agoodway/workie@latest
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/agoodway/workie.git
cd workie

# Build the binary
go build -o workie .

# Optional: Install to PATH
go install .
```

### Requirements

- Go 1.21 or higher
- Git installed and configured
- Optional: [Ollama](https://ollama.com) for AI features

## Basic Usage

### Worktree Management

```bash
# Initialize configuration in your project
workie init

# Create a new worktree with a branch
workie begin feature/new-ui
workie begin bugfix/issue-123

# Create and change to new worktree
workie begin -q feature/new-feature | cd

# List all worktrees
workie --list
workie -l

# Remove a worktree
workie finish feature/completed-work
workie finish feature/old-branch --prune-branch

# Create AI-powered branch names from issues
workie begin --issue 123 --ai
workie begin --issue github:456 --ai
```

### Conflict Monitoring

```bash
# Start the watch server to monitor for rebase conflicts
workie watch

# Check every 10 minutes instead of default 5
workie watch --interval 10m

# Use a custom port
workie watch --port 8081

# Run in quiet mode
workie watch --quiet

# Access the watch server API
curl http://localhost:8080/status
curl http://localhost:8080/conflicts
curl -X POST http://localhost:8080/check
```

## Configuration

Workie uses YAML configuration files to customize behavior. Place `.workie.yaml` in your repository root.

### Basic Configuration

```yaml
# Files to copy to new worktrees
files_to_copy:
  - .env.example
  - scripts/
  - config/development.yaml

# Default issue provider
default_provider: github
```

### Initializing Configuration

The easiest way to get started:

```bash
# Create .workie.yaml with examples
workie init

# Create with custom filename
workie init --output my-config.yaml

# Overwrite existing file
workie init --force
```

## AI Features

### Setup

1. Install [Ollama](https://ollama.com)
2. Pull a model: `ollama pull zephyr`
3. Configure in `.workie.yaml`:

```yaml
ai:
  enabled: true
  model:
    provider: "ollama"
    name: "zephyr"
    temperature: 0.7
    max_tokens: 2048
  ollama:
    base_url: "http://localhost:11434"
    keep_alive: "5m"
```

### Smart Branch Names

```bash
# Standard: fix/123-update-user-authentication-to-support-oauth2
workie begin --issue 123

# AI-powered: fix/123-oauth2-auth
workie begin --issue 123 --ai
```

## Issue Provider Integration

### GitHub

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

### Jira

```yaml
providers:
  jira:
    enabled: true
    settings:
      base_url: "https://your-company.atlassian.net"
      email_env: "JIRA_EMAIL"
      api_token_env: "JIRA_TOKEN"
      project: "PROJ"
    branch_prefix:
      bug: "bugfix/"
      story: "feature/"
      default: "jira/"
```

### Using Issue Providers

```bash
# List issues
workie issues
workie issues --provider github

# View issue details
workie issues github:123
workie issues jira:PROJ-456

# Create worktree from issue
workie issues github:123 --create
workie issues jira:PROJ-456 -c

# Filter issues
workie issues --assignee me --status open
workie issues --labels bug,urgent
```

## Advanced Usage

### File Copying

Workie automatically copies specified files to new worktrees:

```yaml
files_to_copy:
  - .env.example          # Environment template
  - scripts/              # Utility scripts (copied recursively)
  - config/dev.yaml       # Development config
  - docker-compose.yml    # Docker setup
```

**Directory Structure Example:**

```
your-project/                    # Main repository
├── .workie.yaml
├── .env.example
└── scripts/

your-project-worktrees/          # Created automatically
└── feature-new-ui/
    ├── .env.example            # ✓ Copied
    └── scripts/                # ✓ Copied recursively
```

## Troubleshooting

### Common Issues

**Configuration not found:**
```bash
# Ensure .workie.yaml exists
ls -la .workie.yaml

# Initialize if missing
workie init
```

**AI features not working:**
```bash
# Check Ollama is running
ollama list

# Pull required model
ollama pull zephyr
```

### Getting Help

```bash
# Show help
workie --help

# Show version and build info
workie --version

# Report issues
# https://github.com/agoodway/workie/issues
```

## How It Works

1. **Repository Detection**: Uses `git rev-parse --show-toplevel`
2. **Configuration Loading**: Reads `.workie.yaml` from repo root
3. **Worktree Creation**: Creates `<repo>-worktrees/` directory
4. **Branch Management**: Creates new branches in separate worktrees
5. **File Copying**: Copies configured files to new worktrees

## Vision

Workie is evolving into a comprehensive agentic coding assistant that:

- **Learns your patterns** - Adapts to your coding style
- **Automates repetitive tasks** - From branch creation to testing
- **Provides intelligent suggestions** - Context-aware recommendations
- **Integrates seamlessly** - Works with your existing tools
- **Evolves with AI** - Leverages latest AI technologies

Future roadmap includes:
- Automated testing generation
- Intelligent refactoring suggestions
- Context-aware development assistance
- Advanced workflow automation

## Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Start for Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
