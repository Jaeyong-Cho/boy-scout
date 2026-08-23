---
type: Spec Story
title: duplication-detector
description: Add boy-scout go duplication command to flag function copy-paste (Type-1 exact, Type-2 renamed identifiers) using token normalization
tags: [spec, clean-code-checks]
timestamp: 2026-08-24T00:00:00+09:00
---

# duplication-detector

## Value to user
Users can run `boy-scout go duplication` on code to identify functions that are exact copies or copies with renamed identifiers/literals, helping reduce code clutter and maintenance burden; Type-3 near-miss detection is a separate follow-on feature.

## Completion criteria
`boy-scout go duplication [--min-lines=N] [--format=text|json] [--exclude-file=...] [--exclude-func=...] [paths...]` successfully scans Go files (excluding `_test.go`), identifies Type-1 (exact) and Type-2 (renamed identifier) function duplicates, and reports pairs at whole-function granularity; results are included in `boy-scout go all` combined output.

## Spec

### Command: `boy-scout go duplication`

- Scans one or more file/dir paths (defaults to `.` if none given)
- Collects all `.go` files (same walker as `gofunclen`), recursing into directories and skipping `vendor/` and dot-directories
- Skips all files matching `*_test.go` (test files are not scanned for duplication)
- For each eligible file, parses it and extracts every top-level `*ast.FuncDecl` whose physical line count (from opening `{` to closing `}`, inclusive) is >= `--min-lines` (default 5)
- Tokenizes each function's body (using `go/scanner`, comments excluded) to produce:
  - **raw sequence**: token-by-token (identifiers, literals unchanged)
  - **blind sequence**: same, but every identifier → positional alias (`ID1`, `ID2`, ..., scoped per function, same original name always maps to same alias), and every literal (int/float/string/char/imag tokens) → placeholder tagged by kind (`LIT_INT`, `LIT_STRING`, etc.)
- Compares every unordered pair of eligible functions across all files (cross-file, cross-package)
  - If raw sequences match byte-for-byte → Type-1 (exact duplicate), similarity 1.0
  - Else if blind sequences match byte-for-byte → Type-2 (renamed-identifier duplicate), similarity 1.0
  - Else → no violation
- Excludes functions matching `--exclude-func` glob patterns or carrying a `// boy-scout:ignore:duplication` doc-comment directive (same semantics as every other checker)
- Respects `--exclude-file` glob patterns (same semantics as other checkers)
- Reports one violation per matched pair (not two, not per-function), with stable ordering: the function appearing first by (file, line) is always labeled "A", the later one "B"
- Outputs as human-readable text by default or JSON with `--format=json`
- Defaults to min-lines 5, overridable via `--min-lines=N`
- Skips files that fail to parse with a warning, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority over violations)
- Each text violation line is formatted: `fileA:lineA: function funcA is Type-N duplicate of fileB:lineB function funcB (N duplicated lines)`

### Integration with `boy-scout go all`

- Includes duplication results alongside gofunclen, crap, filelen, instability, abstractness in combined text/JSON output
- Text output adds a `[duplication] ` prefixed section showing duplication violations
- JSON output adds a `"duplication"` top-level key with the full duplication report
- Exit code totals (from `exitCodeFor`) include duplication's violation and skipped counts alongside all other checks

## Acceptance criteria

|AC|Category|Verification Method|
|--|--|--|
|Given two functions with identical code but different comments/whitespace - When `duplication.Check` runs - Then one violation is reported with Type `Type-1` and similarity 1.0|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ReportsType1ExactDuplicate`|
|Given `CalculateTax(amount float64)` in `billing/tax.go` and `CalculateFee(price float64)` in `billing/fee.go`, same 7-line structure with only identifiers renamed - When `duplication.Check` runs - Then one violation is reported with Type `Type-2`, correct files/lines/func names, 7 duplicated lines|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ReportsType2RenamedDuplicate`|
|Given two structurally unrelated functions of any length - When checked - Then `Report.Violations` is empty|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_NoViolationForDissimilarFunctions`|
|Given two identical 2-line getter functions and `--min-lines` default of 5 - When checked - Then `Report.Violations` is empty (both below the floor)|Boundary|unit test: `internal/duplication/duplication_test.go: TestCheck_NoViolationBelowMinLines`|
|Given a directory with exactly one eligible function - When checked - Then `Report.Violations` is empty and no panic occurs|Boundary|unit test: `internal/duplication/duplication_test.go: TestCheck_SingleFunctionProducesEmptyReport`|
|Given one copy of a duplicate pair lives in a `_test.go` file - When checked - Then that pair is never reported|Exception|unit test: `internal/duplication/duplication_test.go: TestCheck_SkipsTestFiles`|
|Given a function marked `// boy-scout:ignore:duplication` that would otherwise match another - When checked - Then the pair is not reported, and (with `Debug: true`) the excluded function appears in `Report.ExcludedFuncs`|Exception|unit test: `internal/duplication/duplication_test.go: TestCheck_ExcludeFuncByCommentDirective`|
|Given a directory with one file containing a Go syntax error - When checked - Then that file appears in `Report.Skipped` and other files are still compared|Exception|unit test: `internal/duplication/duplication_test.go: TestCheck_SkipsUnparseableFileAndContinues`|
|Given the `CalculateTax`/`CalculateFee` fixture split across two files in a temp module - When `boy-scout go duplication <dir>` runs via the built CLI - Then stdout names both files/functions, says `Type-2`, and exit code is 1|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_DuplicationReportsRenamedFunctionAsType2Clone`|
|Given the same fixture and `--format=json` - When run - Then stdout is valid JSON unmarshaling into the documented `duplication.Report` schema|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_DuplicationJSONFormatOutputsValidSchema`|

## Assertions

Each line below is a real `assert`-style guard (`assertf(cond, msg, ...)`) written directly in the implementation, not a comment or test-only check.

- `internal/duplication/duplication.go`, `Check`: precondition `assertf(minLines > 0, "minLines must be positive, got %d", minLines)` — mirrors gofunclen's guard
- `internal/duplication/duplication.go`, the pairwise comparison loop: invariant `assertf(i != j, "comparing function against itself")` around the index pair being compared
- `internal/duplication/duplication.go`, `classifyPair`: postcondition `assertf(cloneType == "" || rawA == rawB || blindA == blindB, "classified as %s but neither raw nor blind sequences match", cloneType)` — core correctness guarantee

## Known simplifications (ponytail marks)

- `ponytail:` Treating all identifiers as one class (no distinction between params/locals/fields/package-qualified names) is a known simplification. Upgrade path: split identifier classes if false positives show up in dogfooding.
- `ponytail:` O(n²) pairs — fine at boy-scout's own size (few hundred functions). Upgrade path: bucketing by sequence length before comparing, if a future dogfood run on a much larger repo is too slow.
