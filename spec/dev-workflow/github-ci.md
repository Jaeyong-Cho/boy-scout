---
type: Spec Story
title: github-ci
description: GitHub remote + Actions CI for boy-scout
tags: [spec, dev-workflow]
timestamp: 2026-08-23T17:00:00+09:00
---

# GitHub remote + Actions CI for boy-scout

## Value to user
boy-scout now has a remote repository on GitHub and a CI workflow that automatically tests every push to main and every pull request against main, so a broken build or failing test cannot land on main even from a clone without local hooks configured (e.g., via `--no-verify` or a fresh clone with no hooks installed).

## Completion criteria
- A private GitHub repository `Jaeyong-Cho/boy-scout` exists with `main` pushed as `origin/main`.
- A `.github/workflows/ci.yml` file runs `make check` (which runs `go vet ./...` + `go test ./...`) on every push to `main` and every pull request targeting `main`, using `actions/setup-go@v5` pinned to Go 1.24.
- A commit with a failing test or vet error produces a failed (red) status on that commit's Actions run.
- A workflow run for a PR with a failing test shows conclusion `failure` and is visible via `gh run list --branch <pr-branch>`.

## Spec

### Repository setup
- Repository: `Jaeyong-Cho/boy-scout`, private (can be made public later without risk; making it private later requires manual deletion and recreation).
- Remote origin: `https://github.com/Jaeyong-Cho/boy-scout.git`, already added to local clone.
- Main branch pushed as `origin/main`.

### Workflow file
- Location: `.github/workflows/ci.yml`
- Triggers: `on: push: branches: [main]` and `pull_request: branches: [main]`
- Job: `check` runs on `ubuntu-latest`
- Steps:
  - `actions/checkout@v4`
  - `actions/setup-go@v5` with `go-version: '1.24'` (must match `go.mod`'s `go 1.24` exactly)
  - `run: make check` (runs `go vet ./...` + `go test ./...`)
- Conclusion: `success` if all steps pass, `failure` if any step exits non-zero.

### CI independence
- This workflow is a second, independent safety net on top of local pre-commit/pre-push hooks (Story 1).
- Catches unconfigured clones (without hooks installed) or `--no-verify` pushes that bypass local hooks.

## Acceptance criteria

| AC | Category | Verification Method |
| -- | -- | -- |
| Given `Jaeyong-Cho/boy-scout` (private) with `main` pushed and `.github/workflows/ci.yml` committed - When a commit lands on `main` - Then `gh run list --branch main --limit 1 --json conclusion --jq '.[0].conclusion'` reports `success` | Normal | e2e: `gh run list` after a real push |
| Given an open PR branch pushed to the same repo - When new commits land on that branch - Then a workflow run is triggered for that PR, visible via `gh run list --branch <pr-branch>` | Normal | e2e: `gh run list --branch <pr-branch>` after a real push |
| Given a commit that deliberately breaks a test, pushed to a throwaway branch with a PR open against `main` - When the workflow runs - Then `gh run list --branch <pr-branch> --limit 1 --json conclusion --jq '.[0].conclusion'` reports `failure` | Exception | e2e: `gh run list` after a deliberately-broken throwaway PR push |
| Given `.github/workflows/ci.yml`'s `actions/setup-go` step - When compared against `go.mod` - Then its `go-version` matches `go.mod`'s `go 1.24` line exactly | Boundary | scripted check: `grep 'go-version' .github/workflows/ci.yml` vs `grep '^go ' go.mod` |
