---
type: Spec Story
title: Fix instability.BuildGraph's absolute/relative path mismatch
description: Fix BuildGraph so package files are attributed to the correct import path when invoked with a relative path (the CLI default), removing the phantom catch-all package and the resulting abstractness.surfaceRatio panic
tags: [spec, architecture-checks, fix]
timestamp: 2026-08-23T00:00:00+09:00
---

# Fix instability.BuildGraph's absolute/relative path mismatch

## Value to user

Users can now safely invoke `boy-scout go all` (and `boy-scout go abstractness`, `boy-scout go instability`) without path arguments or with relative paths, and the tools will correctly attribute Go files to their packages instead of collapsing all files into a bogus catch-all package. This fixes a panic crash when a Zone-of-Pain package ends up with zero detected files due to the path mismatch.

## Completion criteria

`boy-scout go all` run from a module root with no path argument (defaults to `.`, the CLI's normal usage) completes without panic and correctly populates `Files` for all packages instead of losing files to a catch-all package.

## Spec

### Module and package scope
- Package import paths are resolved from absolute file paths, so `BuildGraph` behaves identically whether `paths` is given as `.` (the CLI default) or as an absolute path.
- `instability.BuildGraph` resolves each file's package directory to an absolute path before computing its import path relative to the (already-absolute) module root.
- Errors in `filepath.Abs` and `filepath.Rel` are asserted against, matching the existing convention in `internal/instability/instability.go` (which already asserts on module-line parsing and on `Ca+Ce > 0`).
- A package directory's own files are always attributed to that package's own import path — no directory's files can end up bucketed under a different package's import path, and no `moduleName + "/"` catch-all package is ever created unless the module genuinely has `.go` files directly at its root.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a temp module with `go.mod` (module `example.com/test`) and packages `pkg/a` (imports `pkg/b`) and `pkg/b`, and `BuildGraph` is called with `paths=["."]` while the process's cwd is the module root - When `BuildGraph` runs - Then `graph.Packages["example.com/test/pkg/a"].Files` and `graph.Packages["example.com/test/pkg/b"].Files` are both non-empty, and no key equal to `"example.com/test/"` exists in `graph.Packages`|Normal|unit test: `internal/instability/instability_test.go: TestBuildGraph_RelativePathResolvesCorrectPackages`|
|Given the same relative-path scenario, but `pkg/a` has zero exported interfaces/structs (so `Abstractness=0`) and zero callers/imports besides being imported once (so `Instability=0`, making it a Zone-of-Pain candidate) - When `abstractness.CheckWithSurfaceRatio` runs with `paths=["."]` from the module root - Then it returns a report with no panic, and the entry (if any) for `pkg/a` has a non-empty `Files`-derived `SurfaceRatio` computation (i.e. `surfaceRatio` was never called with an empty file list)|Exception|unit test: `internal/abstractness/abstractness_test.go: TestCheckWithSurfaceRatio_NoPanicWithRelativePath`|
|Given the CLI is invoked as `boy-scout go abstractness` with no path argument (defaults to `.`) from within a temp module fixture on disk that reproduces the Zone-of-Pain-with-empty-Files condition above - When run in-process via `run([]string{"go", "abstractness"}, ...)` (same pattern as `TestRun_GofunclenDefaultsToCurrentDir`) - Then it does not panic and returns a normal exit code (0, 1, or 2)|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_AbstractnessDefaultPathDoesNotPanic`|
|Given the CLI is invoked as `boy-scout go all` with no path argument against the same fixture, matching the exact command from the bug report (`./bin/boy-scout go all`) - When run - Then it does not panic and returns a normal exit code|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_AllDefaultPathDoesNotPanic`|
|Given all existing instability/abstractness tests, which always pass absolute paths via `t.TempDir()` - When the full suite runs after the fix - Then every existing test still passes unchanged (no regression from normalizing to absolute paths)|Boundary|full suite: `go test ./...`|
