---
type: Spec Story
title: file-length-limit
description: Add gardener filelen command to flag files exceeding a configurable line-length limit, dispatched per language (go/cpp) like funclen
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# file-length-limit

## Value to user
Users can run `gardener go filelen` or `gardener cpp filelen` on code to identify files that exceed a configurable line-length threshold (default 300 lines), helping maintain code organization and reduce cognitive overhead; results combine with `gardener go all` checks (gofunclen, crap).

## Completion criteria
`gardener go filelen [--max-lines=N] [--format=text|json] [paths...]` and `gardener cpp filelen` successfully scan files and report those exceeding the line limit; `gardener go all` includes filelen results in combined output.

## Spec
The `gardener go filelen` subcommand:
- Scans one or more file/dir paths (defaults to `.` if none given)
- Counts total physical lines per file (count of `\n` bytes, +1 if the file has trailing content with no final newline; an empty file is 0 lines)
- Flags any file where lines > limit (exactly limit is compliant, matching `funclen`'s boundary convention)
- Recurses into directories, skipping `vendor/` and any dir whose name starts with `.`
- Outputs as human-readable text by default or JSON with `--format=json`
- Defaults to limit 300, overridable via `--max-lines=N`
- Skips files that fail to read (permission error, binary content) with a warning, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority over violations)
- Respects `--exclude-file` glob patterns (same semantics as funclen)
- No `--exclude-func` flag (filelen has no function-level concept)

The `gardener cpp filelen` subcommand:
- Identical to `gardener go filelen` but scans only `.cpp`, `.h`, and `.hpp` files

The `gardener go all` subcommand:
- Includes filelen results alongside gofunclen and crap in combined text/JSON output
- Text output adds a `[filelen] ` prefixed section showing filelen violations
- JSON output adds a `"filelen"` top-level key with the full filelen report
- Exit code totals (from `exitCodeFor`) include filelen's violation and skipped counts alongside gofunclen and crap

## Acceptance criteria

|AC|Category|Verification Method|
|--|--|--|
|Given a file with 300 lines or fewer - When `filelen.Check` runs on it - Then `Report.Violations` is empty|Normal|unit test: `internal/filelen/filelen_test.go: TestCheck_NoViolationUnderLimit`|
|Given a file with 301 lines - When `filelen.Check` runs - Then `Report.Violations` has one entry with the correct file, lines=301, limit=300|Normal|unit test: `internal/filelen/filelen_test.go: TestCheck_ReportsViolationOverLimit`|
|Given a file of exactly 300 lines - When checked against limit 300 - Then no violation is reported|Boundary|unit test: `internal/filelen/filelen_test.go: TestCheck_ExactlyAtLimitIsCompliant`|
|Given a completely empty file (0 bytes) - When checked - Then it reports 0 lines and no violation|Boundary|unit test: `internal/filelen/filelen_test.go: TestCheck_EmptyFileIsCompliant`|
|Given a directory with nested subdirectories, some files over the limit, one inside `vendor/`, one inside `.hidden/` - When `filelen.Check` runs - Then violations come from every non-skipped file and `vendor/`/dot-dirs are not scanned|Normal|unit test: `internal/filelen/filelen_test.go: TestCheck_WalksDirectoryRecursivelySkippingVendorAndDotDirs`|
|Given an empty directory (no matching files) - When checked - Then `Report.Violations` and `Report.Skipped` are both empty, no error|Boundary|unit test: `internal/filelen/filelen_test.go: TestCheck_EmptyDirectoryProducesEmptyReport`|
|Given a directory with one file with no read permission - When checked - Then that file appears in `Report.Skipped` with its read error, and other files in the same run are still checked|Exception|unit test: `internal/filelen/filelen_test.go: TestCheck_SkipsUnreadableFileAndContinues`|
|Given `--exclude-file` matching a file that would otherwise violate - When checked - Then that file is excluded, not reported as a violation|Normal|unit test: `internal/filelen/filelen_test.go: TestCheck_RespectsExcludeFilePattern`|
|Given `gardener go filelen --max-lines=50 <path>` against a 60-line `.go` file - When run - Then it's reported as a violation with limit=50|Normal|unit test: `cmd/gardener/main_test.go: TestRun_FilelenRespectsMaxLinesFlag`|
|Given `gardener go filelen <path>` (default text format) with one violation - When run - Then stdout contains a line formatted `file: N lines (limit 300)`|Normal|unit test: `cmd/gardener/main_test.go: TestRun_FilelenTextFormatMatchesExpectedLine`|
|Given `gardener go filelen --format=json <path>` with violations present - When run - Then stdout is valid JSON matching the documented schema|Normal|unit test: `cmd/gardener/main_test.go: TestRun_FilelenJSONFormatOutputsValidSchema`|
|Given `gardener cpp filelen <path>` against a directory with both a 350-line `.cpp` file and a 350-line `.go` file - When run - Then only the `.cpp` file is reported (the `.go` file is out of scope for the `cpp` dispatch)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_CppFilelenOnlyScansCppExtensions`|
|Given `gardener go all` on a fixture with a gofunclen violation, a crap violation, and a filelen violation all present - When run - Then text output contains all three of `[gofunclen] `, `[crap] `, `[filelen] `, and `--format=json` output has all three top-level keys|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllIncludesFilelen` and `TestRun_AllIncludesFilelenJSON`|
|Given a run of `gardener go filelen` with ≥1 skipped file and 0 violations / ≥1 violation and 0 skipped / neither - When it finishes - Then exit code is 2 / 1 / 0 respectively|Exception|unit test: `cmd/gardener/main_test.go: TestRun_FilelenExitCodeReflectsOutcome` (table test, 3 cases, mirrors `TestRun_ExitCodeReflectsOutcome`)|
