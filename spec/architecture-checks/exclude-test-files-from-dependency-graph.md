---
type: Spec Story
title: exclude-test-files-from-dependency-graph
description: Fix instability.BuildGraph to exclude _test.go files from Ca/Ce/I dependency graph
tags: [spec, architecture-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# Exclude `_test.go` files from dependency graph

## Value to user
Fixes two bugs where test-only code was being miscounted as production architecture:
1. A package imported only from test code (e.g., a mock/testutil helper) was inflating another package's coupling metrics.
2. `TestXxx` functions in `_test.go` files were being counted as "exported API surface" because they start with a capital letter, sometimes wrongly flagging legitimate deep modules as violations.

Both bugs share one root cause: `instability.BuildGraph` was collecting every `.go` file in a package, including `_test.go` files, when computing `Ca`/`Ce`/`Instability`. The fix is to exclude `_test.go` files from the file list, which automatically corrects both `instability` and `abstractness` checks (since `abstractness` reuses `BuildGraph`'s file list).

## Completion criteria
- `boy-scout go instability` no longer counts imports from `_test.go` files in the graph.
- `boy-scout go abstractness` no longer counts `TestXxx`/`BenchmarkXxx`/`ExampleXxx` functions as exported symbols.
- All existing tests pass; two new unit tests and two new integration tests confirm the fix.

## Spec

### The fix
`instability.BuildGraph` excludes any file whose name ends in `_test.go` when building the package-import graph:
- Files are skipped during the parse loop — neither their imports nor their declarations are counted.
- A package that is only ever imported from test files loses that edge entirely (no `Ca` contribution).
- If a package ends up with zero edges, it does not appear in `graph.Packages` (same rule as today).

### Effect on instability
- Modules that were only imported from test code are no longer counted as dependencies.
- `Ca` (afferent coupling) is computed only from production-file imports.
- `Ce` (efferent coupling) is computed only from production-file imports.
- Overall graph size (node count, edge count) shrinks when test-only dependencies existed.

### Effect on abstractness
- `abstractness.go` reuses `graph.Packages[...].Files` from `BuildGraph`, so it automatically inherits the fix.
- No separate code change needed in `abstractness.go`.
- `TestXxx`, `BenchmarkXxx`, and `ExampleXxx` functions are never counted toward `SurfaceRatio` or `Abstractness` (because `_test.go` files are never in the file list).

### Scope exclusions (out of this story)
- Build-tag-aware file selection (e.g., `//go:build !windows`): files are partition-able by build constraints; this story does not implement constraint parsing.
- Symbol-usage weighting: files are excluded wholesale; this story does not track which symbols from test-only imports are actually used.
- Non-Go languages: this story is Go-only.

## Acceptance criteria

|AC|Category|Verification Method|
|--|--|--|
|Given package `order` has `order.go` importing `payment` and `order_test.go` importing `mocks` - When `instability.BuildGraph` runs - Then the graph has exactly 1 edge (`order -> payment`) and no edge to `mocks`|Normal|unit test: `internal/instability/instability_test.go: TestBuildGraph_ExcludesTestFileImports`|
|Given the same fixture - When run as `boy-scout go instability --format=json` via the CLI - Then `TotalEdges == 1` and the JSON output contains no reference to `mocks`|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_InstabilityIgnoresTestOnlyImports`|
|Given package `deepcache` (3 exported types, ~21 unexported funcs/types, real `SurfaceRatio` ≈ 0.1) with 25 `TestBehaviorN` funcs in `deepcache_test.go`, and 8 caller packages (`Ca=8, Ce=0`) - When `abstractness.CheckWithSurfaceRatio` runs with default `--min-surface-ratio=0.5` - Then `deepcache` is NOT in `Report.Violations` (test funcs excluded, real ratio stays under the gate)|Normal|unit test: `internal/abstractness/abstractness_test.go: TestCheck_TestFuncsNotCountedAsExported`|
|Given the same `deepcache` fixture - When run as `boy-scout go abstractness --format=json` via the CLI - Then `Violations` is empty and exit code is 0|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_AbstractnessIgnoresTestFuncsAsExported`|
|Given a package directory containing only a `_test.go` file (e.g. `foo_export_test.go` declaring `package foo_test`) and no production `.go` file - When `instability.BuildGraph` runs - Then that directory contributes zero files/imports and never appears as a node in the graph (no panic, no special-case code needed)|Boundary|unit test: `internal/instability/instability_test.go: TestBuildGraph_TestOnlyDirectoryProducesNoNode`|
|Given all existing instability/abstractness fixtures that use only production `.go` files (no `_test.go`) - When the full test suite runs after the fix - Then every existing test in `internal/instability/instability_test.go` and `internal/abstractness/abstractness_test.go` still passes unchanged (no regression)|Normal|full suite: `go test ./...`|
