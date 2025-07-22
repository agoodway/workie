# Workie - Agentic Coding Assistant CLI

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

### 🤖 AI-Powered Features (Maybe Roadmap)

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
go install github.com/yourusername/workie@latest
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

# Create worktree with specific branch name
workie feature/new-ui
workie bugfix/issue-123

# List existing worktrees
workie --list
workie -l

# Remove worktree when finished with branch
workie remove feature/completed-work
workie remove feature/old-branch --prune-branch
workie remove feature/experimental --force
```

### Help and Information

```bash
# Show help and available commands
workie --help

# Display version, commit, and build information
workie --version
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
   workie feature/new-branch --verbose
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

- **Environment templates**: Copy `.env.example` instead of `.env` to avoid secrets
- **Development configs**: Include development-specific configurations only
- **Script organization**: Group related scripts in directories for easy copying
- **Documentation**: Include relevant documentation files (`README.md`, setup guides)
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

### Real-World Examples

#### Full-Stack Web Application
```yaml
files_to_copy:
  # Environment setup
  - .env.example
  - .env.development.example

  # Build and development tools
  - package.json              # For npm dependencies reference
  - docker-compose.dev.yml    # Development containers
  - scripts/                  # Build and utility scripts

  # Configuration files
  - config/development.json   # App configuration
  - .eslintrc.json           # Code quality
  - jest.config.js           # Testing setup
```

#### Microservices Project
```yaml
files_to_copy:
  # Shared configurations
  - .env.example
  - docker-compose.yml
  - scripts/deploy.sh

  # Service-specific configs
  - config/
  - kubernetes/              # K8s manifests
  - docs/setup.md           # Development guide
```

By following these patterns and best practices, you'll ensure that every new worktree is properly configured and ready for development from the moment it's created.

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
  post_create:
    - "echo 'Welcome to your new environment!'"
    - "npm install"
  pre_remove:
    - "echo 'Cleaning up...'"
    - "rm -rf /tmp/*"
```

### Common Use Cases

- **Setting up development environments** with `post_create`, ensuring all dependencies are installed.
- **Cleanup tasks** using `pre_remove` to tidy up temporary files.

### Security Considerations

- Avoid destructive commands like `rm -rf /` in hooks.
- Validate all inputs and paths to prevent injection attacks.
- Limit the number of hooks and complexity to maintain performance.

## Examples

### Example 1: Basic Usage

```bash
cd /path/to/your/repo
workie feature/user-authentication

# Output:
# 🌳 Workie
# ==============================================
# ✓ Detected git repository: /path/to/your/repo
# ✓ Worktrees directory: /path/to/your-repo-worktrees
# ✓ Created worktrees directory: /path/to/your-repo-worktrees
# 📝 Creating worktree for branch 'feature/user-authentication'...
# ✅ Successfully created worktree:
#    Branch: feature/user-authentication
#    Path: /path/to/your-repo-worktrees/feature/user-authentication
#
# 🚀 To start working:
#    cd /path/to/your-repo-worktrees/feature/user-authentication
```

### Example 2: With Configuration File

Create `.workie.yaml`:
```yaml
files_to_copy:
  - .env.example
  - scripts/setup.sh
  - config/
```

Run the tool:
```bash
workie bugfix/database-connection

# Output includes:
# ✓ Loaded configuration from: .workie.yaml
# ✓ Files to copy: 3 entries
# 📂 Copying configured files to worktree...
#    📄 Copying file: .env.example
#    📄 Copying file: scripts/setup.sh
#    📁 Copying directory: config/
# ✓ Finished copying configured files
```

### Example 3: Auto-Generated Branch Names

```bash
workie

# Creates branch like: feature/work-20240120-143022
```

## Error Handling

- **Not in Git Repository**: Shows clear error message
- **Branch Already Exists**: Prevents duplicate branches
- **Missing Configuration Files**: Shows warnings but continues
- **File Copy Errors**: Shows warnings for individual files but continues

## Building and Development

### Prerequisites

- Go 1.21 or later
- Git

### Building

```bash
go mod download
go build -o workie .
```

### Testing

```bash
go test ./...
```

### Cross-Platform Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o workie-linux .

# macOS
GOOS=darwin GOARCH=amd64 go build -o workie-macos .

# Windows
GOOS=windows GOARCH=amd64 go build -o workie.exe .
```

## Roadmap

Workie is evolving beyond Git worktree management into a comprehensive agentic coding assistant. Here's what's planned:

### 🎯 Short Term (Q1-Q2 2024)
- **Enhanced configuration** - More flexible `.workie.yaml` options for project setup
- **Template system** - Pre-configured templates for common project types
- **Integration plugins** - Support for popular development tools and workflows
- **Smart branch suggestions** - AI-powered branch naming based on commit history and patterns

### 🚀 Medium Term (Q3-Q4 2024)
- **Code analysis engine** - Intelligent code quality assessment and suggestions
- **Automated test generation** - Generate unit tests based on existing code patterns
- **Workflow learning** - Learn and suggest optimizations for your development patterns
- **Context-aware assistance** - Understand project structure and provide relevant recommendations

### 🌟 Long Term (2025+)
- **Full agentic capabilities** - Autonomous code generation and refactoring
- **Natural language commands** - Describe what you want and let Workie implement it
- **Team collaboration features** - Share and synchronize development patterns across teams
- **Advanced AI integrations** - Integration with cutting-edge AI models for code generation
- **Ecosystem expansion** - Plugin architecture for community-driven extensions

### 🤝 Community Driven
- **Open source contributions** - Community-driven feature development
- **Plugin marketplace** - Ecosystem of community-created extensions
- **Best practices sharing** - Learn from and contribute to collective development wisdom

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

[Your chosen license]

## Contributing

[Your contribution guidelines]
