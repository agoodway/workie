## ADDED Requirements

### Requirement: Setup command runs post-create lifecycle on existing worktree

The `workie setup` command SHALL copy configured files from `.workie.yaml` into an existing worktree directory and execute `post_create` hooks in that directory. It SHALL NOT run `git worktree add` or create branches.

#### Scenario: Successful setup with explicit paths

- **WHEN** the user runs `workie setup feature/foo --source-tree-path <repo> --worktree-path <worktree>`
- **AND** `<worktree>` exists and is a directory
- **AND** `<repo>` is the git repository root matching the detected repo
- **THEN** configured files are copied into `<worktree>`
- **AND** `post_create` hooks run with `<worktree>` as the working directory
- **AND** success output includes branch name and worktree path
- **AND** success output does not describe the worktree as created

#### Scenario: Setup does not create a worktree or branch

- **WHEN** the user runs `workie setup` with valid existing `--source-tree-path` and `--worktree-path`
- **THEN** the command does not invoke `git worktree add`
- **AND** no new git branch is created

#### Scenario: Existing configured files are overwritten

- **WHEN** a configured destination file already exists in the worktree
- **THEN** setup overwrites it using the same copy behavior as `workie begin`
- **AND** setup continues to `post_create` hooks

#### Scenario: Copy warnings do not abort setup

- **WHEN** a configured source file is missing or a copy fails
- **THEN** setup prints a warning
- **AND** setup continues to `post_create` hooks (matching `workie begin`)

#### Scenario: Setup uses Codex environment variables

- **WHEN** the user runs `workie setup` without path flags
- **AND** `CODEX_SOURCE_TREE_PATH` and `CODEX_WORKTREE_PATH` are set
- **THEN** paths are resolved from those environment variables
- **AND** the post-create lifecycle runs as with explicit flags

#### Scenario: Flags override environment variables

- **WHEN** both `--source-tree-path` / `--worktree-path` flags and Codex env vars are set
- **THEN** flag values take precedence over environment variables

#### Scenario: Branch name derived from worktree HEAD

- **WHEN** no positional branch name is provided
- **AND** the worktree has a checked-out branch
- **THEN** the branch name is derived from the branch checked out in the resolved worktree path

### Requirement: Setup validates path inputs

The `workie setup` command SHALL reject invalid path combinations before copying files or running hooks.

#### Scenario: Missing required paths

- **WHEN** neither flags nor `CODEX_SOURCE_TREE_PATH` / `CODEX_WORKTREE_PATH` provide both paths
- **THEN** the command exits with an error describing required inputs

#### Scenario: Worktree path does not exist

- **WHEN** the resolved worktree path does not exist or is not a directory
- **THEN** the command exits with an error before copying files or running hooks

#### Scenario: Worktree path equals repo root

- **WHEN** the resolved worktree path equals the source tree (repo root)
- **THEN** the command exits with an error before copying files or running hooks

#### Scenario: Source tree mismatch

- **WHEN** `--source-tree-path` does not match the repository root detected from the source tree path
- **THEN** the command exits with an error before copying files or running hooks

#### Scenario: Worktree is not registered with source tree

- **WHEN** the resolved worktree path is not listed as a worktree of the source tree by `git worktree list`
- **THEN** the command exits with an error before copying files or running hooks

#### Scenario: Branch name mismatch

- **WHEN** a positional branch name is provided
- **AND** it does not match the branch checked out in the resolved worktree path
- **THEN** the command exits with an error before copying files or running hooks

#### Scenario: Detached HEAD is rejected

- **WHEN** the resolved worktree has a detached HEAD
- **THEN** the command exits with an error before copying files or running hooks
- **AND** the error states that a branch must be checked out

### Requirement: Setup preserves hook failure semantics

The `workie setup` command SHALL use the same hook failure behavior as `workie begin`: hook failures warn but do not abort setup, including when all hooks fail per `ExecuteHooks`. Hook warnings SHALL NOT say the worktree was created.

#### Scenario: Partial hook failure

- **WHEN** some `post_create` hooks fail but not all
- **THEN** setup completes with a warning
- **AND** files that were copied remain in the worktree

#### Scenario: All hooks fail

- **WHEN** all `post_create` hooks fail
- **THEN** setup completes with a warning
- **AND** files that were copied remain in the worktree

### Requirement: Setup does not support issue-based branch creation

The `workie setup` command SHALL NOT accept `--issue`.

#### Scenario: No issue flag on setup

- **WHEN** the user inspects `workie setup --help`
- **THEN** no `--issue` flag is listed
