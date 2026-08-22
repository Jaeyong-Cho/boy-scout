---
name: gardener-go
description: Auto-fix code quality violations from the gardener-go lint checker
disable-model-invocation: true
---

# gardener-go

Auto-fix code quality violations from the gardener-go lint checker.

**Note on CRAP violations in test files:** CRAP violations never appear for `_test.go` files by default (no flag needed) — the coverage formula is meaningless for test code by construction, since `go test -coverprofile` never instruments test files. If a `crap` violation is ever reported in a test file, it's a tool bug to flag, not test code to refactor.

Run `gardener-go all` to find violations. For each violation, edit the flagged function (funclen violations first, then crap violations). After each edit, re-run `gardener-go all` and `go test ./...` to verify the fix. If the fix succeeds (both commands green), move to the next violation. If a violation fails to fix after 3 attempts, mark it unresolved and continue with the rest. At the end, report the count of fixed vs. unresolved violations. If `gardener-go all` finds zero violations, report clean and make no edits. Never commit changes with git.
