---
type: Spec Story
title: instability
description: Add boy-scout go instability check for Ca/Ce dependency-direction violations
tags: [spec, architecture-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# instability

## Value to user
Users can run `boy-scout go instability` on a Go module to identify packages that are stable (many things depend on them, they depend on little) but depend on packages that are still unstable (churning, high coupling) — a classic architecture smell invisible without an explicit package-import graph. This helps prevent solid foundations from taking on volatile dependencies.

## Completion criteria
`boy-scout go instability [--min-gap=N] [--format=text|json] [--exclude-file=...] [paths...]` successfully scans a Go module and reports Ca/Ce dependency-direction violations between first-party packages; `boy-scout go all` includes the instability check alongside existing checks.

## Spec
The `boy-scout go instability` subcommand:

### Module and package scope
- Scans exactly one Go module per invocation (the nearest `go.mod` found by walking up from the first scanned path)
- Module unit = Go package (a directory of `.go` files under one import path)
- Only first-party packages count (packages under the scanned module's own `go.mod` `module` path)
- Stdlib and third-party imports are invisible to the graph — not counted toward instability, never appear as nodes
- `_test.go` files are excluded from both the file list and the import scan for a package
- Package import paths are resolved from absolute file paths, so `BuildGraph` behaves identically whether `paths` is given as `.` (the CLI default) or as an absolute path

### Instability formula
For a package P:
- `Ca` (afferent coupling) = count of other first-party packages that import P
- `Ce` (efferent coupling) = count of other first-party packages that P imports
- `I` (instability) = `Ce / (Ca + Ce)` (close to 0 = stable, close to 1 = unstable)

For every import edge `A -> B` (A imports B) between two first-party packages:
- `Gap = I_B - I_A`
- It's a **violation** when `Gap > minGap` (default `minGap = 0`, i.e., any stable package depending on a less-stable one)

### Output and options
- `--min-gap=N` (float, default 0): print only violations with `Gap > minGap` (but summary stats always computed over all edges)
- `--format=text|json` (default text): text format lists violations line-by-line, JSON includes full report
- `--exclude-file` (comma-separated globs): skip files matching patterns (same as other checks)
- `--exclude-func` (unused, present for consistency with other checks)
- Default path `.` if none given
- Skips files that fail to parse (syntax errors anywhere in the file, not just the import block) with a warning, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority)

### Report structure
The report contains:
- `Violations`: list of `{Source, Target, I_A, I_B, Gap}` tuples where `Gap > minGap`
- `Skipped`: list of files that couldn't be parsed
- `TotalEdges`: total count of import edges between first-party packages (regardless of `--min-gap`)
- `ViolationRate`: (# edges with `Gap > 0`) / `TotalEdges` (not affected by `--min-gap`)
- `WeightedViolationRate`: (sum of `max(0, Gap)` over all edges) / `TotalEdges` (not affected by `--min-gap`)

The `boy-scout go all` subcommand includes instability alongside gofunclen, crap, and filelen checks.

Dependencies: Go stdlib only. No third-party modules.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given package `httpapi` (Ca=0, Ce=4, I=1.0) imports package `domain` (Ca=5, Ce=0, I=0.0) - When `instability.Check` runs - Then `Report.Violations` is empty (I_A ≥ I_B follows the expected direction)|Normal|unit test: `internal/instability/instability_test.go: TestCheck_NoViolationWhenUnstableDependsOnStable`|
|Given package `domain` (Ca=5, Ce=0, I=0.0) imports package `httpapi` (Ca=0, Ce=4, I=1.0) - When `instability.Check` runs - Then `Report.Violations` has one entry `domain -> httpapi`, `Gap=1.0`|Normal|unit test: `internal/instability/instability_test.go: TestCheck_ReportsViolationWhenStableDependsOnUnstable`|
|Given two packages `C` (Ca=2, Ce=2, I=0.5) and `D` (Ca=1, Ce=1, I=0.5) with edge `C -> D` - When checked - Then `Gap=0.0` and no violation is reported (boundary: `I_A < I_B` is strict)|Boundary|unit test: `internal/instability/instability_test.go: TestCheck_EqualInstabilityIsCompliant`|
|Given a violating edge with `Gap=0.05` - When checked with default `--min-gap=0` then re-checked with `--min-gap=0.1` - Then the violation list has 1 entry, then 0 entries — but `ViolationRate`/`WeightedViolationRate` are identical in both runs|Boundary|unit test: `internal/instability/instability_test.go: TestCheck_MinGapFiltersListNotSummary`|
|Given a Go file with invalid syntax among otherwise-valid files - When `instability.Check` runs - Then that file is recorded in `Report.Skipped` and the rest of the module is still checked|Exception|unit test: `internal/instability/instability_test.go: TestCheck_SkipsUnparsableFile`|
|Given a path with no `go.mod` anywhere above it - When `instability.Check` runs - Then it returns a non-nil error and an empty `Report`|Exception|unit test: `internal/instability/instability_test.go: TestCheck_ErrorsWithoutGoMod`|
|Given a module where no package imports any other first-party package - When `instability.Check` runs - Then `Report` has 0 edges, 0 violations, `ViolationRate=0`, no panic|Boundary|unit test: `internal/instability/instability_test.go: TestCheck_NoInternalEdgesIsClean`|
|Given the `domain`/`httpapi` violating fixture - When run as `boy-scout go instability --format=json` via the CLI - Then stdout is valid JSON containing the violation and exit code is 1|Normal|unit test: `cmd/boy-scout/main_test.go: TestRunGoInstability_JSONOutput`|
|Given the same violating fixture - When run as `boy-scout go all` - Then the combined report's `instability` field contains the violation and the combined exit code is 1|Normal|unit test: `cmd/boy-scout/main_test.go: TestRunGoAll_IncludesInstability`|
