---
type: Spec Story
title: crap-ignores-test-files
description: CRAP scorer excludes _test.go files by default since coverage data never exists for them
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# crap-ignores-test-files

## Value to user

CRAP violations are never reported for functions in `_test.go` files by default, eliminating false positives from the coverage formula (which is meaningless for test code by construction, since `go test -coverprofile` never instruments test files). The solver doesn't waste time on "fixing" untested test helpers.

## Completion criteria

`gardener-go crap` and `gardener-go all` exclude `*_test.go` files by default (no flag, no opt-out) from CRAP scoring, while `funclen` remains unaffected and continues to score test files exactly as before.

## Spec

The CRAP checker's coverage formula relies on instrumentation data from `go test -coverprofile`, which never includes `_test.go` files — so every function in one always scores 0% coverage, making the formula meaningless for test code by construction. Exclude `*_test.go` by default in `crap.Check`, reusing the exclude-glob machinery already built for `--exclude-file`, with no new flag and no opt-out.

### Spec changes

- `crap.Check` (and therefore `gardener-go crap` and the `crap` section of `gardener-go all`) excludes any file matching `*_test.go` by default — no flag needed.
- The built-in default merges (union) with any `--exclude-file`/`ExcludeFiles` patterns the caller already passes; passing your own patterns never removes or replaces the default.
- No new CLI flag is added — no way to opt back in, because coverage data for a `_test.go` file structurally can't exist (it's never in a `go test -coverprofile` report), so scoring one would always be meaningless.
- `--debug`'s existing `ExcludedFiles` reporting shows default-excluded test files exactly like manually-excluded ones — no new field, no new "reason" tag.
- `funclen` is unaffected — it keeps scoring functions inside `_test.go` files exactly as before this story (explicit out-of-scope).
- Coverage collection (`go test -coverprofile` run across all input paths) is unaffected — only the post-coverage file-scan/report stage skips `*_test.go`.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given a `_test.go` file with a complex helper function (never in any coverage profile) and no `--exclude-file` flag - When `crap.Check` runs - Then no violation is reported for that function|Normal|unit test: `internal/crap/crap_test.go: TestCheck_TestFilesExcludedFromCrapByDefault`|
|Given the same fixture - When `crap.Check` runs with `Options.Debug=true` - Then `report.ExcludedFiles` contains the `_test.go` file|Normal|unit test: `internal/crap/crap_test.go: TestCheck_DefaultTestFileExcludeVisibleInDebugOutput`|
|Given the caller also passes `Options.ExcludeFiles: []string{"mocks_test.go"}` alongside a `_test.go` fixture - When `crap.Check` runs - Then both the user pattern and the built-in `*_test.go` default apply, no error, no duplicate entries crash|Boundary|unit test: `internal/crap/crap_test.go: TestCheck_DefaultTestFileExcludeCombinesWithUserExcludeFile`|
|Given a file named `contest.go` (contains "test" but wrong suffix) with a complex untested function - When `crap.Check` runs - Then it IS still reported as a violation (suffix match only, no accidental substring match)|Boundary|unit test: `internal/crap/crap_test.go: TestCheck_FileNotMatchingTestSuffixStillScored`|
|Given the same kind of complex, never-covered function placed in a `_test.go` file - When `funclen.Check` runs (not `crap.Check`) - Then it is still evaluated exactly as before this story (funclen has no default test-file exclude)|Exception (regression/out-of-scope lock)|unit test: `internal/funclen/funclen_test.go: TestCheck_StillScoresTestFilesUnaffectedByCrapDefault`|
|Given `gardener-go crap <path>` run from the CLI (no flags) against a fixture with a complex `_test.go` function - When run - Then stdout has no violation line for it and the exit code reflects only real (non-test) violations|Normal|integration test: `cmd/gardener-go/main_test.go: TestRun_CrapIgnoresTestFilesByDefaultOnCLI`|
|Given `gardener-go all <path>` run against the same fixture - When run - Then the `[crap]`-prefixed section of the combined output has no line for it, same as running `crap` alone|Normal|integration test: `cmd/gardener-go/main_test.go: TestRun_AllCrapSectionIgnoresTestFilesByDefault`|
|Given `internal/setup/skill.md` after this change - When read - Then it states CRAP ignores `_test.go` files by default and tells the auto-fix loop not to chase violations there|Normal|manual review: open the file, confirm the sentence is present|
