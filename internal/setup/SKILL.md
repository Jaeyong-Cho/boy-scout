---
name: boy-scout
description: List code quality violations from the boy-scout lint checker, with code and reasoning; once the user picks which to fix, propose a fix plan for review, then stop — the user drives the actual change via @skills/to-plan and @skills/do-plan
disable-model-invocation: true
---

# boy-scout

List code quality violations from the boy-scout lint checker — show each candidate's code and the reason it's flagged. Once you pick which to fix, it proposes a fix plan for each pick so you can review the approach, then stops. This skill never edits or commits a file; you drive the actual change through @skills/to-plan and @skills/do-plan.

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

**Check the tests are green:** Run your project's test suite once, before reviewing anything. If it fails, stop immediately and report that tests were already failing before this review — fix the test suite first, then re-run this skill. Violations reviewed against a red baseline are not trustworthy: you can't tell a real violation from fallout of the existing breakage.

## Selecting Candidates

Review violations in order of disruption (least to most): funclen, same-package duplication, crap, filelen, instability, abstractness, cross-package duplication.

**Within each type, select one candidate per run.** Identify the worst by boy-scout's severity: lines-over-limit (funclen/filelen), CRAP score (crap), summed DupLines (duplication), Gap (instability), Distance (abstractness). Test file violations (`*_test.go`, C++ tests) go last. This produces at most one candidate per type — never select more than one, and never edit anything in this step.

Before showing a candidate, read: language-agnostic reference (`references/{violation-type}.md`) and language-specific example (`references/lang/{go|cpp}/{violation-type}.md`).

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

**Duplication violations:** Run `--format=json` (not text output) to read the cluster's `members`, `pairs`, `dupLines`, and `crossPackage` fields — these are necessary to show which functions are duplicates of each other and how they're related. `references/duplication.md`'s "How to fix it" section covers the actual fix strategy for whoever runs @skills/to-plan next; this skill only needs the fields above to explain the candidate.

## Showing Each Candidate

For each selected candidate, show:

1. The file:line and the severity number boy-scout printed.
2. The flagged code — read the actual function/file boy-scout pointed at and quote it (the whole function for funclen/crap, the relevant snippet for filelen/duplication/instability/abstractness).
3. A one-paragraph *why*, using the matching reference file's "Why this is a problem" section, filled in with this candidate's real numbers (e.g. "complexity=8, coverage=75%" for a crap violation) — not a generic copy-paste.
4. A one-line *fix strategy* pointer, quoting the first sentence of the matching reference file's "How to fix it" section.

**Never edit or write to any file in this step.** Reading a flagged file to quote its code is fine; writing to it is not.

## End of Run: Hand Off to the User

- If boy-scout finds zero violations, report clean and stop — nothing to show, nothing to select.
- Otherwise, list every candidate selected above (at most one per type), each numbered with its code, why, and fix-strategy pointer, then ask which one(s) the user wants fixed. Example shape:

  ```
  Found N violation(s) to review:

  1. [crap] internal/duplication/duplication.go:454 — sortClustersByDupLines, CRAP 9.00
     <quoted code>
     Why: ...
     Fix strategy: ...

  2. [filelen] ...

  Which one(s) do you want fixed?
  ```

## Proposing a Fix Plan for Selected Violations

Once the user answers which candidate(s) they want fixed:

- If the answer doesn't match any listed candidate, say so and re-show the numbered list — never fabricate a fix plan for something that wasn't offered.
- Otherwise, for each selected candidate, show one **Fix Plan** block: 2-5 ordered steps, each naming the exact file(s)/function(s)/package(s) it touches, expanded from the matching reference file's "How to fix it" section (the same file already read in Selecting Candidates). If more than one candidate was selected, show one Fix Plan block per candidate, in the order they were listed above.
- This is a preview only — no code is written, no file is edited, nothing is drafted to disk in this step.

Example shape:

```
Fix Plan — #1 [crap] internal/duplication/duplication.go:454 sortClustersByDupLines

1. Extract the `if dup.CrossPackage` branch (lines 460-468) into `classifyDuplicate(dup) string`.
2. Extract the sort comparator (lines 470-478) into `byDupLinesDesc(a, b Duplicate) bool`.
3. Replace the two extracted blocks in `sortClustersByDupLines` with calls to the new functions.
4. Re-run `go test ./internal/duplication/...` to confirm CRAP drops below 6.00.

Run @skills/to-plan on this pick to draft the real plan, then @skills/do-plan to execute it.
```

- Do not edit, refactor, or commit anything in this skill — the Fix Plan block above is a preview, not a change. `@skills/to-plan` and `@skills/do-plan` own the actual change from here.

