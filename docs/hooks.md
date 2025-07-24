# Workie Hooks Documentation

Hooks allow you to run custom commands at various stages of workie operations and Claude Code interactions. This powerful feature enables automation, validation, and integration with your development workflow.

## Overview

Workie supports two categories of hooks:
1. **Workie Lifecycle Hooks** - Triggered during worktree operations
2. **Claude Code Hooks** - Triggered during Claude Code interactions

## Configuration

Hooks are configured in your `.workie.yaml` file:

```yaml
hooks:
  # Workie lifecycle hooks
  post_create:
    - 'echo "Starting work session"'
    - 'npm install'
  pre_remove:
    - 'git status'
    - 'echo "Cleaning up"'
  
  # Claude Code hooks
  claude_pre_tool_use:
    - 'echo "Claude is about to use: $TOOL_NAME"'
  claude_post_tool_use:
    - 'echo "Tool completed: $TOOL_NAME"'
  claude_user_prompt_submit:
    - 'echo "Processing user prompt"'
  claude_stop:
    - 'echo "Claude finished responding"'
    - 'npm test'
  
  # Global timeout for all hooks (in minutes)
  timeout_minutes: 5
```

## Hook Types

### Workie Lifecycle Hooks

#### `post_create`
- **When**: After creating a new worktree
- **Use cases**: Installing dependencies, setting up environment, running initial builds
- **Example**:
  ```yaml
  post_create:
    - npm install
    - cp .env.example .env
    - echo "Worktree ready at $(pwd)"
  ```

#### `pre_remove`
- **When**: Before removing a worktree
- **Use cases**: Cleanup tasks, saving work, final checks
- **Example**:
  ```yaml
  pre_remove:
    - git status
    - echo "Removing worktree at $(pwd)"
  ```

### Claude Code Hooks

#### `claude_pre_tool_use`
- **When**: Before Claude Code uses any tool (Bash, Edit, Read, etc.)
- **Environment**: `$TOOL_NAME` contains the tool being used
- **Use cases**: Logging, validation, security checks
- **Special Feature**: Supports AI-powered decision making
- **Example**:
  ```yaml
  claude_pre_tool_use:
    - 'echo "[$(date)] Claude using tool: $TOOL_NAME" >> ~/.workie/claude.log'
    - 'test "$TOOL_NAME" = "Write" && echo "WARNING: Write operation" >&2 || true'
  ```

#### `claude_post_tool_use`
- **When**: After Claude Code successfully uses a tool
- **Environment**: `$TOOL_NAME` contains the tool that was used
- **Use cases**: Post-processing, validation, notifications
- **Example**:
  ```yaml
  claude_post_tool_use:
    - 'test "$TOOL_NAME" = "Edit" && npm run lint'
  ```

#### `claude_notification`
- **When**: Claude Code shows notifications (permissions, idle prompts)
- **Use cases**: Custom notifications, logging
- **Example**:
  ```yaml
  claude_notification:
    - 'osascript -e "display notification \"Claude needs attention\" with title \"Workie\""'
  ```

#### `claude_user_prompt_submit`
- **When**: User submits a prompt to Claude Code
- **Use cases**: Prompt logging, context injection, validation
- **Example**:
  ```yaml
  claude_user_prompt_submit:
    - 'echo "[$(date)] User prompt submitted" >> ~/.workie/prompts.log'
  ```

#### `claude_stop`
- **When**: Claude Code finishes responding (excludes interruptions)
- **Use cases**: Running tests, generating reports, cleanup
- **Example**:
  ```yaml
  claude_stop:
    - npm test
    - git status
  ```

#### `claude_subagent_stop`
- **When**: A Claude Code subagent (Task tool) finishes
- **Use cases**: Subagent-specific validation or logging
- **Example**:
  ```yaml
  claude_subagent_stop:
    - 'echo "Subagent task completed"'
  ```

#### `claude_pre_compact`
- **When**: Before Claude Code compacts context (manual or automatic)
- **Use cases**: Saving state, preparing for context reduction
- **Example**:
  ```yaml
  claude_pre_compact:
    - 'echo "Context compaction starting"'
  ```

## AI-Powered Hook Decisions

Workie supports AI-powered decision making for `claude_pre_tool_use` hooks. This feature uses an LLM to analyze hook outputs and determine whether to approve or block tool usage.

### Configuration

```yaml
hooks:
  claude_pre_tool_use:
    - 'security-check.sh "$TOOL_NAME"'
    - 'policy-validator.sh'
  ai_decision:
    enabled: true        # Enable AI decision making
    model: "llama3.2"    # Optional: override default model
    strict_mode: false   # If true, any hook failure = automatic block
```

### How It Works

1. **Hook Execution**: All `claude_pre_tool_use` hooks run normally
2. **Output Collection**: stdout, stderr, and exit codes are collected
3. **AI Analysis**: The LLM analyzes:
   - Tool name and parameters
   - Hook script outputs and warnings
   - Exit codes and error messages
4. **Decision**: Returns JSON with `approve`, `block`, or undefined (continue normal flow)

### Testing AI Decisions

```bash
# Create test input
cat > test-write.json << EOF
{
  "session_id": "test",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/etc/passwd",
    "content": "malicious content"
  }
}
EOF

# Test with AI decision
workie hooks claude-test --input test-write.json --ai

# Test without AI (rule-based only)
workie hooks claude-test --input test-write.json
```

### Decision Logic

The AI considers:
- Security implications of the tool operation
- Warnings or errors from hook scripts
- File paths and their sensitivity
- Tool parameters and potential risks

### Example Hook Scripts

```bash
#!/bin/bash
# security-check.sh
TOOL=$1

# Check for dangerous tools
if [[ "$TOOL" =~ ^(Delete|Bash)$ ]]; then
    echo "SECURITY WARNING: High-risk tool detected" >&2
    exit 1
fi

# Check for sensitive paths
if [[ "$2" =~ ^(/etc/|/sys/|/root/) ]]; then
    echo "POLICY VIOLATION: Access to system directory" >&2
    exit 2
fi

echo "Security check passed"
```

## Hook Commands

### List Configured Hooks
```bash
workie hooks list
workie hooks list -q  # Quiet mode - no output
```
Shows all hooks configured in your `.workie.yaml` file.

### Run Hooks Manually
```bash
workie hooks run <hook-type>
workie hooks run <hook-type> -q  # Quiet mode - shows only hook output
```
Manually execute hooks of a specific type. Useful for testing.

Example:
```bash
workie hooks run post_create
workie hooks run claude_pre_tool_use -q
```

### Test Hooks
```bash
workie hooks test
workie hooks test -q  # Quiet mode - exit code indicates success
```
Performs a dry run of all configured hooks to validate they're properly formatted.

### Add Hook Configuration
```bash
workie hooks add <hook-type> <command> [--timeout duration]
workie hooks add <hook-type> <command> -q  # Quiet mode - outputs only YAML
```
Generates configuration snippet for adding a new hook.

Example:
```bash
workie hooks add claude_pre_tool_use "echo 'Tool: $TOOL_NAME'" --timeout 30s
workie hooks add claude_stop "npm test" -q >> .workie.yaml
```

### Test Claude Code Hooks
```bash
workie hooks claude-test --input <json-file>
workie hooks claude-test --input <json-file> --ai  # With AI decision
```
Tests Claude Code PreToolUse hooks by simulating a tool use request.

Example:
```bash
# Create test input
echo '{"tool_name":"Write","tool_input":{"file_path":"/tmp/test.txt"}}' > test.json
workie hooks claude-test --input test.json --ai
```

### Quiet Mode
All hooks subcommands support the `-q` or `--quiet` flag:
- **list**: Suppresses all output
- **run**: Shows only hook command output, no status messages
- **test**: Shows no output, exit code indicates success (0) or failure (non-zero)
- **add**: Shows only the YAML configuration snippet
- **claude-test**: Shows only the decision JSON

## Best Practices

1. **Keep hooks fast**: Hooks run synchronously and can slow down operations
2. **Handle errors gracefully**: Use conditional logic to prevent failures
3. **Use environment variables**: Claude Code provides context through environment variables
4. **Log appropriately**: Avoid verbose output unless debugging
5. **Test thoroughly**: Use `workie hooks test` before relying on hooks

## Environment Variables

Hooks have access to standard environment variables plus hook-specific ones:

- `$TOOL_NAME` - Available in claude_pre_tool_use and claude_post_tool_use hooks
- `$USER_PROMPT` - Available in claude_user_prompt_submit hooks (if provided)
- Standard variables: `$PWD`, `$USER`, `$HOME`, etc.

## Examples

### Development Setup Hook
```yaml
hooks:
  post_create:
    - 'test -f package.json && npm install'
    - 'test -f requirements.txt && pip install -r requirements.txt'
    - 'test -f .env.example && cp .env.example .env'
    - 'echo "✅ Development environment ready"'
```

### Security Validation Hook
```yaml
hooks:
  claude_pre_tool_use:
    - |
      if [ "$TOOL_NAME" = "Bash" ]; then
        echo "⚠️  Bash command will be executed"
      fi
```

### Continuous Testing Hook
```yaml
hooks:
  claude_post_tool_use:
    - |
      if [ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "Write" ]; then
        npm test 2>/dev/null || echo "Tests need attention"
      fi
```

### Git Status Monitor
```yaml
hooks:
  claude_stop:
    - 'git diff --stat'
    - 'git status --short'
```

## Troubleshooting

### Hooks not executing
- Ensure `.workie.yaml` is in the repository root
- Check YAML syntax is valid
- Verify hook names are spelled correctly

### Hook failures
- Run `workie hooks test` to validate syntax
- Test commands manually in the worktree directory
- Check timeout settings aren't too restrictive

### Performance issues
- Reduce hook complexity
- Use background processes for long-running tasks
- Increase timeout_minutes if needed

## Integration with Claude Code

When using Claude Code with workie, hooks provide powerful integration points:

1. **Validation**: Use claude_pre_tool_use to validate operations before they occur
2. **Logging**: Track all Claude Code activities with claude_post_tool_use
3. **Testing**: Automatically run tests after code changes with claude_stop hooks
4. **Notifications**: Get alerts for important events with claude_notification hooks

Remember that Claude Code hooks require Claude Code to be configured with your `.workie.yaml` file path for the hooks to be recognized and executed.
