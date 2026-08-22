---
type: Spec Story
title: func-length-limit
description: Add gardener funclen command to flag functions exceeding a configurable line-length limit
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# func-length-limit

## Value to user
Users can run `gardener funclen` on Go code to identify functions that exceed a configurable line-length threshold (default 100 lines), helping maintain code readability and reduce cognitive complexity before functions land in the codebase.

## Completion criteria
`gardener funclen [--max-lines=N] [--format=text|json] [paths...]` successfully scans Go files and reports functions exceeding the line limit; `gardener all` runs every registered check (funclen today) and combines their output.

## Spec
The `gardener funclen` subcommand:
- Scans one or more file/dir paths (defaults to `.` if none given)
- Counts function length as physical source lines from opening `{` to closing `}`, inclusive
- Flags any function where length > limit (exactly limit is compliant)
- Checks every top-level `*ast.FuncDecl` (functions and methods, including generics; anonymous closures are NOT separately flagged)
- Recurses into directories, skipping `vendor/` and any dir whose name starts with `.`
- Outputs as human-readable text by default or JSON with `--format=json`
- Defaults to limit 100, overridable via `--max-lines=N`
- Skips files that fail to parse with a warning, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority over violations)

The `gardener all` subcommand runs every registered check (funclen only, today) and combines their results into a single output.

Bare `gardener` with no subcommand prints usage to stderr and exits non-zero.

Dependencies: Go stdlib only. No third-party modules.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a Go file where every function is ≤100 lines - When `funclen.Check` runs on it - Then `Report.Violations` is empty|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_NoViolationUnderLimit`|
|Given a Go file with one 105-line function - When `funclen.Check` runs - Then `Report.Violations` has one entry with the correct file, func name, line, length=105, limit=100|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ReportsViolationOverLimit`|
|Given a function of exactly 100 lines - When checked against limit 100 - Then no violation is reported|Boundary|unit test: `internal/funclen/funclen_test.go: TestCheck_ExactlyAtLimitIsCompliant`|
|Given a directory with nested subdirectories containing multiple `.go` files, some over the limit, one inside `vendor/` and one inside a `.hidden/` dir - When `funclen.Check` runs on that directory - Then violations are reported from every non-skipped file and `vendor/`/dot-dirs are not scanned|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_WalksDirectoryRecursivelySkippingVendorAndDotDirs`|
|Given an empty directory (no `.go` files) - When checked - Then `Report.Violations` and `Report.Skipped` are both empty and no error is returned|Boundary|unit test: `internal/funclen/funclen_test.go: TestCheck_EmptyDirectoryProducesEmptyReport`|
|Given a directory with one file containing a Go syntax error - When checked - Then that file appears in `Report.Skipped` with its parse error, and other files in the same run are still checked|Exception|unit test: `internal/funclen/funclen_test.go: TestCheck_SkipsUnparseableFileAndContinues`|
|Given `gardener funclen --max-lines=50 <path>` against a 60-line function - When run - Then it's reported as a violation with limit=50|Normal|unit test: `cmd/gardener/main_test.go: TestRun_FunclenRespectsMaxLinesFlag`|
|Given `gardener funclen --format=json <path>` with violations present - When run - Then stdout is valid JSON that unmarshals into the documented schema|Normal|unit test: `cmd/gardener/main_test.go: TestRun_JSONFormatOutputsValidSchema`|
|Given `gardener funclen <path>` (default text format) with one violation - When run - Then stdout contains a line formatted `file:line: function Name is N lines (limit M)`|Normal|unit test: `cmd/gardener/main_test.go: TestRun_TextFormatMatchesExpectedLine`|
|Given `gardener funclen` with no path argument, run from a directory containing Go files - When run - Then it scans the current directory (`.`)|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_FunclenDefaultsToCurrentDir`|
|Given a run with zero violations and zero skipped files / given a run with ≥1 violation and 0 skipped / given a run with ≥1 skipped file - When it finishes - Then exit code is 0 / 1 / 2 respectively (skipped takes priority over violations)|Exception|unit test: `cmd/gardener/main_test.go: TestRun_ExitCodeReflectsOutcome` (table test, 3 cases)|
|Given `gardener all <path>` - When run - Then every registered check (funclen only, today) runs against that path and results are combined into one output|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllRunsEveryRegisteredCheck`|
|Given `gardener` invoked with no subcommand - When run - Then usage text is printed to stderr, exit code is non-zero, and no scan is performed|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_NoSubcommandPrintsUsage`|
