# Inspector Review-Update — workie-setup-cleanup

**Reviewed:** 2026-08-26
**Reviewer:** inspector-review-update
**Mode:** auto
**Verdict:** Ready to implement

## Summary

This change adds `workie setup` and `workie cleanup` for externally managed worktrees, extracting copy/hooks and pre-remove preflight from `begin`/`finish` without changing those commands. The artifacts were already in good shape from a prior 2026-07-01 review; this pass caught remaining implementer traps (cwd bootstrap vs `DetectGitRepositoryAt`, "created/removed" wording leaking into the new commands, membership/symlink/detached-HEAD validation). All findings were patched; none remain.

**Original counts:** Critical: 0 · Warning: 12 · Suggestion: 6
**Patches:** Auto-patched: 12 · User-guided: 0 · Model-recommended: 6 · Skipped: 0

## Scope inspected

- Proposal: `openspec/changes/workie-setup-cleanup/proposal.md`
- Design: `openspec/changes/workie-setup-cleanup/design.md`
- Tasks: `openspec/changes/workie-setup-cleanup/tasks.md`
- Deltas:
  - `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md`
  - `openspec/changes/workie-setup-cleanup/specs/workie-cleanup/spec.md`
- Canonical specs consulted: none; `openspec/specs/` is empty
- Other active changes consulted: none; `workie-setup-cleanup` is the only unarchived change

## Patches applied

12 findings were auto-patched. 0 were patched after user guidance. 6 were patched from model recommendations. 0 were skipped.

### Auto-patched

1. **Impact list omitted `cmd/codex_paths.go`** — `openspec/changes/workie-setup-cleanup/proposal.md:26` → Added `cmd/codex_paths.go` to Code impact and command-level tests to Tests impact. Locked design Decision 3 to that filename (dropped "or similar").
2. **Migration plan skipped USAGE.md** — `openspec/changes/workie-setup-cleanup/design.md:81` → Migration step 2 now updates README.md and USAGE.md, and distinguishes `setup` from existing `setup-sprite`.
3. **Tasks 4.2/5.2 implied cwd bootstrap** — `openspec/changes/workie-setup-cleanup/tasks.md:22` → Rewrote 4.2/5.2 to construct manager options, resolve Codex paths (`DetectGitRepositoryAt` then `LoadConfig`), and not detect from cwd or call `CreateWorktreesDirectory`/`Run`.
4. **Setup SHALL NOT create, but no scenario** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:5` → Added scenario that setup does not invoke `git worktree add` and does not create a branch.
5. **Path compare used Abs/Clean only** — `openspec/changes/workie-setup-cleanup/design.md:48` → Canonicalization now includes `filepath.EvalSymlinks` to match `DetectGitRepositoryAt` tests on macOS.
6. **Copy failures vs begin semantics unspecified** — `manager/manager.go:384` → Design/task 1.1 and a setup scenario now require `copyConfiguredFiles` warn-and-continue (do not fail setup on copy warnings).
7. **Finish `ExactArgs(1)` / shared `forceFinish` trap** — `cmd/finish.go:49` → Tasks 4.1/5.1 require `cobra.MaximumNArgs(1)`, no `MarkFlagRequired` on path flags, and a dedicated cleanup force variable.
8. **`--issue` "or any issue-provider flags" untestable** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:92` → Narrowed the SHALL and help scenario to `--issue` only.
9. **HEAD derivation lived under a rejection requirement** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:69` → Moved derivation scenarios under the main setup/cleanup lifecycle requirements.
10. **Overwrite behavior had no task** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:16` → Task 1.3 (and 6.4) now assert overwrite of existing configured destination files.
11. **No cmd-level test helper guidance** — `openspec/changes/workie-setup-cleanup/tasks.md:36` → Task 6.4 requires `cmd/` tests with a temp git-repo helper (`initTestRepo` is unexported).
12. **USAGE.md still documents `workie remove`** — `USAGE.md:59` → Task 6.1 requires rewriting USAGE basic usage to `begin`/`finish` before adding setup/cleanup.

### Model-recommended patches

1. **`prepareWorktreeCleanup` vs path-resolver layer split** — `openspec/changes/workie-setup-cleanup/design.md:38`
   - **Chose:** Path resolver owns Codex validation (flags/env, `DetectGitRepositoryAt`, membership, not repo root, derive/match HEAD). `prepareWorktreeCleanup` keeps finish's current preflight only: non-empty branch, path exists, `pre_remove` hooks, dirty check plus force. `finishWorktree` still joins `WorktreesDir` itself.
   - **Rationale:** Finish today does not check HEAD or worktree membership (`cmd/finish.go:82-93`). Folding those checks into the helper would change `finish` and violate the preserve-exactly non-goal.

2. **Begin "created worktree" success stdout must not move** — `openspec/changes/workie-setup-cleanup/design.md:30`
   - **Chose:** Extract only `copyConfiguredFiles` and `post_create` hook execution into `SetupExistingWorktree`. Leave begin's created-worktree success block in `CreateOrReuseWorktree`. `cmd/setup` prints a setup-specific success line with branch and path.
   - **Rationale:** Setup SHALL NOT create a worktree (`specs/workie-setup/spec.md:5`); reusing `manager/manager.go:477` would print "Successfully created worktree". Spec only requires branch name and path in success output.

3. **Worktree membership algorithm unspecified** — `openspec/changes/workie-setup-cleanup/design.md:51`
   - **Chose:** From the detected source `RepoPath`, parse `git worktree list --porcelain` and require the canonical worktree path to appear as a `worktree` entry. Do not use `pathMatchesWorktree`.
   - **Rationale:** Workie does not register worktrees itself. `pathMatchesWorktree` only compares `rev-parse --abbrev-ref HEAD` and treats git errors as a non-match, so it would accept an unrelated clone on the same branch.

4. **Hook/dirty messages would imply creation or removal** — `openspec/changes/workie-setup-cleanup/design.md:38`
   - **Chose:** Caller-specific hook-continuation and dirty-error wrapping. Finish keeps "worktree removal will continue" / "worktree removal blocked". Cleanup prints non-removal equivalents; `--force` only skips the dirty check.
   - **Rationale:** Specialist's shared-neutral-warning option would change finish wording. Caller-specific messages preserve `begin`/`finish` exactly while stopping cleanup from claiming removal.

5. **Detached HEAD would become branch name `HEAD`** — `openspec/changes/workie-setup-cleanup/design.md:52`
   - **Chose:** After `rev-parse --abbrev-ref HEAD`, if the result is empty or `HEAD`, error and require a checked-out branch. Do not derive `branchName` `"HEAD"` and do not treat explicit `"HEAD"` as a match. Added scenarios to both deltas.
   - **Rationale:** Spec derivation assumes a real checked-out branch. Detached HEAD is an error case, not a product fork, and matches "branch name" as a git branch.

6. **Cleanup validation coupled to setup by prose** — `openspec/changes/workie-setup-cleanup/specs/workie-cleanup/spec.md:29`
   - **Chose:** Replace "same path validation rules as `workie setup`" with an explicit SHALL listing the rules; keep scenarios; name flags and `CODEX_*` vars like setup.
   - **Rationale:** After archive the two capabilities must stand alone. Shared implementation stays in `cmd/codex_paths.go`, not in spec cross-references with drifted wording.

### Skipped

_None._

## Critical

_None._

## Warning

_None._

## Suggestion

_None._

## Alignment notes

- **Other active changes:** No conflicts; this is the only unarchived change.
- **Canonical specs:** No canonical `workie-setup` or `workie-cleanup` specs exist yet; both deltas are ADDED.
- **Codebase assumptions verified:** `DetectGitRepositoryAt` exists (`manager/manager.go:60`) and tests compare via `EvalSymlinks` (`manager/manager_test.go:17`); `CreateOrReuseWorktree` returns early on reuse without copy/hooks (`manager/manager.go:409`); `ExecuteHooks` errors only when all hooks fail (`manager/manager.go:690`) and begin/finish warn-and-continue (`manager/manager.go:466`, `cmd/finish.go:100`); `copyConfiguredFiles` always returns nil (`manager/manager.go:384`); copy overwrite is `os.Create` (`manager/manager.go:232`); finish constructs `WorktreesDir/branchName` (`cmd/finish.go:88`) and uses `cobra.ExactArgs(1)` (`cmd/finish.go:49`); `forceFinish` is a package global used by `executeWorktreeRemove` (`cmd/finish.go:15,185`); `setup-sprite` already exists (`cmd/setup_sprite.go:21`); `cmd/` has no tests; USAGE.md still documents `workie remove` (`USAGE.md:65`).

## What looks good

- Additive commands with an explicit preserve-`begin`/`finish` non-goal.
- Codex path handling stays in `cmd/`, not `manager/`.
- Hook failure, copy overwrite, and flag-over-env precedence already specified.
- Cleanup non-removal and no `--prune-branch` are testable help/behavior scenarios.
- Prior inspector-review patches (cwd-safe `DetectGitRepositoryAt`, HEAD-based branch matching, hook warn-and-continue) remain intact.
