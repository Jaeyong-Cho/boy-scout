---
name: gardener-go
description: Auto-fix code quality violations from the gardener-go lint checker
disable-model-invocation: true
---

# gardener-go

Auto-fix code quality violations from the gardener-go lint checker.

Run `gardener-go all` to find violations. For each violation, edit the flagged function (funclen violations first, then crap violations). After each edit, re-run `gardener-go all` and `go test ./...` to verify the fix. If the fix succeeds (both commands green), move to the next violation. If a violation fails to fix after 3 attempts, mark it unresolved and continue with the rest. At the end, report the count of fixed vs. unresolved violations. If `gardener-go all` finds zero violations, report clean and make no edits. Never commit changes with git.
