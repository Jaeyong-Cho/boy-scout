---
name: boy-scout
description: Auto-fix code quality violations from the boy-scout lint checker
disable-model-invocation: true
---

# boy-scout

Auto-fix code quality violations from the boy-scout lint checker.

**Note on CRAP violations in test files:** CRAP violations never appear for `_test.go` files by default (no flag needed) — the coverage formula is meaningless for test code by construction, since `go test -coverprofile` never instruments test files. If a `crap` violation is ever reported in a test file, it's a tool bug to flag, not test code to refactor.

**Before starting, check the tests are green:** run `go test ./...` once, before touching anything. If it fails, stop immediately and report that tests were already failing before any boy-scout edit — fix the test suite first, then re-run this skill. Never refactor on top of a red suite; you can't tell your edit from pre-existing breakage.

Run `boy-scout go all` to find violations. For C++ codebases, run `boy-scout cpp funclen` instead (C++ has no CRAP check and no `boy-scout:ignore` comment support yet). For each violation, edit the flagged function or file in this order: funclen violations first, then crap, then filelen, then instability, then abstractness. The more disruptive the fix, the later it runs — file-level reorganization doesn't invalidate line numbers of unresolved violations yet to come, but package-boundary reshapes do, so instability and abstractness go last.

**Cap each run at 5 violations.** Within each kind above, process the worst violations first: rank by the severity number `boy-scout go all` already prints — for funclen/gofunclen and filelen, lines minus the limit; for crap, the CRAP score; for instability, the Gap; for abstractness, the Distance — highest number first. Regardless of kind or severity, violations in test files (`*_test.go`) are deferred to the end of the list, after every non-test violation. Stop once 5 violations total have been processed (fixed or marked unresolved) this run, including any characterization test step — same accounting as the existing 3-attempts-per-violation cap. At the end, report fixed vs. unresolved as before, plus how many remaining violations were skipped by the cap (never attempted) — distinct from unresolved (attempted, failed 3 times).

Before editing each violation, read the corresponding reference file below. Each file has two sections: why it's a problem, and how to fix it. Both sections are concrete and specific to that violation kind — read both before starting to edit.

| Violation kind | Reference file |
|---|---|
| `funclen` (or `gofunclen` in Go) | `references/funclen.md` |
| `crap` | `references/crap.md` |
| `filelen` | `references/filelen.md` |
| `instability` | `references/instability.md` |
| `abstractness` | `references/abstractness.md` |

Note: Go's boy-scout uses the token `gofunclen` for the CLI subcommand, and C++'s uses `funclen`; both violations are explained in `references/funclen.md`.

For a crap violation, check the coverage percentage already printed in the violation line. If it's 0%, first add one minimal characterization test — a test that pins down what the function does right now, not what it should do — and confirm it passes, before refactoring. If coverage is already above 0%, skip straight to refactoring; the existing tests plus the re-run-after-edit step below are enough of a safety net.

After each edit — including writing a characterization test — re-run `boy-scout go all` and `go test ./...` to verify the fix. If the fix succeeds (both commands green), move to the next violation. If a violation, including its characterization test if one was needed, fails to fix after 3 attempts total, mark it unresolved and continue with the rest. At the end, report the count of fixed vs. unresolved violations. If `boy-scout go all` finds zero violations, report clean and make no edits. Never commit changes with git.
