---
type: Spec Story
title: commit-convention-and-hooks
description: Enforce Conventional Commits + local build/test git hooks
tags: [spec, dev-workflow]
timestamp: 2026-08-23T17:00:00+09:00
---

# Enforce Conventional Commits + local build/test git hooks

## Value to user
Developers get immediate feedback from local git hooks if a commit message violates Conventional Commits format, or if the build/tests are broken before committing or pushing. This catches issues locally without needing CI, and enables a future SemVer release tool to compute version bumps automatically from commit types.

## Completion criteria
- Every commit message's first line must match Conventional Commits format `<type>(<scope>)?!: <description>`, types: `feat|fix|docs|style|refactor|perf|test|chore`.
- Commit messages starting with `Merge` or `Revert` are exempt from the format check.
- `.githooks/commit-msg` enforces the above, rejecting non-matching, non-exempt messages with exit 1 and a usage example printed to stderr.
- `.githooks/pre-commit` and `.githooks/pre-push` both run `make check` (vet+test) and block the commit/push on non-zero exit.
- `make install-hooks` sets `git config core.hooksPath .githooks` for the current clone.

## Spec

### Conventional Commits format enforcement
- Commit message first line must match `^(feat|fix|docs|style|refactor|perf|test|chore)(\([a-z0-9-]+\))?!?: .+`
- Commits starting with `Merge` or `Revert` bypass the format check
- Non-matching messages are rejected with exit 1 and a helpful error message printed to stderr

### Hook installation
- Hooks are versioned in `.githooks/` directory (not `.git/hooks/`, which is local and untracked)
- `make install-hooks` runs `git config core.hooksPath .githooks` to enable hooks for the clone
- Each hook is a bash script with `#!/usr/bin/env bash` and `set -e`

### Pre-commit and pre-push behavior
- Both `.githooks/pre-commit` and `.githooks/pre-push` run `make check` (which runs `go vet ./...` + `go test ./...`)
- If `make check` exits non-zero, the hook exits non-zero and blocks the commit/push
- No special git plumbing (e.g., `git rev-parse`) — hooks just run `make check` in the current working directory

## Acceptance criteria

| AC | Category | Verification Method |
| -- | -- | -- |
| Given a commit message file containing `fix: exclude test files from dependency graph` - When `.githooks/commit-msg` runs against that file - Then it exits 0 | Normal | integration test: `hooks_test.go: TestCommitMsgHook_AcceptsConventionalFormat` |
| Given a commit message file containing `fixed the bug` (no type prefix) - When `.githooks/commit-msg` runs against it - Then it exits 1 and stderr contains "Conventional Commits" and an example line | Exception | integration test: `hooks_test.go: TestCommitMsgHook_RejectsNonConventionalFormat` |
| Given a commit message file containing `Merge branch 'feat/x' into main` - When `.githooks/commit-msg` runs against it - Then it exits 0 (exempt) | Boundary | integration test: `hooks_test.go: TestCommitMsgHook_ExemptsMergeCommits` |
| Given `.githooks/pre-commit` invoked with its working directory set to a temp dir containing a stub `Makefile` whose `check` target exits 0 - When the hook runs - Then it exits 0 | Normal | integration test: `hooks_test.go: TestPreCommitHook_PassesWhenCheckSucceeds` |
| Given `.githooks/pre-push` invoked with its working directory set to a temp dir containing a stub `Makefile` whose `check` target exits 1 - When the hook runs - Then it exits non-zero | Exception | integration test: `hooks_test.go: TestPrePushHook_FailsWhenCheckFails` |
| Given a fresh clone with default (unset) `core.hooksPath` - When `make install-hooks` is run - Then `git config core.hooksPath` reports `.githooks` | Normal | integration test: `hooks_test.go: TestInstallHooks_SetsHooksPath` |
