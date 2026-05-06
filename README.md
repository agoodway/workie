# Workie

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/agoodway/workie)](https://goreportcard.com/report/github.com/agoodway/workie)
[![GitHub release](https://img.shields.io/github/release/agoodway/workie.svg)](https://github.com/agoodway/workie/releases/latest)

A developer CLI for Git worktree management, issue-driven branching, and AI-powered autopilot workflows.

## Install

```bash
go install github.com/agoodway/workie@latest
```

Or build from source:

```bash
git clone https://github.com/agoodway/workie.git
cd workie
go build -o workie .
```

Requires Go 1.22+.

## Commands

### `workie begin [branch-name]`

Create a new worktree for isolated development.

```bash
workie begin feature/new-ui
workie begin --issue 123              # branch name from issue
workie begin --issue github:456       # explicit provider
```

Creates a worktree under `<repo>-worktrees/<branch>/`, copies configured files, and runs `post_create` hooks.

### `workie finish [branch-name]`

Remove a worktree.

```bash
workie finish feature/old-branch
workie finish feature/old-branch --prune-branch   # also delete the branch
workie finish feature/old-branch --force           # skip uncommitted changes check
```

### `workie autopilot <change> [change...]`

Run the full OpenSpec aviation pipeline by spawning AI backend sessions for each phase:

1. **prepare-for-takeoff** — Create td issues from `tasks.md`
2. **takeoff** — Create branch, implement all issues
3. **pre-landing-check** — Multi-specialist code review
4. **prepare-for-landing** — Fix review findings
5. **land-the-plane** — Verify, approve, commit, open PR

Each change must have a directory at `openspec/changes/<name>/` with a `tasks.md` file.

```bash
workie autopilot add-user-auth
workie autopilot add-user-auth improve-dashboard   # multiple changes
workie autopilot add-user-auth --start-phase 3     # resume from phase 3
workie autopilot add-user-auth --dry-run            # preview without running
workie autopilot add-user-auth --show-output        # stream AI output
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--backend` | `claude` | AI backend (`claude`, `codex`, `opencode`) |
| `--model` | — | Override the model for the backend |
| `--start-phase` | `1` | Phase to start from |
| `--phase-timeout` | `45m` | Timeout per phase |
| `--auto-approve` | `false` | Skip confirmation prompts |
| `--no-commit` | `false` | Skip intermediate commits |
| `--dry-run` | `false` | Show plan without executing |
| `--show-output` | `false` | Stream backend output |
| `--notify` | `true` | OS notifications on phase completion |

### `workie setup-sprite`

Run setup commands on [sprite.dev](https://sprite.dev) VMs.

```bash
workie setup-sprite
workie setup-sprite --sprite my-vm --org my-org
```

### `workie init`

Generate a `.workie.yaml` configuration file with commented examples.

```bash
workie init
workie init --force              # overwrite existing
workie init --output custom.yaml # custom filename
```

### Root flags

```bash
workie --list       # list worktrees
workie --version    # version info
workie --config f   # custom config path
workie --verbose    # verbose output
workie --quiet      # quiet mode
```

## Configuration

Place `.workie.yaml` in your repository root. Run `workie init` to generate a starter config.

```yaml
# Files/directories to copy into new worktrees
files_to_copy:
  - .env.example
  - scripts/
  - config/development.yaml

# Lifecycle hooks
hooks:
  post_create:
    - "npm install"
    - "make setup"
  pre_remove:
    - "npm run cleanup"
  timeout_minutes: 5

# Default issue provider
default_provider: github

# Issue providers
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

  linear:
    enabled: true
    settings:
      api_key_env: "LINEAR_API_KEY"
      team_id: "TEAM"
    branch_prefix:
      bug: "fix/"
      feature: "feat/"
      default: "issue/"

# Sprite VM setup
sprite:
  name: "my-dev-vm"
  org: "my-org"
  pre_setup:
    - "mkdir -p ~/go/bin"
  post_setup:
    - "which td && td --version"

# Autopilot settings
autopilot:
  backend: "claude"
  models:
    claude: "opus"
    codex: "gpt-5.3-codex"
    opencode: "openai/gpt-5.5"
  opencode:
    server_url: "http://127.0.0.1:4096"
    username_env: "OPENCODE_SERVER_USERNAME"
    password_env: "OPENCODE_SERVER_PASSWORD"
```

## How it works

1. Detects the repository root via `git rev-parse --show-toplevel`
2. Loads `.workie.yaml` from the repo root
3. Creates worktrees under `<repo>-worktrees/`
4. Copies configured files into the new worktree
5. Runs lifecycle hooks (`post_create` / `pre_remove`)

## License

MIT - see [LICENSE](LICENSE).
