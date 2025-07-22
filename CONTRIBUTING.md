# Contributing to Workie

Thank you for your interest in contributing to Workie! We welcome contributions from the community and are excited to see what you'll bring to this agentic coding assistant CLI.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment Setup](#development-environment-setup)
- [Code Style Guidelines](#code-style-guidelines)
- [Running Tests](#running-tests)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting Guidelines](#issue-reporting-guidelines)
- [Code of Conduct](#code-of-conduct)
- [Additional Resources](#additional-resources)

## Getting Started

Before you begin contributing, please:

1. Fork the repository on GitHub
2. Read through our [Code of Conduct](#code-of-conduct)
3. Check the [Issues](https://github.com/agoodway/workie/issues) page for open tasks
4. Set up your development environment following the instructions below

## Development Environment Setup

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Git 2.5+** - Required for Git worktree functionality
- **Make** - For build automation (optional but recommended)

### Setup Steps

1. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/workie.git
   cd workie
   ```

2. **Add the upstream remote:**
   ```bash
   git remote add upstream https://github.com/agoodway/workie.git
   ```

3. **Install dependencies:**
   ```bash
   make deps
   # or manually:
   go mod download && go mod tidy
   ```

4. **Verify your setup:**
   ```bash
   make build
   ./build/workie --version
   ```

5. **Run tests to ensure everything works:**
   ```bash
   make test
   ```

### Development Workflow

1. **Create a new branch for your feature:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our [code style guidelines](#code-style-guidelines)

3. **Test your changes:**
   ```bash
   make test
   make build
   ./build/workie --help  # Test the CLI
   ```

4. **Commit and push your changes:**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   git push origin feature/your-feature-name
   ```

### Available Make Commands

- `make help` - Show all available commands
- `make build` - Build the binary for your platform
- `make test` - Run all tests
- `make clean` - Clean build artifacts
- `make deps` - Download and tidy dependencies
- `make build-all` - Cross-platform builds
- `make version` - Show version information

## Code Style Guidelines

We follow Go best practices and maintain consistency across the codebase.

### Go Standards

- **Follow [Effective Go](https://golang.org/doc/effective_go.html)** guidelines
- **Use `gofmt`** to format your code automatically
- **Run `go vet`** to catch common mistakes
- **Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)**

### Code Formatting

```bash
# Format your code (required before committing)
go fmt ./...

# Run static analysis
go vet ./...

# Optional: Use golangci-lint for comprehensive linting
golangci-lint run
```

### Naming Conventions

- **Packages:** Short, lowercase, single words (e.g., `manager`, `config`)
- **Functions:** CamelCase for exported, camelCase for private
- **Variables:** Descriptive names, avoid abbreviations unless obvious
- **Constants:** CamelCase or ALL_CAPS for package-level constants

### Project Structure

```
workie/
├── cmd/           # Cobra CLI commands
├── config/        # Configuration handling
├── manager/       # Core worktree management logic
├── docs/          # Documentation
├── examples/      # Example configurations
└── scripts/       # Build and setup scripts
```

### Code Organization

- Keep functions focused and single-purpose
- Group related functionality into packages
- Use interfaces for testability
- Add meaningful comments for exported functions
- Include examples in documentation comments when helpful

### Error Handling

- Use Go's standard error handling patterns
- Provide meaningful error messages
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Don't panic unless absolutely necessary (program cannot continue)

### Example Code Style

```go
// ConfigManager handles worktree configuration operations.
type ConfigManager struct {
    configPath string
    logger     Logger
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(configPath string, logger Logger) (*ConfigManager, error) {
    if configPath == "" {
        return nil, fmt.Errorf("config path cannot be empty")
    }
    
    return &ConfigManager{
        configPath: configPath,
        logger:     logger,
    }, nil
}

// LoadConfig reads and parses the configuration file.
func (cm *ConfigManager) LoadConfig() (*Config, error) {
    data, err := os.ReadFile(cm.configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file %s: %w", cm.configPath, err)
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }
    
    return &config, nil
}
```

## Running Tests

We use Go's built-in testing framework. All contributions should include appropriate tests.

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests in a specific package
go test ./config

# Run a specific test
go test -run TestConfigLoad ./config
```

### Test Structure

- Place test files alongside the code they test (e.g., `config_test.go`)
- Use table-driven tests for multiple test cases
- Test both success and error scenarios
- Use meaningful test names that describe what they're testing

### Example Test

```go
func TestConfigLoad(t *testing.T) {
    tests := []struct {
        name        string
        configData  string
        expectError bool
    }{
        {
            name: "valid config",
            configData: `files_to_copy:
  - .env.example
  - scripts/`,
            expectError: false,
        },
        {
            name:        "invalid yaml",
            configData:  `invalid: yaml: content:`,
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Coverage

- Aim for at least 80% test coverage for new code
- Focus on testing public APIs and error conditions
- Mock external dependencies (file system, git commands)
- Include integration tests for CLI commands

## Pull Request Process

### Before Submitting

1. **Sync with upstream:**
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Rebase your branch:**
   ```bash
   git checkout your-feature-branch
   git rebase main
   ```

3. **Run the full test suite:**
   ```bash
   make test
   make build
   ```

4. **Ensure your code is formatted:**
   ```bash
   go fmt ./...
   go vet ./...
   ```

### PR Requirements

- ✅ **Tests pass** - All existing and new tests must pass
- ✅ **Code formatted** - Run `go fmt` and `go vet`
- ✅ **Documentation updated** - Update README.md if needed
- ✅ **Tests included** - New features must include tests
- ✅ **Commit messages** - Use conventional commit format
- ✅ **No breaking changes** - Unless discussed in an issue first

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `style:` Code style changes
- `chore:` Build process or auxiliary tool changes

**Examples:**
```
feat(manager): add support for nested worktree directories

fix(config): handle missing configuration files gracefully

docs: update installation instructions for Windows users

test(config): add tests for YAML parsing edge cases
```

### PR Template

When creating a PR, please include:

1. **What changed** - Brief description of the changes
2. **Why** - Explain the motivation for the changes
3. **How to test** - Steps to test the changes
4. **Screenshots** - If applicable, for UI changes
5. **Related issues** - Link to related issues using `Closes #123`

### Review Process

1. **Automated checks** - CI will run tests and linting
2. **Code review** - At least one maintainer will review
3. **Address feedback** - Make requested changes
4. **Final approval** - Maintainer approves and merges

## Issue Reporting Guidelines

When reporting issues, please help us help you by providing detailed information.

### Before Reporting

1. **Search existing issues** - Your issue might already be reported
2. **Update to latest version** - Check if the issue persists in the latest release
3. **Try minimal reproduction** - Isolate the problem as much as possible

### Issue Template

Please include the following information:

#### Bug Reports

```markdown
## Bug Description
A clear and concise description of what the bug is.

## Steps to Reproduce
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- OS: [e.g. macOS 13.0, Ubuntu 22.04, Windows 11]
- Go version: [e.g. 1.21.0]
- Workie version: [e.g. 1.0.0]
- Git version: [e.g. 2.39.0]

## Additional Context
Add any other context about the problem here.
- Configuration files
- Command output
- Error messages
- Screenshots
```

#### Feature Requests

```markdown
## Feature Description
A clear and concise description of what you want to happen.

## Problem Statement
Explain the problem you're trying to solve.

## Proposed Solution
Describe your proposed solution.

## Alternatives Considered
Any alternative solutions or features you've considered.

## Additional Context
Add any other context or screenshots about the feature request.
```

### Issue Labels

We use labels to categorize issues:

- `bug` - Something isn't working
- `enhancement` - New feature or request
- `documentation` - Improvements or additions to documentation
- `good first issue` - Good for newcomers
- `help wanted` - Extra attention is needed
- `question` - Further information is requested

## Code of Conduct

We are committed to providing a welcoming and inspiring community for all. Please read and follow our Code of Conduct.

### Our Pledge

We pledge to make participation in our project and community a harassment-free experience for everyone, regardless of:

- Age, body size, visible or invisible disability
- Ethnicity, sex characteristics, gender identity and expression
- Level of experience, education, socio-economic status
- Nationality, personal appearance, race, religion
- Sexual identity and orientation

### Our Standards

**Positive behavior includes:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Unacceptable behavior includes:**
- The use of sexualized language or imagery
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without explicit permission
- Other conduct which could reasonably be considered inappropriate

### Enforcement

Project maintainers are responsible for clarifying standards of acceptable behavior and will take appropriate and fair corrective action in response to any instances of unacceptable behavior.

Report any incidents to the project maintainers. All complaints will be reviewed and investigated promptly and fairly.

## Additional Resources

### Learning Resources

- [Go Documentation](https://golang.org/doc/)
- [Cobra CLI Documentation](https://cobra.dev/)
- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)
- [YAML Specification](https://yaml.org/spec/)

### Development Tools

- **Recommended IDEs:** VS Code with Go extension, GoLand, Vim with vim-go
- **Linting:** [golangci-lint](https://golangci-lint.run/)
- **Testing:** Built-in Go testing, [Testify](https://github.com/stretchr/testify) for assertions
- **Documentation:** [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc)

### Communication

- **Issues:** Use GitHub Issues for bug reports and feature requests
- **Discussions:** Use GitHub Discussions for questions and community chat
- **Security:** Report security issues privately to maintainers

### Getting Help

If you need help contributing:

1. Check the [documentation](./README.md) and [usage guide](./USAGE.md)
2. Look for `good first issue` labeled issues
3. Ask questions in GitHub Discussions
4. Reach out to maintainers if you're stuck

Thank you for contributing to Workie! 🚀

---

*This contributing guide is inspired by best practices from the open-source community and is designed to help you contribute effectively to the project.*
