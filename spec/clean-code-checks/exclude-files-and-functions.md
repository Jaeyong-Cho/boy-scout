---
type: Spec Story
title: exclude-files-and-functions
description: Add --exclude-file, --exclude-func, --debug flags and // gardener:ignore comment directive to funclen/crap/all
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# exclude-files-and-functions

## Value to user
Users can suppress noisy clean-code violations by excluding entire files via glob patterns, or by excluding individual functions via glob patterns or a comment directive, and optionally see which items were excluded via a `--debug` flag.

## Completion criteria
`gardener funclen`, `gardener crap`, and `gardener all` accept `--exclude-file`, `--exclude-func`, and `--debug` flags, and respect the `// gardener:ignore` comment directive on functions, such that excluded items never appear as violations and never affect the exit code.

## Spec

Add two exclude mechanisms to `go-gardener` (funclen + crap + all): a `--exclude-file` glob flag (skips whole files) and a `--exclude-func` glob flag plus a `// gardener:ignore` doc-comment directive (skips one function). Excluded items never appear in `Violations`/`Skipped` and never change the exit code. A new `--debug` flag is the only way to see what got excluded (new `ExcludedFiles`/`ExcludedFuncs` report fields, empty unless `--debug` is passed).

### Spec changes

- Add `--exclude-file` flag (comma-separated glob patterns, e.g. `*_test.go,internal/mocks/*.go`) to `funclen`, `crap`, `all` — a file matching any pattern (against its full relative path OR its basename) is skipped entirely, before parsing.
- Add `--exclude-func` flag (comma-separated glob patterns matched against the bare function name, e.g. `Test*`) to `funclen`, `crap`, `all` — a matching function produces no violation.
- Add the `// gardener:ignore` doc-comment directive — a function with this exact comment (any leading/trailing whitespace after stripping `//`) directly in its doc comment is excluded, independent of any flag.
- Add `--debug` flag to `funclen`, `crap`, `all` — when set, the report's `ExcludedFiles`/`ExcludedFuncs` fields are populated and printed; when unset (default) they stay empty and print nothing.
- Excluded files/functions never enter `Report.Violations` or `Report.Skipped`, and never affect the 0/1/2 exit code — only real violations and real skip-errors do.
- A malformed `--exclude-file`/`--exclude-func` glob pattern is a usage error: message to stderr, exit code 2, checked before any file is scanned.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given a directory with `foo.go` and `foo_test.go` and pattern `*_test.go` - When `gofiles.Collect(paths, []string{"*_test.go"})` runs - Then `excluded` contains `foo_test.go`, `files` contains only `foo.go`|Normal|unit test: `internal/gofiles/gofiles_test.go: TestCollect_ExcludesFilesMatchingGlobPattern`|
|Given a file at `pkg/foo_test.go` and pattern `*_test.go` (no slash) - When `Collect` runs - Then it still matches by basename, not just full path|Boundary|unit test: `internal/gofiles/gofiles_test.go: TestCollect_ExcludeGlobMatchesByBasenameOrFullPath`|
|Given pattern `*.nomatch` that matches nothing in the tree - When `Collect` runs - Then `excluded` is empty and no error/skip is produced|Boundary|unit test: `internal/gofiles/gofiles_test.go: TestCollect_ExcludePatternMatchingNothingIsNoOp`|
|Given a file matched by `Options.ExcludeFiles` - When `funclen.Check` runs with `Options.Debug=true` - Then no violation is reported for that file and it appears in `Report.ExcludedFiles`|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug`|
|Given the same excluded file/function - When `Options.Debug=false` (default) - Then `Report.ExcludedFiles`/`ExcludedFuncs` are both empty, even though the file/function was still skipped|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludedItemsHiddenUnlessDebug`|
|Given a function `TestHelper` and `Options.ExcludeFuncs=[]string{"Test*"}` - When `funclen.Check` runs (debug on) - Then it's not a violation and appears in `ExcludedFuncs` with `Reason="flag"`|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludeFuncByNamePattern`|
|Given a function with doc comment `// gardener:ignore` and no matching flag pattern - When `funclen.Check` runs (debug on) - Then it's not a violation and appears in `ExcludedFuncs` with `Reason="comment"`|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludeFuncByCommentDirective`|
|Given a function with doc comment `// gardenerignore` (typo, missing colon) - When `funclen.Check` runs - Then it is NOT excluded and is evaluated normally|Exception|unit test: `internal/funclen/funclen_test.go: TestCheck_CommentDirectiveTypoIsNotExcluded`|
|Given the same 4 behaviors above (file exclude, func-name exclude, comment directive, debug gating) - When run against `crap.Check` instead of `funclen.Check` - Then they behave identically|Normal|unit test: `internal/crap/crap_test.go: TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug`, `TestCheck_ExcludedItemsHiddenUnlessDebug`|
|Given `gardener funclen --exclude-file="*_test.go" --exclude-func="Test*" <path>` against a fixture with a matching file and a matching function - When run - Then stdout has no violation lines for either, and exit code reflects only the remaining real violations|Normal|unit test: `cmd/gardener/main_test.go: TestRun_ExcludeFlagsFilterFunclenOutput`|
|Given `gardener funclen --exclude-file="[" <path>` (malformed glob) - When run - Then stderr contains a clear error, exit code is 2, and no file is scanned|Exception|unit test: `cmd/gardener/main_test.go: TestRun_MalformedExcludePatternExitsWithUsageError`|
|Given `gardener funclen --debug --exclude-file="*_test.go" <path>` with a matching file present - When run - Then stdout includes an "excluded file" line naming it|Normal|unit test: `cmd/gardener/main_test.go: TestRun_DebugFlagShowsExcludedFilesAndFuncs`|
|Given `gardener all --exclude-file=... --exclude-func=...` - When run - Then both the `[funclen]` and `[crap]` sections of the combined output respect the same excludes|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllRespectsExcludeFlags`|
|Given a file that would otherwise violate funclen's limit, fully excluded via `--exclude-file` - When run with zero other violations/skips - Then exit code is 0 (clean), not 1|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_ExcludedViolationsDoNotAffectExitCode`|
