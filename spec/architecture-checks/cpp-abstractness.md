---
type: Spec Story
title: cpp abstractness
description: Add boy-scout cpp abstractness check for Zone of Pain / Zone of Uselessness detection on per-file dependency graph
tags: [spec, architecture-checks]
timestamp: 2026-08-23T21:59:46+09:00
---

# cpp abstractness

## Value to user
Users can run `boy-scout cpp abstractness` on a C++ codebase to identify source files in two classic architecture smells: the **Zone of Pain** (files that are concrete/rigid and stable/depended-on — hard to change) and the **Zone of Uselessness** (files that are abstract/all-interfaces but unstable/churning — interfaces nobody's anchored to). This detects the two worst combinations of abstractness and instability per Robert Martin's Distance-from-Main-Sequence principle.

## Completion criteria
`boy-scout cpp abstractness [--min-distance=N] [--format=text|json] [--exclude-file=...] [paths...]` successfully scans C++ source files and reports files in the Pain/Uselessness zones; output format reuses `renderAbstractnessText`/`renderAbstractnessJSON` unchanged from the Go check.

## Spec
The `boy-scout cpp abstractness` subcommand:

### File and package scope
- Scans `.cpp`, `.h`, and `.hpp` files under the given paths
- Package unit = one source file (not a directory) — same as Story 1 (instability)
- Only files with ≥1 class or struct declaration are scored; files with 0 classes/structs are skipped (no divide-by-zero)
- Isolated files (0 includes, 0 dependents) are included in the analysis if they have classes; Instability is 0 or undefined by convention

### Abstractness formula
For a file F:
- **Abstract class/struct**: any `class` or `struct` declaration that contains ≥1 pure virtual method (declared with `= 0;`)
- **Concrete class/struct**: any `class` or `struct` declaration with 0 pure virtual methods
- `A` (abstractness) = (# abstract classes/structs) / (# abstract + # concrete)
  - If a file declares zero classes/structs, `A = 0` (no abstraction); file is skipped entirely
- `I` (instability) = `Ce / (Ca + Ce)` (computed from the dependency graph via Story 1)
- `signedD = A + I - 1` (ranges roughly -1 to 1; ideal at 0)
- `Distance = |signedD|`

### Zone classification
A file is flagged as a violation if:
- **Zone of Pain**: `signedD < -minDistance` (concrete-and-stable: no pure virtuals, many dependents, few dependencies)
- **Zone of Uselessness**: `signedD > minDistance` (abstract-and-unstable: all pure virtuals, few dependents, many dependencies)

Otherwise no violation is reported (ideal combos: abstract-stable or concrete-unstable).

### No surface-ratio or deep-module gate
This story does **not** implement a surface-ratio gate or deep-module gate. Every Pain or Uselessness candidate is reported unconditionally. A future story may introduce a gate once a stable definition of "public surface" exists for C++ (headers vs implementation, access specifiers, etc.).

### Output and options
- `--min-distance=N` (float, default 0.5): flag files with `|signedD| > minDistance`
- `--format=text|json` (default text): text format lists flagged files line-by-line, JSON includes full report
- `--exclude-file` (comma-separated globs): skip files matching patterns (same as other checks)
- `--exclude-func` (unused, present for consistency with other checks)
- Default path `.` if none given
- Files with syntax errors that tree-sitter can't parse are listed in `Skipped`, contribute no classes, and the run continues
- Files with 0 classes/structs do not appear in violations and are silently skipped
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority)

### Report structure
The report contains:
- `Violations`: list of `{ImportPath, Abstractness, Instability, Distance, Zone, SurfaceRatio}` tuples
  - `SurfaceRatio = 0` for all rows (gate not implemented)
- `Skipped`: list of files that couldn't be parsed
- `TotalPackages`: count of files with ≥1 class/struct declaration

Dependencies: tree-sitter C++ grammar via `github.com/smacker/go-tree-sitter/cpp` (CGO-based; requires C compiler). Reuses `cppinstability.BuildGraph` from Story 1.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given `shape.h` declares `class Shape { virtual void draw() = 0; };` (1 abstract, 0 concrete, Abstractness=1.0) with Instability=0.2 from the graph (signedD = 1.0+0.2-1=0.2, inside `min-distance=0.5` default) - When `boy-scout cpp abstractness testdata/cpp-abstractness/` runs - Then it reports 0 violations for `shape.h`|Normal|unit test: `internal/cppabstractness/cppabstractness_test.go`, integration test: `cmd/boy-scout/main_test.go`|
|Given `rigid.h` declares only `struct Rigid { int x; };` (0 abstract, 1 concrete, Abstractness=0.0) with Instability=0.1 (signedD = 0+0.1-1 = -0.9, `< -0.5`) - When run - Then it reports 1 violation for `rigid.h` with `Zone="Pain"`|Normal|unit test: `internal/cppabstractness/cppabstractness_test.go`, integration test: `cmd/boy-scout/main_test.go`|
|Given a file declaring 0 classes/structs at all (e.g. a free-function-only utility file) - When run - Then it's skipped from abstractness scoring entirely (no divide-by-zero), same as Go's `total==0` guard|Boundary|unit test: `internal/cppabstractness/cppabstractness_test.go`|
|Given a file that's a Zone-of-Pain candidate - When run with default flags (no gate implemented) - Then it's always reported, regardless of how small its public surface looks — confirms the "no gate" decision is actually implemented, not silently half-done|Boundary|unit test: `internal/cppabstractness/cppabstractness_test.go`|
|Given a built `boy-scout` binary and `testdata/cpp-abstractness/` fixture - When `boy-scout cpp abstractness testdata/cpp-abstractness/ --format=json` runs - Then the JSON matches the `rigid.h` Zone-of-Pain case above exactly|Normal|integration test: `cmd/boy-scout/main_test.go`|
