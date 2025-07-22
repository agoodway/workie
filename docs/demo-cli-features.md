# CLI Features Demo

This document demonstrates the enhanced CLI features implemented for Workie.

## New CLI Features

### 1. Custom Config File Support
- **Flag:** `--config`, `-c`
- **Description:** Specify a custom configuration file path
- **Usage:** 
  ```bash
  workie --config /path/to/custom-config.yaml feature/new-branch
  workie -c my-config.yml feature/hotfix
  ```

### 2. Verbose Output Mode
- **Flag:** `--verbose`, `-v`
- **Description:** Enable detailed output with additional information
- **Usage:**
  ```bash
  workie --verbose feature/debug-branch
  workie -v
  ```

### 3. Quiet Output Mode
- **Flag:** `--quiet`, `-q`
- **Description:** Minimize output, show only essential information
- **Usage:**
  ```bash
  workie --quiet feature/silent-work
  workie -q
  ```

### 4. Enhanced Help and Examples
The help system now includes comprehensive usage examples:

```
Usage:
  workie [branch-name] [flags]

Flags:
  -c, --config string   Path to custom configuration file (default: .worktree.yaml or worktree.yaml)
  -h, --help           help for workie
  -l, --list           List existing worktrees and exit
  -q, --quiet          Enable quiet mode with minimal output
  -v, --verbose        Enable verbose output with detailed information

Examples:
  # Create worktree with auto-generated branch name
  workie

  # Create worktree with specific branch name
  workie feature/new-ui
  workie bugfix/issue-123

  # List existing worktrees
  workie --list

  # Use custom config file
  workie --config custom-worktree.yaml feature/new-feature

  # Run in quiet mode (minimal output)
  workie --quiet feature/hotfix

  # Run in verbose mode (detailed output)
  workie --verbose feature/debug
```

## Output Mode Behaviors

### Normal Mode (Default)
- Shows standard informational messages
- Displays worktree creation progress
- Shows configuration loading information
- Displays "To start working" instructions

### Verbose Mode (`--verbose`)
- Prefixes output with "VERBOSE:"
- Shows additional details like:
  - Repository name and parent directory
  - Git commands being executed
  - Source and destination paths during file copying
  - Configuration file search process

### Quiet Mode (`--quiet`)
- Minimal output - only essential information
- Still shows:
  - Error messages and warnings
  - Final worktree creation success info (branch name and path)
  - File copy warnings (important for troubleshooting)
- Suppresses:
  - Progress messages
  - Configuration loading info
  - "To start working" instructions
  - Worktree listings

## Error Handling

### Flag Validation
- Cannot use both `--verbose` and `--quiet` together
- Custom config file must exist if specified
- Provides clear error messages for invalid combinations

### Configuration File Handling
- **Default behavior:** Searches for `.worktree.yaml` then `worktree.yaml`
- **Custom config:** Must exist, returns error if not found
- **Missing default config:** Not an error - continues with empty configuration

## File Structure

The implementation includes:
- **CLI Layer:** `cmd/root.go` - Enhanced with new flags and validation
- **Manager Layer:** `manager/manager.go` - Updated with Options struct and output control
- **Config Layer:** `config/config.go` - Enhanced to support custom config paths

## Key Implementation Details

1. **Options Struct:** Clean separation of CLI options from business logic
2. **Smart Output Control:** Different message levels (always show vs. mode-dependent)
3. **Path Handling:** Supports both absolute and relative custom config paths
4. **Backward Compatibility:** All existing functionality preserved
5. **Professional CLI Patterns:** Follows Cobra best practices with proper flag groupings

## Testing the Features

```bash
# Basic usage (unchanged)
workie feature/test

# New quiet mode
workie --quiet feature/quiet-test

# New verbose mode  
workie --verbose feature/verbose-test

# Custom config
workie --config project-specific.yaml feature/custom-config

# List with verbose details
workie --list --verbose

# Help shows all new options
workie --help
```

The enhanced CLI now provides professional-grade options while maintaining full backward compatibility with existing usage patterns.
