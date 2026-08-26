## 1. Manager Refactor

- [x] 1.1 Extract `SetupExistingWorktree(branchName, worktreePath string) error` in `manager/manager.go` with file copy and `post_create` hooks only. Do not move begin's "created worktree" success output, quiet-path print, or next-steps. Preserve `copyConfiguredFiles` warn-and-continue (do not fail the command on copy warnings). Hook warnings MUST NOT say the worktree was created.
- [x] 1.2 Update `CreateOrReuseWorktree` to delegate to `SetupExistingWorktree` after `git worktree add`, then print the existing created-worktree success block
- [x] 1.3 Add unit tests for `SetupExistingWorktree` in `manager/manager_test.go`, including overwrite of existing configured destination files

## 2. Finish Refactor

- [x] 2.1 Extract `prepareWorktreeCleanup(wm, branchName, worktreePath string, force bool) error` in `cmd/finish.go` with finish's current preflight only: non-empty branch, path existence, `pre_remove` hooks, and dirty-worktree check. Do not add HEAD matching, worktree-membership, or Codex path resolution. Hook-failure continuation wording and dirty-error wrapping MUST be caller-specific so `finish` keeps its current removal messages.
- [x] 2.2 Update `finishWorktree` to construct `filepath.Join(wm.WorktreesDir, branchName)`, call `prepareWorktreeCleanup`, then continue with `git worktree remove` and optional branch pruning
- [x] 2.3 Add tests for `prepareWorktreeCleanup` behavior (dirty check, force bypass, hook warnings) without changing `finish` removal messages

## 3. Codex Path Resolver

- [x] 3.1 Add `cmd/codex_paths.go` with shared path resolution (flags → `CODEX_SOURCE_TREE_PATH` / `CODEX_WORKTREE_PATH` → error). Do not `MarkFlagRequired` on path flags.
- [x] 3.2 Implement validation: resolve source tree with `DetectGitRepositoryAt(sourceTreePath)` then `LoadConfig` (do not detect from cwd, do not call `CreateWorktreesDirectory` or `Run`); canonicalize with `filepath.Abs` and `filepath.EvalSymlinks` before comparing to `wm.RepoPath`; worktree exists and is a directory; membership via `git worktree list --porcelain` from the source tree (not `pathMatchesWorktree`); is not repo root; reject detached HEAD (`rev-parse --abbrev-ref HEAD` empty or `HEAD`); branch name matches or is derived from the worktree's checked-out git branch
- [x] 3.3 Add unit tests for path resolver (flag precedence, env fallback, symlink canonicalization, membership, detached HEAD, validation errors)

## 4. Setup Command

- [x] 4.1 Add `cmd/setup.go` with `workie setup [branch-name]`, `--source-tree-path`, `--worktree-path` flags, `cobra.MaximumNArgs(1)`, and a setup-specific success line (branch name + worktree path, not "created")
- [x] 4.2 Wire command to construct manager options, resolve Codex paths via `cmd/codex_paths.go` (resolver calls `DetectGitRepositoryAt(sourceTreePath)` then `LoadConfig`), then call `SetupExistingWorktree`
- [x] 4.3 Register command in `cmd/root.go` and update help examples

## 5. Cleanup Command

- [x] 5.1 Add `cmd/cleanup.go` with `workie cleanup [branch-name]`, `--source-tree-path`, `--worktree-path`, `--force` flags, `cobra.MaximumNArgs(1)`, and a dedicated force variable (do not reuse `forceFinish`)
- [x] 5.2 Wire command to construct manager options, resolve Codex paths via `cmd/codex_paths.go` (resolver calls `DetectGitRepositoryAt(sourceTreePath)` then `LoadConfig`), then call `prepareWorktreeCleanup`. Dirty-error and hook-continuation messages MUST NOT imply worktree removal.
- [x] 5.3 Register command in `cmd/root.go` and update help examples

## 6. Documentation and Verification

- [x] 6.1 Update `README.md` and `USAGE.md` with `workie setup` and `workie cleanup` usage, flags, and Codex env vars. Rewrite USAGE.md basic usage to current `begin`/`finish` commands (it still documents `workie remove`). Distinguish `setup` from existing `setup-sprite`.
- [x] 6.2 Run `go test ./...` and fix any failures
- [x] 6.3 Regression check: `workie begin` and `workie finish` behavior unchanged
- [x] 6.4 Add automated command-level tests under `cmd/` (temp git repo helper; `cmd/` has no tests today and `manager.initTestRepo` is unexported) for `begin`/`finish` unchanged behavior and setup/cleanup flags, help output, env precedence, hook ordering, dirty-worktree handling, copy overwrite, detached HEAD rejection, and that cleanup does not remove the worktree
- [x] 6.5 Manual Codex flow: external worktree + `workie setup` copies files and runs hooks
- [x] 6.6 Manual Codex flow: `workie cleanup` runs hooks, blocks on dirty worktree, respects `--force`, does not remove worktree
