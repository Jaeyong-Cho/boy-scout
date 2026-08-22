---
type: Spec Story
title: crap-score
description: Add gardener crap command to flag functions whose CRAP score (complexity × untested-risk) exceeds a configurable threshold
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# crap-score

## Value to user
Users can run `gardener crap` on Go code to identify functions with high Change Risk Anti-Pattern (CRAP) scores — those combining cyclomatic complexity with low test coverage — helping catch risky code before it lands. A fully tested simple function scores low (safe to change); an untested tangled function scores high (risky to change).

## Completion criteria
`gardener crap [--threshold=N] [--format=text|json] [paths...]` successfully scans Go files, calculates CRAP scores (formula: `comp² × (1−cov)³ + comp`), and reports functions exceeding the threshold (default 6); `gardener all` now runs both funclen and crap and combines their output.

## Spec
The `gardener crap` subcommand:
- Scans one or more file/dir paths (defaults to `.` if none given)
- Computes cyclomatic complexity for each top-level `*ast.FuncDecl`:
  - Base complexity 1
  - +1 for each `if`, `for`, `range`, `case` (non-default), `comm` (select), logical AND/OR operator
- Automatically runs `go test -covermode=set -coverprofile=<tempfile>` itself (no pre-existing coverage file needed, no `--coverage` flag)
- Extracts test coverage from the generated profile, matched to source files via the module path from `go.mod`
- Calculates CRAP score: `comp² × (1 − cov)³ + comp` (comp = complexity, cov = fraction covered in [0.0, 1.0])
- Treats files absent from coverage profile as 0% covered; functions with zero coverable statements as 100% covered (vacuous)
- If `go test` fails to build, returns fatal error (exit 2); if tests compile but fail assertions, proceeds with partial coverage
- Defaults to threshold 6, overridable via `--threshold=N`
- Outputs as human-readable text by default or JSON with `--format=json`
- Reports format: `file:line: function Name has CRAP score X.XX (complexity=C, coverage=P.P%, threshold=T.TT)`
- Skips files that fail to parse, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped or fatal error (takes priority)

The `gardener all` subcommand now runs both funclen and crap with hardcoded defaults (maxLines=50, threshold=6) and combines their results. Text output prefixes lines with `[funclen]` or `[crap]`; JSON nests results per-check as `{"funclen":{...},"crap":{...}}`. Exit code 2 takes priority if either check hits skipped/fatal conditions.

Dependencies: Go stdlib only (go/ast, go/parser, go/token, os/exec, regexp or manual parsing for coverage file). No third-party modules.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a function with no branches - When `cyclomaticComplexity` runs on it - Then it returns 1|Boundary|unit test: `internal/crap/complexity_test.go: TestCyclomaticComplexity_StraightLineIsOne`|
|Given a function with one `if` statement - When `cyclomaticComplexity` runs - Then it returns 2|Normal|unit test: `internal/crap/complexity_test.go: TestCyclomaticComplexity_IfAddsOne`|
|Given a function with one `for` loop and one `range` loop - When `cyclomaticComplexity` runs - Then it returns 3 (base 1 + 1 + 1)|Normal|unit test: `internal/crap/complexity_test.go: TestCyclomaticComplexity_ForAndRangeEachAddOne`|
|Given a function with `if a \|\| b {}` - When `cyclomaticComplexity` runs - Then `\|\|` and the `if` each add 1 (returns 3)|Normal|unit test: `internal/crap/complexity_test.go: TestCyclomaticComplexity_LogicalAndOrAddsOne`|
|Given a `switch` with 3 non-default cases and one `default:` - When `cyclomaticComplexity` runs - Then it returns 4 (base 1 + 3, default excluded)|Boundary|unit test: `internal/crap/complexity_test.go: TestCyclomaticComplexity_SwitchCasesCountExcludingDefault`|
|Given a valid `go test -coverprofile` text with 2 profile lines - When `parseProfile` runs - Then it returns 2 blocks with the correct file/line-range/numStmt/count fields|Normal|unit test: `internal/crap/coverage_test.go: TestParseProfile_ParsesValidEntries`|
|Given a scanned file with zero matching blocks anywhere in the profile - When `functionCoverage` runs for any function in it - Then it returns 0.0|Boundary|unit test: `internal/crap/coverage_test.go: TestFunctionCoverage_FileAbsentFromProfileIsZero`|
|Given a function with zero coverable statements in a file that DOES appear elsewhere in the profile - When `functionCoverage` runs - Then it returns 1.0 (vacuously covered, not a divide-by-zero)|Boundary|unit test: `internal/crap/coverage_test.go: TestFunctionCoverage_EmptyFunctionBodyIsFullyCovered`|
|Given a function whose matching blocks sum to 10 statements, 6 of them with count>0 - When `functionCoverage` runs - Then it returns 0.6|Normal|unit test: `internal/crap/coverage_test.go: TestFunctionCoverage_PartialCoverageComputesFraction`|
|Given comp=2,cov=1.0 / comp=2,cov=0.0 / comp=4,cov=0.5 / comp=1,cov=1.0 - When `crapScore` runs - Then it returns 2 / 6 / 6 / 1 respectively|Normal|unit test: `internal/crap/crap_test.go: TestCrapScore_MatchesFormula` (table test, 4 cases)|
|Given a score exactly equal to the threshold - When `evaluate` runs - Then it reports not-violated|Boundary|unit test: `internal/crap/crap_test.go: TestEvaluate_ScoreExactlyAtThresholdIsCompliant`|
|Given a score greater than the threshold - When `evaluate` runs - Then it reports violated|Normal|unit test: `internal/crap/crap_test.go: TestEvaluate_ScoreOverThresholdIsViolation`|
|Given a real temp Go module with a package that has a branchy, untested function - When `crap.Check` runs against it - Then `Report.Violations` has one entry with correct file/line/func/complexity/coverage/score|Normal|unit test: `internal/crap/crap_test.go: TestCheck_ReportsViolationForComplexUntestedFunction`|
|Given a real temp Go module where the package fails to compile - When `crap.Check` runs - Then it returns a non-nil `error` and no coverage file is left behind|Exception|unit test: `internal/crap/crap_test.go: TestCheck_BuildFailureReturnsError`|
|Given a real temp Go module where tests compile but one assertion fails - When `crap.Check` runs - Then it still returns a report scored from the partial coverage produced (no error)|Exception|unit test: `internal/crap/crap_test.go: TestCheck_TestFailureStillScoresWithPartialCoverage`|
|Given a temp directory with no `go.mod` anywhere above it - When `crap.Check` runs - Then it returns a non-nil `error`|Exception|unit test: `internal/crap/crap_test.go: TestCheck_MissingGoModReturnsError`|
|Given the same directory-collection scenario funclen already covers (nested dirs, `vendor/`, dot-dirs) - When `gofiles.Collect` runs - Then it returns the same file set funclen's existing test expects, and `funclen.Check`'s existing tests still pass unmodified after the refactor|Normal|unit test: `internal/gofiles/gofiles_test.go: TestCollect_WalksDirectoryRecursivelySkippingVendorAndDotDirs` + existing `internal/funclen/funclen_test.go` suite (regression, must stay green)|
|Given `gardener crap --threshold=2 <path>` against a function scoring 3 - When run - Then it's reported as a violation with threshold=2 in the output|Normal|unit test: `cmd/gardener/main_test.go: TestRun_CrapRespectsThresholdFlag`|
|Given `gardener crap <path>` (default text format) with one violation - When run - Then stdout contains a line formatted `file:line: function Name has CRAP score X.XX (complexity=C, coverage=P.P%, threshold=T.TT)`|Normal|unit test: `cmd/gardener/main_test.go: TestRun_CrapTextFormatMatchesExpectedLine`|
|Given `gardener crap --format=json <path>` with violations present - When run - Then stdout is valid JSON that unmarshals into the documented schema|Normal|unit test: `cmd/gardener/main_test.go: TestRun_CrapJSONFormatOutputsValidSchema`|
|Given `gardener all <path>` against a package with both a funclen violation and a crap violation - When run - Then the combined output (text and JSON) contains both, nested/labelled per check|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllCombinesFunclenAndCrap`|
|Given a run where either check has ≥1 skipped/fatal-error condition - When `all` finishes - Then exit code is 2, taking priority over violations from the other check|Exception|unit test: `cmd/gardener/main_test.go: TestRun_AllExitCodePriorityAcrossBothChecks` (table test, 4 cases)|
