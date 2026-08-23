---
type: Spec Story
title: cpp instability
description: Add boy-scout cpp instability check for Ca/Ce dependency-direction violations via #include graph
tags: [spec, architecture-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# cpp instability

## Value to user
Users can run `boy-scout cpp instability` on a C++ codebase to identify source files that are stable (many things include them, they include little) but include files that are still unstable (churning, high coupling) — a classic architecture smell invisible without an explicit file-dependency graph. This helps prevent solid foundations from taking on volatile dependencies.

## Completion criteria
`boy-scout cpp instability [--min-gap=N] [--format=text|json] [--exclude-file=...] [paths...]` successfully scans C++ files and reports Ca/Ce dependency-direction violations between source files; output format matches the Go instability check's report shape, reusing `renderInstabilityText`/`renderInstabilityJSON`.

## Spec
The `boy-scout cpp instability` subcommand:

### File and package scope
- Scans `.cpp`, `.h`, and `.hpp` files under the given paths
- Package unit = one source file (not a directory, not a header/source pair) — a deliberate simplification vs. the Go check
- Only quoted `#include "..."` directives (in-project includes) count as edges
- Angle-bracket `#include <...>` (system/third-party includes) are silently ignored
- Include paths are resolved relative to the including file's directory using standard C/C++ semantics
- An include counts as an edge only if the resolved path matches a collected file
- Package keys are made stable by normalizing to absolute path first, then relative to the scan root (per `03-buildgraph-relative-path-crash` lesson)

### Instability formula
For a file F:
- `Ca` (afferent coupling) = count of other scanned files that include F
- `Ce` (efferent coupling) = count of other scanned files that F includes
- `I` (instability) = `Ce / (Ca + Ce)` (close to 0 = stable, close to 1 = unstable)

For every include edge `A -> B` (A includes B) between two scanned files:
- `Gap = I_B - I_A`
- It's a **violation** when `Gap > minGap` (default `minGap = 0`, i.e., any stable file depending on a less-stable one)

### Output and options
- `--min-gap=N` (float, default 0): print only violations with `Gap > minGap` (but summary stats always computed over all edges)
- `--format=text|json` (default text): text format lists violations line-by-line, JSON includes full report
- `--exclude-file` (comma-separated globs): skip files matching patterns (same as other checks)
- `--exclude-func` (unused, present for consistency with other checks)
- Default path `.` if none given
- Files with syntax errors that tree-sitter can't parse are listed in `Skipped`, contribute no edges, and the run continues
- Isolated files (0 includes, 0 dependents) do not appear in `graph.Packages` (only packages appearing in at least one edge are included, matching Go's behavior)
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority)

### Report structure
The report contains (same shape as Go instability):
- `Violations`: list of `{Source, Target, I_A, I_B, Gap}` tuples where `Gap > minGap`
- `Skipped`: list of files that couldn't be parsed
- `TotalEdges`: total count of include edges between scanned files (regardless of `--min-gap`)
- `ViolationRate`: (# edges with `Gap > 0`) / `TotalEdges` (not affected by `--min-gap`)
- `WeightedViolationRate`: (sum of `max(0, Gap)` over all edges) / `TotalEdges` (not affected by `--min-gap`)

Dependencies: tree-sitter C++ grammar via `github.com/smacker/go-tree-sitter/cpp` (CGO-based; requires C compiler).

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given `token.h` (no includes, included by `lexer.h`, `lexer.cpp`, `parser.cpp` — Ca=3, Ce=0, I=0) and `parser.cpp` (includes `parser.h` and `token.h`, included by nobody — Ca=0, Ce=2, I=1.0) with no direction violations - When `boy-scout cpp instability testdata/cpp-instability/` runs - Then it reports 0 violations|Normal|unit test: `internal/cppinstability/cppinstability_test.go`, integration test: `cmd/boy-scout/main_test.go`|
|Given the same fixture but `token.h` additionally does `#include "parser.h"` (I(token.h)=0.25 after the added edge) - When run - Then it reports exactly 1 violation: `token.h -> parser.h` with positive gap|Normal|unit test: `internal/cppinstability/cppinstability_test.go`|
|Given a file with `#include <vector>` and `#include "token.h"` - When run - Then only the `token.h` edge is counted; `<vector>` never appears as a package or edge|Boundary|unit test: `internal/cppinstability/cppinstability_test.go`|
|Given a `.cpp` file with 0 includes and 0 dependents (fully isolated) - When run - Then it does not appear in `graph.Packages` at all and does not break `ViolationRate`'s division|Boundary|unit test: `internal/cppinstability/cppinstability_test.go`|
|Given a file with a syntax error tree-sitter can't parse - When run - Then it's listed in `Skipped`, contributes no edges, and the run doesn't crash|Exception|unit test planned for future story (high confidence in tree-sitter robustness)|
|Given a built `boy-scout` binary and `testdata/cpp-instability/` fixture - When `boy-scout cpp instability testdata/cpp-instability/ --format=json` runs - Then the JSON output contains 0 violations and valid report structure|Normal|integration test: `cmd/boy-scout/main_test.go`|
