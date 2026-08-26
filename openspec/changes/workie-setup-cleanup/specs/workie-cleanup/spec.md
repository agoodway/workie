## ADDED Requirements

### Requirement: Cleanup command runs pre-remove lifecycle without removal

The `workie cleanup` command SHALL execute `pre_remove` hooks and dirty-worktree checks on an existing worktree directory. It SHALL NOT run `git worktree remove` or delete branches.

#### Scenario: Successful cleanup with explicit paths

- **WHEN** the user runs `workie cleanup feature/foo --source-tree-path <repo> --worktree-path <worktree>`
- **AND** `<worktree>` exists and is a directory
- **AND** the worktree has no uncommitted changes
- **THEN** `pre_remove` hooks run with `<worktree>` as the working directory
- **AND** the worktree directory still exists after the command completes

#### Scenario: Cleanup uses Codex environment variables

- **WHEN** the user runs `workie cleanup` without path flags
- **AND** `CODEX_SOURCE_TREE_PATH` and `CODEX_WORKTREE_PATH` are set
- **THEN** paths are resolved from those environment variables
- **AND** the pre-remove lifecycle runs as with explicit flags

#### Scenario: Flags override environment variables

- **WHEN** both `--source-tree-path` / `--worktree-path` flags and Codex env vars are set
- **THEN** flag values take precedence over environment variables

#### Scenario: Branch name derived from worktree HEAD

- **WHEN** no positional branch name is provided
- **AND** the worktree has a checked-out branch
- **THEN** the branch name is derived from the branch checked out in the resolved worktree path

### Requirement: Cleanup validates path inputs

The `workie cleanup` command SHALL reject invalid path combinations before running hooks. It SHALL require both a source tree path and a worktree path from `--source-tree-path` / `--worktree-path` or `CODEX_SOURCE_TREE_PATH` / `CODEX_WORKTREE_PATH`. It SHALL reject a worktree path that does not exist, is not a directory, equals the source tree (repo root), is not listed as a worktree of the source tree by `git worktree list`, or whose checked-out branch does not match an explicit positional branch name. When no positional branch name is provided, it SHALL derive the branch name from the worktree's checked-out branch. It SHALL reject a worktree with a detached HEAD.

#### Scenario: Missing required paths

- **WHEN** neither `--source-tree-path` / `--worktree-path` nor `CODEX_SOURCE_TREE_PATH` / `CODEX_WORKTREE_PATH` provide both paths
- **THEN** the command exits with an error describing required inputs

#### Scenario: Worktree path does not exist

- **WHEN** the resolved worktree path does not exist or is not a directory
- **THEN** the command exits with an error before running hooks

#### Scenario: Worktree path equals repo root

- **WHEN** the resolved worktree path equals the source tree (repo root)
- **THEN** the command exits with an error before running hooks

#### Scenario: Source tree mismatch

- **WHEN** `--source-tree-path` does not match the repository root detected from the source tree path
- **THEN** the command exits with an error before running hooks

#### Scenario: Worktree is not registered with source tree

- **WHEN** the resolved worktree path is not listed as a worktree of the source tree by `git worktree list`
- **THEN** the command exits with an error before running hooks

#### Scenario: Branch name mismatch

- **WHEN** a positional branch name is provided
- **AND** it does not match the branch checked out in the resolved worktree path
- **THEN** the command exits with an error before running hooks

#### Scenario: Detached HEAD is rejected

- **WHEN** the resolved worktree has a detached HEAD
- **THEN** the command exits with an error before running hooks
- **AND** the error states that a branch must be checked out

### Requirement: Cleanup blocks on uncommitted changes unless forced

The `workie cleanup` command SHALL check for uncommitted changes in the worktree using the same logic as `workie finish`. It SHALL block when changes exist unless `--force` is passed. Error text SHALL NOT imply `git worktree remove` or data loss from removal; `--force` SHALL only skip the dirty-worktree check.

#### Scenario: Uncommitted changes without force

- **WHEN** the worktree has uncommitted changes
- **AND** `--force` is not set
- **THEN** the command exits with an error that does not mention worktree removal
- **AND** `pre_remove` hooks have already run (matching `finish` ordering)

#### Scenario: Uncommitted changes with force

- **WHEN** the worktree has uncommitted changes
- **AND** `--force` is set
- **THEN** the command completes without error
- **AND** the worktree is not removed

### Requirement: Cleanup preserves hook failure semantics

The `workie cleanup` command SHALL use the same hook failure behavior as `workie finish`: hook failures warn but do not abort cleanup, including when all hooks fail per `ExecuteHooks`. Hook warnings SHALL NOT say that worktree removal will continue.

#### Scenario: Partial hook failure

- **WHEN** some `pre_remove` hooks fail but not all
- **THEN** cleanup continues to the dirty-worktree check
- **AND** a warning is printed

#### Scenario: All hooks fail

- **WHEN** all `pre_remove` hooks fail
- **THEN** cleanup continues to the dirty-worktree check
- **AND** a warning is printed

### Requirement: Cleanup does not prune branches

The `workie cleanup` command SHALL NOT accept `--prune-branch` and SHALL NOT delete git branches.

#### Scenario: No prune flag on cleanup

- **WHEN** the user inspects `workie cleanup --help`
- **THEN** no `--prune-branch` flag is listed
- **AND** no branch deletion occurs during cleanup
