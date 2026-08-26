# Inspector Review — workie-setup-cleanup

**Reviewed:** 2026-07-01
**Reviewer:** inspector/review-update
**Verdict:** Ready to implement

## Summary

This change proposes additive `workie setup` and `workie cleanup` commands for externally managed worktrees, extracting existing setup/cleanup lifecycle behavior while preserving `begin` and `finish`. The initial artifacts were structurally sound but had several codebase-alignment issues around hook error semantics, cwd-sensitive repository detection, slash-containing branch names, and validation coverage. All findings were patched, including two decisions resolved with user guidance.

**Counts:** Critical: 3 · Warning: 4 · Suggestion: 1

## Scope inspected

- Proposal: `openspec/changes/workie-setup-cleanup/proposal.md`
- Design: `openspec/changes/workie-setup-cleanup/design.md`
- Tasks: `openspec/changes/workie-setup-cleanup/tasks.md`
- Deltas:
  - `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md`
  - `openspec/changes/workie-setup-cleanup/specs/workie-cleanup/spec.md`
- Canonical specs consulted: none; no `openspec/specs/**/spec.md` files exist
- Other active changes consulted: none; no other active change proposals were found

## Critical

_None._

## Warning

_None._

## Suggestion

_None._

## Patches applied

8 findings were patched. 6 findings were auto-patched. 2 findings were patched after user guidance. 0 findings were skipped.

### Auto-patched

1. **Hook failure semantics misstated** — `openspec/changes/workie-setup-cleanup/design.md:58` → Updated design and setup/cleanup deltas to preserve current `begin`/`finish` behavior where hook errors warn and continue, including all-hooks-failed errors from `ExecuteHooks`.
2. **Source-tree detection was cwd-sensitive** — `openspec/changes/workie-setup-cleanup/design.md:49` → Updated path resolver design and tasks to resolve source tree first with `DetectGitRepositoryAt(sourceTreePath)` and compare canonical paths.
3. **Cleanup validation coverage incomplete** — `openspec/changes/workie-setup-cleanup/specs/workie-cleanup/spec.md:29` → Added cleanup scenarios for repo-root rejection, source-tree mismatch, non-registered worktree path, and branch derivation.
4. **Setup validation missed registered-worktree requirement** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:36` → Added setup scenario requiring the worktree path to be a Git worktree belonging to the source tree before copying files or running hooks.
5. **Reuse-path task contradicted unchanged behavior** — `openspec/changes/workie-setup-cleanup/tasks.md:4` → Removed “or reuse path” so `CreateOrReuseWorktree` delegates only after `git worktree add`, matching current reuse behavior in `manager/manager.go:409`.
6. **Docs impact omitted USAGE.md and automated regression scope** — `openspec/changes/workie-setup-cleanup/proposal.md:26`, `openspec/changes/workie-setup-cleanup/tasks.md:33` → Added `USAGE.md` to docs impact/tasks and added automated command-level regression tests for `begin`/`finish`, setup/cleanup flags/help/env precedence, hook ordering, and dirty-worktree handling.

### User-guided patches

1. **Branch-name validation conflicted with slash-containing names** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:58` → Updated design and both deltas to validate explicit branch names against the worktree's checked-out git branch and derive from git HEAD when omitted. User chose: use git HEAD.
2. **Existing-file copy semantics were unspecified** — `openspec/changes/workie-setup-cleanup/specs/workie-setup/spec.md:5` → Added design and setup scenario stating setup preserves `begin` copy behavior and may overwrite existing configured destination files. User chose: overwrite existing.

### Skipped

_None._

## Alignment notes

- **Other active changes:** No conflicts found; no other active change proposals were present.
- **Canonical specs:** No canonical specs exist yet for `workie-setup` or `workie-cleanup`.
- **Codebase assumptions verified:** `manager.CreateOrReuseWorktree` returns early on matching existing worktrees before copy/hooks (`manager/manager.go:409`); `ExecuteHooks` returns an error only when all hooks fail (`manager/manager.go:690`), while begin/finish catch hook errors and continue (`manager/manager.go:464`, `cmd/finish.go:100`); `cmd/finish.go` currently uses cwd-based `DetectGitRepository()` (`cmd/finish.go:62`); current file copy uses `os.Create`, which overwrites destination files (`manager/manager.go:232`); `USAGE.md` exists and contains command documentation (`USAGE.md:49`).

## What looks good

- The change is additive and explicitly preserves existing `begin`/`finish` UX.
- New setup and cleanup responsibilities are separated cleanly between manager lifecycle reuse and cmd-level Codex path resolution.
- The updated specs now cover flag/env precedence, invalid paths, branch validation, hook failure behavior, dirty-worktree behavior, and non-removal/non-pruning guarantees.
- OpenSpec validation passes with `openspec validate workie-setup-cleanup --strict`.
