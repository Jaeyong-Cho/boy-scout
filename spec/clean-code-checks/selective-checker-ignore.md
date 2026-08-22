---
type: Spec Story
title: Selective per-checker gardener:ignore comment directive
description: Extend the `// gardener:ignore` doc-comment directive to target individual checkers (funclen/crap) instead of silencing both at once
tags: [spec, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---

# Selective per-checker gardener:ignore comment directive

## Value to user

Users can now silence a function for only one checker while keeping the other live, using `// gardener:ignore:funclen` or `// gardener:ignore:crap`. The bare `// gardener:ignore` still means "all checkers" — fully backward compatible.

## Completion criteria

- A function marked with `// gardener:ignore:funclen` is excluded only by the funclen checker, not by crap.
- A function marked with `// gardener:ignore:crap` is excluded only by the crap checker, not by funclen.
- Comma-separated checker lists (`// gardener:ignore:funclen,crap`) work correctly.
- Bare `// gardener:ignore` (no suffix) excludes from both checkers, unchanged from before.
- Unknown checker names are silently ignored (no error).
- Whitespace around checker names is trimmed; duplicates are a no-op; empty lists after the colon exclude nothing.

## Spec

Add an optional `:checker[,checker...]` suffix to the `// gardener:ignore` doc-comment directive. Semantics:

- **Bare directive** `// gardener:ignore` (no suffix): excludes the function from all checkers. Existing behavior, unchanged.
- **Named directive** `// gardener:ignore:funclen`, `// gardener:ignore:crap`, or `// gardener:ignore:funclen,crap`: excludes the function only from the named checker(s).
- **Checker names**: exact matches of CLI subcommand names (`funclen`, `crap`); case-sensitive, no aliasing.
- **Whitespace/duplicates**: names are comma-separated; whitespace around each name is trimmed; duplicate names are a no-op; an empty list after the colon (e.g. `// gardener:ignore:`) excludes nothing.
- **Unknown checker names**: silently inert, no error (same tolerance as existing `// gardenerignore` typo case).
- **Integration**: `gardener all`'s combined report reflects this per-section automatically — each checker still parses the directive independently, filtering only for its own name.

The parsing logic is extracted from `internal/funclen/funclen.go` and `internal/crap/crap.go` into a shared `internal/funcignore` package.

### Out of scope

- `--exclude-func`/`--exclude-file` flags remain shared across checkers under `gardener all` (unchanged).
- No new CLI flags.
- No new checkers beyond funclen/crap.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given a function violating funclen's limit with doc comment `// gardener:ignore:funclen` - When `funclen.Check` runs - Then no violation, `ExcludedFuncs` contains it with `Reason="comment"`|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludeFuncByCommentDirective_NamesThisChecker`|
|Given the same function - When `crap.Check` runs (directive names only `funclen`) - Then it IS reported as a violation, not excluded|Normal|unit test: `internal/crap/crap_test.go: TestCheck_ExcludeFuncByCommentDirective_NamesOtherCheckerOnly`|
|Given a function violating crap's threshold with doc comment `// gardener:ignore:crap` - When `crap.Check` runs - Then no violation, `ExcludedFuncs` contains it with `Reason="comment"`|Normal|unit test: `internal/crap/crap_test.go: TestCheck_ExcludeFuncByCommentDirective_NamesThisChecker`|
|Given the same function - When `funclen.Check` runs (directive names only `crap`) - Then it IS reported as a violation, not excluded|Normal|unit test: `internal/funclen/funclen_test.go: TestCheck_ExcludeFuncByCommentDirective_NamesOtherCheckerOnly`|
|Given doc comment `// gardener:ignore:funclen,crap` (comma list) - When `funcignore.Reason` runs once for checker `"funclen"` and once for `"crap"` - Then both calls return `excluded=true, reason="comment"`|Normal|unit test: `internal/funcignore/funcignore_test.go: TestReason_CommaListMatchesEitherChecker`|
|Given bare `// gardener:ignore` (no checker name, existing directive) - When either checker's `Check` runs - Then both still exclude it, unchanged from before this story|Normal|unit test (existing, must still pass unmodified): `internal/funclen/funclen_test.go: TestCheck_ExcludeFuncByCommentDirective`, `internal/crap/crap_test.go: TestCheck_ExcludeFuncByCommentDirective`|
|Given `// gardener:ignore: funclen , crap ` (extra whitespace around names) - When `funcignore.Reason` runs - Then names are trimmed before matching and both checkers match|Boundary|unit test: `internal/funcignore/funcignore_test.go: TestReason_WhitespaceAroundNamesIsTrimmed`|
|Given `// gardener:ignore:funclen,funclen` (duplicate name) - When `funcignore.Reason` runs for checker `"funclen"` - Then `excluded=true`, no error from the duplicate|Boundary|unit test: `internal/funcignore/funcignore_test.go: TestReason_DuplicateCheckerNameInListIsNoOp`|
|Given `// gardener:ignore:` (colon, empty list) - When `funcignore.Reason` runs for any checker name - Then `excluded=false`|Boundary|unit test: `internal/funcignore/funcignore_test.go: TestReason_EmptyListAfterColonExcludesNothing`|
|Given `// gardener:ignore:fooey` (checker name not recognized by this tool) - When `funcignore.Reason` runs for checker `"funclen"` or `"crap"` - Then `excluded=false`, no error raised|Exception|unit test: `internal/funcignore/funcignore_test.go: TestReason_UnknownCheckerNameExcludesNothing`|
|Given `// gardenerignore` (typo, missing colon, existing behavior) - When either checker's `Check` runs - Then it is NOT excluded, unchanged from before this story|Exception|unit test (existing, must still pass unmodified): `internal/funclen/funclen_test.go: TestCheck_CommentDirectiveTypoIsNotExcluded`, `internal/crap/crap_test.go: TestCheck_CommentDirectiveTypoIsNotExcluded`|
|Given a function with `// gardener:ignore:crap` that violates both funclen's limit and crap's threshold - When `gardener all --format=json <path>` runs - Then the `funclen` section of the JSON reports it as a violation and the `crap` section does not|Normal|integration test: `cmd/gardener-go/main_test.go: TestRun_AllRespectsPerCheckerIgnoreComment`|
