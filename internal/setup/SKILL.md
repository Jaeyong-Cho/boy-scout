---
name: boy-scout
description: Auto-fix code quality violations from the boy-scout lint checker
disable-model-invocation: true
---

# boy-scout

Auto-fix code quality violations from the boy-scout lint checker.

## Discover Your Project's Language

Before running boy-scout, determine what languages your project contains:

- **Go:** Look for a `go.mod` file in your project root.
- **C++:** Look for `CMakeLists.txt`, or `.cpp` and `.hpp` files in your project.

If your project matches multiple language markers (e.g., both a `go.mod` and `CMakeLists.txt`), boy-scout will run checks for every matching language and merge all violations into a single ranked list. If your project doesn't match any language marker that boy-scout supports, stop here — no checks can be run.

## Per-Language Setup

Once you've identified your language(s), read the appropriate language guide to understand the test command, available checks, and syntax for ignoring violations:

- **Go:** Read `references/lang/go/index.md` for language-specific test command, test-file rules, available checks, and ignore-comment syntax.
- **C++:** Read `references/lang/cpp/index.md` for available checks and current limitations.

## Running Checks

Run boy-scout with your language(s):

```bash
# Go
boy-scout go all

# C++
boy-scout cpp funclen

# Mixed project (runs both, merges results)
boy-scout go all
boy-scout cpp funclen
```

For mixed-language projects, merge all violations from each language into a single list, then process them in the order below.

## Before Starting

**Check the tests are green:** Run your project's test suite once, before touching anything. If it fails, stop immediately and report that tests were already failing before any boy-scout edit — fix the test suite first, then re-run this skill. Never refactor on top of a red test suite; you can't tell your edit from pre-existing breakage.

## Processing Violations

For each violation, edit the flagged function or file in this order: funclen violations first, then same-package duplication clusters, then crap, then filelen, then instability, then abstractness, then cross-package duplication clusters last. The more disruptive the fix, the later it runs — same-package duplication clusters are local extractions (cheap), file-level reorganization doesn't invalidate line numbers of unresolved violations yet to come, but package-boundary reshapes (including cross-package duplication clusters) do, so instability, abstractness, and cross-package clusters go last.

**Cap each run at 5 violations.** Within each kind above, process the worst violations first: rank by the severity number boy-scout already prints — for funclen and filelen, lines minus the limit; for crap, the CRAP score; for duplication, the summed `DupLines` (lines resolved by fixing this cluster); for instability, the Gap; for abstractness, the Distance — highest number first. A duplication cluster counts as 1 against the 5-violation cap regardless of how many pairwise duplicate lines it contains. Regardless of kind or severity, violations in test files (`*_test.go` in Go, test files in C++) are deferred to the end of the list, after every non-test violation. Stop once 5 violations total have been processed (fixed or marked unresolved) this run, including any characterization test step — same accounting as the existing 3-attempts-per-violation cap. At the end, report fixed vs. unresolved as before, plus how many remaining violations were skipped by the cap (never attempted) — distinct from unresolved (attempted, failed 3 times).

Before editing each violation, read both reference files below: the language-agnostic explanation and your language's concrete example.

| Violation kind | Why/How | Go Example | C++ Example |
|---|---|---|---|
| `funclen` (or `gofunclen` in Go) | `references/funclen.md` | `references/lang/go/funclen.md` | `references/lang/cpp/funclen.md` |
| `crap` | `references/crap.md` | `references/lang/go/crap.md` | (not yet supported for C++) |
| `filelen` | `references/filelen.md` | `references/lang/go/filelen.md` | `references/lang/cpp/filelen.md` |
| `duplication` | `references/duplication.md` | `references/lang/go/duplication.md` | (not yet supported for C++) |
| `instability` | `references/instability.md` | `references/lang/go/instability.md` | `references/lang/cpp/instability.md` |
| `abstractness` | `references/abstractness.md` | `references/lang/go/abstractness.md` | `references/lang/cpp/abstractness.md` |

Each "Why/How" file explains the concept and fix strategy generically. Each language-specific file shows a concrete before/after code example in that language's syntax.

## Special Rules by Violation Kind

**CRAP violations in Go test files:** CRAP violations never appear for `_test.go` files by default (no flag needed) — the coverage formula is meaningless for test code by construction, since `go test -coverprofile` never instruments test files. If a `crap` violation is ever reported in a test file, it's a tool bug to flag, not test code to refactor.

**Coverage for CRAP violations:** For a crap violation, check the coverage percentage already printed in the violation line. If it's 0%, first add one minimal characterization test — a test that pins down what the function does right now, not what it should do — and confirm it passes, before refactoring. If coverage is already above 0%, skip straight to refactoring; the existing tests plus the re-run-after-edit step below are enough of a safety net.

**Duplication violations:** Run `--format=json` (not text output) to read the cluster's `members`, `pairs`, `dupLines`, and `crossPackage` fields — these are necessary to understand which functions are duplicates and how they're related. Fix the whole cluster in one atomic multi-file edit: extract one shared helper function covering every member's duplicate body, repoint every caller to use the shared helper, and delete every duplicate body once all callers switch over. This is one violation fix, not N separate fixes. After the extraction, run your test suite once to confirm the shared helper's behavior matches all the original copies. Default fix is always extract-a-helper (reversible, leaves both functions available if they later diverge), never delete-one-copy (you lose one implementation and can't recover it if the copies turn out to serve different purposes).

## Verification Loop

After each edit — including writing a characterization test — re-run your project's test suite and boy-scout for your language(s) to verify the fix. If the fix succeeds (both commands green), move to the next violation. If a violation, including its characterization test if one was needed, fails to fix after 3 attempts total, mark it unresolved and continue with the rest.

## End of Run: Summary and Commit

At the end of the run, report the count of fixed vs. unresolved violations:
- If boy-scout finds zero violations, report clean and make no edits.
- If violations were fixed, show a summary: how many fixed, how many unresolved, how many skipped by the 5-per-run cap.

**Check first**: the consistency of naming of source file, function, variables and documents.

**Commit your changes:** Ask the user to confirm they want to commit the fixes:

```
You fixed N violations (M unresolved, K skipped by cap).
Ready to commit these changes? (yes/no)
```

- If **yes:** Run `git commit -m "Fix boy-scout violations (fixed: N, unresolved: M)"` to commit with a standard message.
- If **no:** Stop here. The changes remain in your working directory for manual review or further edits before you commit manually.
