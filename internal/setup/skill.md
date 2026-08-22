---
name: gardener-go
description: Auto-fix code quality violations from the gardener-go lint checker
disable-model-invocation: true
---

# gardener-go

Auto-fix code quality violations from the gardener-go lint checker.

**Note on CRAP violations in test files:** CRAP violations never appear for `_test.go` files by default (no flag needed) — the coverage formula is meaningless for test code by construction, since `go test -coverprofile` never instruments test files. If a `crap` violation is ever reported in a test file, it's a tool bug to flag, not test code to refactor.

**Before starting, check the tests are green:** run `go test ./...` once, before touching anything. If it fails, stop immediately and report that tests were already failing before any gardener-go edit — fix the test suite first, then re-run this skill. Never refactor on top of a red suite; you can't tell your edit from pre-existing breakage.

Run `gardener-go all` to find violations. For each violation, edit the flagged function (funclen violations first, then crap violations). Before editing, state in plain terms which clean-code rule it breaks:

- **funclen violation:** the function is too big to hold one level of abstraction — it's doing more than one thing. Fix it by extracting the sub-steps into well-named helper functions, not by just trimming lines.
- **crap violation:** the function combines high complexity plus low test coverage — nobody can prove a change to it is safe. Fix it by simplifying the logic, backed by a real test.

For a crap violation, check the coverage percentage already printed in the violation line. If it's 0%, first add one minimal characterization test — a test that pins down what the function does right now, not what it should do — and confirm it passes, before refactoring. If coverage is already above 0%, skip straight to refactoring; the existing tests plus the re-run-after-edit step below are enough of a safety net.

After each edit — including writing a characterization test — re-run `gardener-go all` and `go test ./...` to verify the fix. If the fix succeeds (both commands green), move to the next violation. If a violation, including its characterization test if one was needed, fails to fix after 3 attempts total, mark it unresolved and continue with the rest. At the end, report the count of fixed vs. unresolved violations. If `gardener-go all` finds zero violations, report clean and make no edits. Never commit changes with git.
