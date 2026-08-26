## Why

Codex and other external agents create git worktrees themselves, but still need Workie's post-create and pre-remove lifecycle (file copying, hook execution, validation) without duplicating `begin`/`finish` git operations. Today there is no CLI command to run those lifecycle steps against an existing worktree path.

## What Changes

- Add `workie setup [branch-name]` to copy configured files and run `post_create` hooks on an existing worktree
- Add `workie cleanup [branch-name]` to run `pre_remove` hooks and dirty-worktree checks on an existing worktree without removing it
- Accept `--source-tree-path` and `--worktree-path` flags with `CODEX_SOURCE_TREE_PATH` and `CODEX_WORKTREE_PATH` environment fallbacks
- Extract shared lifecycle helpers from `manager.CreateOrReuseWorktree` and `cmd/finish.go` so `begin`/`finish` behavior stays unchanged
- Leave `begin` and `finish` UX unchanged; no issue-provider support on `setup`; no branch pruning or worktree removal on `cleanup`

## Capabilities

### New Capabilities

- `workie-setup`: Post-create lifecycle for externally managed worktrees (file copy, hooks, path validation)
- `workie-cleanup`: Pre-remove lifecycle for externally managed worktrees (hooks, dirty check, no removal)

### Modified Capabilities

- _(none — no existing specs in `openspec/specs/`)_

## Impact

- **Code**: `cmd/setup.go`, `cmd/cleanup.go`, `cmd/codex_paths.go`, `cmd/finish.go`, `cmd/root.go`, `manager/manager.go`
- **CLI**: Two new subcommands; README/USAGE/help updates
- **Tests**: New unit tests for path resolution, validation, and extracted helpers; command-level tests under `cmd/`; regression on `begin`/`finish`
- **Consumers**: Codex-managed worktree flows; any agent that creates worktrees externally and delegates setup/cleanup to Workie
