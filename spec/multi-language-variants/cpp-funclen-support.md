---
type: Spec Story
title: cpp-funclen-support
description: Add gardener cpp funclen (C++ function-length check) as the first non-Go language behind the unified gardener {lang} {command} dispatch
tags: [spec, multi-language-variants, cpp]
timestamp: 2026-08-22T21:29:19+09:00
---

# cpp-funclen-support

## Value to user
`gardener cpp funclen` brings function-length checking to C++ codebases via a tree-sitter C++ parser, extending the multi-language dispatch architecture to its first non-Go language. CRAP/coverage checking and `gardener:ignore` comment support are deferred to future stories.

## Completion criteria
`gardener cpp funclen [--max-lines N] [--format text|json] [--exclude-file PATTERNS] [--exclude-func PATTERNS] [--debug] [paths...]` is wired and functional, with funclen-only scope (no CRAP/coverage, no `gardener:ignore` for C++). All 110 tests pass (107 baseline + 7 cppfunclen + 3 CLI-level tests).

## Spec

### CLI surface
- `gardener cpp funclen [--max-lines N] [--format text|json] [--exclude-file PATTERNS] [--exclude-func PATTERNS] [--debug] [paths...]` added, mirroring `gardener go funclen`'s flags and `Report`/`Violation` JSON shape exactly.
- Scans `.cpp`/`.h`/`.hpp` files under the given paths (default `.`), same vendor/dot-dir pruning and glob-exclude semantics as the Go scanner.
- A function is anything with a body (free function, member function in-class or out-of-line, constructor/destructor, operator overload); bodyless declarations are never counted; nested lambda lines count toward the enclosing function.
- Out-of-line member functions report `Class::method` as `Func`; everything else reports the bare name.
- A file with any parse error (tree-sitter ERROR node) is added to `Skipped` in full — none of its functions are evaluated.
- No `gardener:ignore` comment support and no CRAP/coverage check for C++ — explicitly out of scope, documented so it isn't mistaken for an oversight.
- `gardener cpp crap` / `gardener cpp all` are not registered — calling them hits the "unknown subcommand for cpp" path, not a silent no-op.

### Implementation details
- Parser: tree-sitter C++ grammar via `github.com/smacker/go-tree-sitter` and its `cpp` subpackage (CGO-based; requires C compiler on machine running `gardener cpp`).
- File discovery generalized: `internal/gofiles` renamed to `internal/srcfiles`, `Collect` signature changed to accept extensions as a parameter (`Collect(paths []string, extensions []string, excludePatterns []string)`), call sites in `funclen` and `crap` updated to pass `[]string{".go"}`.
- New package `internal/cppfunclen` (copies shape of `internal/funclen` but as fresh, separate code — no shared imports).
- Function name extraction: reads qualified identifier from tree-sitter function declarator node; falls back to bare identifier for free functions and lambdas.
- Error detection: walks parsed tree for ERROR nodes before function evaluation; if found, skips whole file and adds to Skipped.

### Skill documentation
- Updated `internal/setup/skill.md` to note C++ support: "For C++ codebases, run `gardener cpp funclen` instead (C++ has no CRAP check and no `gardener:ignore` comment support yet)."

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a `.cpp` file with a function body longer than `--max-lines` (default 50) - When `gardener cpp funclen <dir>` runs - Then it reports one `Violation{File,Line,Func,Length,Limit}` and exits 1|Normal|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_ReportsViolationForOverLimitFunction`; CLI-level: `cmd/gardener/main_test.go: TestRun_CppFunclenReportsViolation`|
|Given a `.hpp` file with only bodyless declarations (`void foo();`) - When run - Then zero violations|Boundary|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_SkipsBodylessDeclarations`|
|Given a directory with zero `.cpp`/`.h`/`.hpp` files - When `gardener cpp funclen <dir>` runs - Then it exits 0 with empty `Violations`/`Skipped`|Boundary|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_ZeroMatchingFilesReturnsEmptyReport`|
|Given a function with a nested lambda whose body pushes the enclosing function's line count over the limit - When run - Then the violation is attributed to the enclosing function only, no separate violation for the lambda|Boundary|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_AttributesLambdaLinesToEnclosingFunction`|
|Given `Widget::resize() { ... }` defined out-of-line and over the limit - When run - Then `Func` is reported as `Widget::resize`|Normal|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_QualifiesOutOfLineMethodName`|
|Given `--exclude-func "*Test"` and a matching over-limit function - When run - Then it's excluded from `Violations` (and listed in `ExcludedFuncs` with `Reason: "flag"` only under `--debug`)|Normal|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_ExcludeFuncFlagFiltersByGlobName`|
|Given a `.cpp` file with a genuine syntax error (tree-sitter ERROR node) - When run - Then the whole file is added to `Skipped{File,Error}` and none of its functions are evaluated, process exits 2 (skipped takes priority over violations, same convention as Go)|Exception|unit test: `internal/cppfunclen/cppfunclen_test.go: TestCheck_SyntaxErrorFileIsSkippedEntirely`; CLI-level: `cmd/gardener/main_test.go: TestRun_CppSyntaxErrorFileSkippedExitsTwo`|
|Given `gardener cpp crap` or `gardener cpp all` - When run - Then exit 2 with "unknown subcommand for cpp: crap" (not a silent no-op)|Exception|unit test: `cmd/gardener/main_test.go: TestRun_CppUnregisteredSubcommandExitsUsageError`|
|Given a `.go` file sitting in the same directory as `.cpp` files - When `gardener cpp funclen <dir>` runs - Then the `.go` file is silently ignored, not scanned or errored on|Boundary|unit test: `internal/srcfiles/srcfiles_test.go: TestCollect_FiltersByGivenExtensions`|
|Given `gardener go funclen` and `gardener cpp funclen` both exist - When either runs on its own file type - Then `internal/srcfiles.Collect` is the single shared file-discovery implementation for both (extensions passed as a parameter, not a second copy of the walker)|Normal|unit test: existing `internal/srcfiles/srcfiles_test.go` cases pass unmodified for `.go`, new case passes for `.cpp`/`.h`/`.hpp`|
|Given the refactored `internal/srcfiles` package (renamed from `internal/gofiles`) - When all tests run - Then 110 tests pass (99 baseline + 7 cppfunclen + 3 CLI-level + 1 srcfiles extension test), all green|Normal|`go build ./... && go test ./...`|
