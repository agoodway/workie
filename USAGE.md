# Workie - Your Intelligent Coding Assistant

Workie is a powerful coding assistant designed to streamline your development workflow. Initially focused on Git worktree management, Workie is evolving into a comprehensive coding companion that helps you manage your development environment, automate repetitive tasks, and enhance your productivity.

## Quick Start

### 1. Installation

#### Option A: Using the install script (recommended)
```bash
./install.sh
```

#### Option B: Using the executable wrapper
```bash
./workie --help
```

#### Option C: Manual build
```bash
make build
./build/workie --help
```

### 2. Configuration

#### Option A: Use init command (recommended)
```bash
# Generate a comprehensive configuration file with examples
workie init

# Edit the generated .workie.yaml to match your project's needs
```

#### Option B: Copy from example
1. Copy the example configuration:
   ```bash
   cp .workie.yaml.example .workie.yaml
   ```

2. Edit `.workie.yaml` to match your project's needs:
   ```yaml
   files_to_copy:
     - .env.example
     - config/development.yaml
     - scripts/
   ```

### 3. Basic Usage

```bash
# Show help and available commands
workie

# Initialize configuration file (recommended first step)
workie init

# Create a new worktree with branch
workie feature/new-feature

# List all worktrees
workie --list

# Remove a worktree when finished
workie remove feature/completed-feature

# Remove worktree and delete the branch
workie remove feature/old-work --prune-branch
```

## Testing the Installation

### Prerequisites
- Git repository (initialize with `git init` if needed)
- Go 1.21+ (for building from source)

### Test Steps

1. **Verify executables are working:**
   ```bash
   ./workie --help
   ./install.sh
   ```

2. **Test in a Git repository:**
   ```bash
   # Initialize git if not already done
   git init
   git add .
   git commit -m "Initial commit"
   
   # Create configuration
   cp .workie.yaml.example .workie.yaml
   ```

3. **Test configuration file:**
   ```bash
   # Check YAML syntax
   # The tool will validate the configuration when run
   ```

4. **Test worktree functionality:**
   ```bash
   # This requires the Go binary to be built
   # If Go is not available, install it first
   ```

## Troubleshooting

### Error: "Go is not installed"
- Install Go from https://golang.org/dl/
- Or use a pre-built binary if available

### Error: "Binary not found"
- Run `make build` to build the binary
- Or run `./install.sh` for automatic build and install

### Error: "not a git repository"
- Make sure you're in a Git repository
- Run `git init` to initialize if needed

## Configuration Examples

### For Node.js Projects
```yaml
files_to_copy:
  - package.json
  - package-lock.json
  - .env.example
  - .eslintrc.js
  - .prettierrc
  - config/
```

### For Go Projects
```yaml
files_to_copy:
  - go.mod
  - go.sum
  - .env.example
  - config/
  - scripts/
  - Makefile
```

### For Python Projects
```yaml
files_to_copy:
  - requirements.txt
  - requirements-dev.txt
  - .env.example
  - config/
  - scripts/
  - pyproject.toml
```

## Future Capabilities and Extensibility

Workie is designed to grow with your development needs. While currently focused on Git worktree management, the architecture supports extensible functionality through configuration and plugins.

### Planned Features

#### Enhanced Project Management
- **Smart Environment Setup**: Automatically detect and configure development environments based on project type
- **Dependency Management**: Intelligent handling of package managers (npm, pip, go mod, composer, etc.)
- **Database Integration**: Seamless setup of database connections and migrations for new worktrees
- **Container Support**: Docker and Podman integration for consistent development environments

#### Code Intelligence
- **Code Analysis**: Static code analysis and suggestions for improvements
- **Test Integration**: Automatic test discovery and execution in new worktrees
- **Linting and Formatting**: Integrated code quality tools and auto-formatting
- **Security Scanning**: Built-in vulnerability detection and dependency auditing

#### Workflow Automation
- **CI/CD Integration**: Seamless integration with GitHub Actions, GitLab CI, and other platforms
- **Task Runner**: Built-in task execution and workflow automation
- **Template Management**: Project templates and boilerplate generation
- **Hook System**: Customizable pre/post hooks for various operations

#### Team Collaboration
- **Shared Configurations**: Team-wide configuration sharing and synchronization
- **Branch Policies**: Intelligent branch naming and management policies
- **Code Review Integration**: Streamlined code review workflows
- **Documentation Generation**: Automatic documentation updates and generation

### Extensibility Model

Workie's extensible architecture allows for:

#### Plugin System
```yaml
# .workie.yaml
plugins:
  - name: "eslint-integration"
    version: "^1.0.0"
    config:
      auto_fix: true
      rules: "recommended"
  
  - name: "docker-compose"
    version: "^2.1.0"
    config:
      services: ["db", "redis"]
      auto_start: true
```

#### Custom Commands
```yaml
# .workie.yaml
custom_commands:
  setup-env:
    description: "Set up development environment"
    steps:
      - "npm install"
      - "cp .env.example .env"
      - "docker-compose up -d db"
  
  run-tests:
    description: "Run test suite with coverage"
    steps:
      - "npm run test:coverage"
      - "npm run test:e2e"
```

#### Hooks and Triggers
```yaml
# .workie.yaml
hooks:
  pre_create:
    - "echo 'Setting up new worktree...'"
    - "./scripts/pre-setup.sh"
  
  post_create:
    - "npm install"
    - "./scripts/post-setup.sh"
  
  pre_switch:
    - "git stash"
  
  post_switch:
    - "npm install"
```

#### Environment-Specific Configuration
```yaml
# .workie.yaml
environments:
  development:
    files_to_copy:
      - ".env.development"
      - "config/dev.yaml"
    services:
      - "database"
      - "redis"
  
  staging:
    files_to_copy:
      - ".env.staging"
      - "config/staging.yaml"
    hooks:
      post_create:
        - "./scripts/staging-setup.sh"
```

### Configuration Schema Evolution

The `.workie.yaml` configuration will continue to evolve with backward compatibility:

```yaml
# Advanced .workie.yaml example
version: "2.0"

project:
  type: "nodejs"
  language: "typescript"
  package_manager: "npm"

files_to_copy:
  - ".env.example"
  - "package.json"
  - "tsconfig.json"
  - "config/"

services:
  database:
    type: "postgresql"
    version: "15"
    auto_migrate: true
  
  cache:
    type: "redis"
    version: "7"

tasks:
  install:
    command: "npm install"
    description: "Install dependencies"
  
  build:
    command: "npm run build"
    description: "Build the project"
    depends_on: ["install"]
  
  test:
    command: "npm test"
    description: "Run tests"
    depends_on: ["build"]

integrations:
  github:
    auto_pr: true
    template: ".github/pull_request_template.md"
  
  jira:
    auto_link: true
    project_key: "PROJ"
  
  slack:
    notifications: true
    channel: "#development"
```

### Contributing to Workie's Evolution

Workie is open to community contributions and feature requests. The modular architecture allows for:

- **Plugin Development**: Create and share plugins for specific workflows
- **Template Contributions**: Share project templates and configurations
- **Feature Requests**: Suggest new capabilities and integrations
- **Documentation**: Help improve guides and examples

The roadmap prioritizes features that enhance developer productivity while maintaining simplicity and reliability. Each new capability is designed to integrate seamlessly with existing workflows and provide immediate value to developers across different programming languages and project types.
