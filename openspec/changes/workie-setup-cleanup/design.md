## Context

Workie's `begin` command runs `git worktree add`, copies `files_to_copy` from `.workie.yaml`, and executes `post_create` hooks. `finish` runs `pre_remove` hooks, checks for uncommitted changes, then runs `git worktree remove` and optionally deletes the branch.

Codex creates worktrees externally and sets `CODEX_SOURCE_TREE_PATH` / `CODEX_WORKTREE_PATH` so a follow-up tool can configure the environment. Workie needs thin commands that perform only the lifecycle portions, leaving git worktree add/remove to the caller.

Current implementation keeps post-create logic inline in `manager.CreateOrReuseWorktree` and pre-remove logic inline in `cmd/finish.go`. Both must be extracted without changing existing command behavior.

## Goals / Non-Goals

**Goals:**

- Provide `workie setup` and `workie cleanup` commands with Codex-compatible path inputs
- Reuse existing file-copy, hook execution, and validation logic via extracted helpers
- Preserve `begin`/`finish` behavior exactly (regression-safe refactor)
- Keep Codex-specific path resolution in `cmd/`, not `manager/`

**Non-Goals:**

- Changing `begin` or `finish` UX or flags
- Adding `--issue` support to `setup`
- Branch pruning or `git worktree remove` in `cleanup`
- New global config keys or hook types
- Supporting worktree creation or removal in the new commands

## Decisions

### 1. Extract `SetupExistingWorktree` in `manager/manager.go`

Move post-create steps (copy configured files and run `post_create` hooks) from `CreateOrReuseWorktree` into `SetupExistingWorktree(branchName, worktreePath string) error`. Do **not** move begin's created-worktree success block, quiet-path print, or "To start working" next-steps; those stay in `CreateOrReuseWorktree` so `begin` wording is unchanged.

`CreateOrReuseWorktree` calls the helper after `git worktree add` succeeds, then prints its existing success output. `copyConfiguredFiles` currently warns and returns nil on missing sources or copy failures; `SetupExistingWorktree` MUST preserve that and MUST NOT fail the command on copy warnings.

Hook-failure warnings in the helper MUST be operation-neutral (they MUST NOT say the worktree was created). The `setup` command prints a setup-specific success line that includes branch name and worktree path and MUST NOT describe the worktree as created.

**Rationale:** Single source of truth for copy+hooks; `setup` would otherwise claim it created a worktree.

**Alternative considered:** Duplicate logic in `cmd/setup.go` — rejected to avoid drift. Moving begin's success stdout into the helper — rejected because that stdout says "Successfully created worktree".

### 2. Extract `prepareWorktreeCleanup` in `cmd/finish.go`

Move **finish's current** pre-remove preflight into `prepareWorktreeCleanup(wm, branchName, worktreePath string, force bool) error`:

- non-empty branch name
- path existence (`os.Stat`)
- `pre_remove` hooks (warn and continue)
- dirty-worktree check with the `force` parameter

The helper MUST NOT add HEAD matching, worktree-membership, repo-root rejection, or Codex path resolution. Those belong in `cmd/codex_paths.go`. Putting them in the helper would change `finish`, which today only checks a non-empty name and `filepath.Join(wm.WorktreesDir, branchName)` existence (`cmd/finish.go`).

`finishWorktree` still constructs that `WorktreesDir` path, calls the helper, then continues with `git worktree remove` and optional branch pruning. `cleanup` resolves Codex paths first, then calls the helper with the resolved path.

Hook-failure continuation wording and dirty-error wrapping are **caller-specific** so `finish` keeps its current messages ("worktree removal will continue", "worktree removal blocked", "--force to remove anyway"). `cleanup` MUST print non-removal equivalents: hook failures continue cleanup; `--force` only skips the dirty check and MUST NOT imply `git worktree remove` or data loss from removal.

**Rationale:** `cleanup` shares pre-remove behavior without removal; keeping the helper in `cmd/` matches current `finish` structure; HEAD/membership checks stay out so `finish` UX is unchanged.

**Alternative considered:** Move to `manager` package — rejected; dirty-check and finish-specific flags are cmd-layer concerns today. Folding HEAD matching into the helper — rejected because it would change `finish`.

### 3. Shared path resolver in `cmd/codex_paths.go`

A helper in `cmd/codex_paths.go` (not a similarly named file) resolves paths with precedence: CLI flags → env vars → error. Do not `MarkFlagRequired` on the path flags; env fallback is resolved in code. `setup` and `cleanup` use `cobra.MaximumNArgs(1)` so the branch name may be omitted. Responsibilities:

- Canonicalize user paths and `wm.RepoPath` with `filepath.Abs` and `filepath.EvalSymlinks` before equality checks (git may return the symlink-resolved path; `Abs`/`Clean` alone false-fails on macOS `/var` vs `/private/var`)
- Resolve `source-tree-path` first and initialize the manager with `DetectGitRepositoryAt(sourceTreePath)` so external worktree cwd does not change repo detection
- Call `LoadConfig` after detection; do **not** call `CreateWorktreesDirectory` or `WorktreeManager.Run` (creating the sibling `-worktrees` directory would be an unspecified side effect)
- Validate `source-tree-path` matches `wm.RepoPath` after that canonicalization (catches a subdirectory passed as the source tree)
- Validate `worktree-path` exists and is a directory
- Validate membership: from the detected source `RepoPath`, parse `git worktree list --porcelain` and require the canonical worktree path to appear as a `worktree` entry. Do **not** use `pathMatchesWorktree` as the belonging check (it only compares `rev-parse --abbrev-ref HEAD` and treats git errors as a non-match)
- Reject a worktree path that equals the repo root
- Resolve the worktree's checked-out branch with `git -C <worktree-path> rev-parse --abbrev-ref HEAD`
- If the result is empty or `HEAD` (detached HEAD), error and ask the caller to check out a branch; do not derive `branchName` `"HEAD"` and do not treat an explicit `"HEAD"` as a successful match
- Derive `branchName` from the positional arg or the worktree's checked-out branch
- If branch name is provided, it must match the worktree's checked-out branch

**Rationale:** Codex env handling stays out of `manager`; both commands share validation.

### 4. Hook failure behavior unchanged

Hook failures warn but do not abort, including when `ExecuteHooks` returns an all-hooks-failed error. Matches current `begin`/`finish` behavior, which catches hook errors and continues.

### 5. Setup copy overwrite behavior unchanged

`workie setup` uses the same configured-file copy behavior as `workie begin`; existing destination files may be overwritten by configured source files. This preserves current lifecycle semantics and avoids adding a new conflict mode.

### 6. `cleanup` supports `--force` only for dirty-worktree bypass

Same semantics as `finish --force` for the status check; does not imply worktree removal. Bind cleanup's flag to a dedicated variable (do not reuse `forceFinish`); `executeWorktreeRemove` still reads that global, so sharing it would make a future in-process mix-up force-remove.

## Risks / Trade-offs

- **[Risk] Branch name vs worktree HEAD mismatch** → Validate explicit branch arg against the worktree's checked-out branch; derive from HEAD when omitted; reject detached HEAD
- **[Risk] Existing files in externally created worktree** → Preserve current copy behavior by allowing configured files to overwrite destination files
- **[Risk] Refactor breaks `begin`/`finish`** → Extract-only refactor with existing tests; keep begin success stdout and finish removal messages in the callers; add regression checklist in tasks
- **[Risk] Codex env vars absent** → Clear error messages listing required flags/env vars
- **[Risk] Unrelated clone on the same branch** → Membership is `git worktree list --porcelain` from the source tree, not a HEAD-name comparison
- **[Trade-off] `cleanup` does not remove worktree** → Callers must handle removal; intentional per Codex contract

## Migration Plan

1. Ship new commands alongside existing ones (no breaking changes)
2. Update README.md and USAGE.md with `setup`/`cleanup` usage, flags, and Codex env var docs; distinguish `setup` from existing `setup-sprite`
3. Codex workflows switch from manual script invocation to `workie setup` / `workie cleanup`

Rollback: new commands are additive; revert by ignoring them; `begin`/`finish` unchanged.

## Open Questions

- None — plan from `.opencode/plans/1776384636344-sunny-meadow.md` is complete and scoped.
