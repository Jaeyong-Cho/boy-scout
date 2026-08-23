---
type: Spec Story
title: abstractness
description: Add boy-scout go abstractness check for Zone of Pain / Zone of Uselessness detection
tags: [spec, architecture-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# abstractness

## Value to user
Users can run `boy-scout go abstractness` on a Go module to identify packages in two classic architecture smells: the **Zone of Pain** (packages that are concrete/rigid and stable/depended-on — hard to change) and the **Zone of Uselessness** (packages that are abstract/all-interfaces but unstable/churning — interfaces nobody's anchored to). This detects the two worst combinations of abstractness and instability per Robert Martin's Distance-from-Main-Sequence principle.

## Completion criteria
`boy-scout go abstractness [--min-distance=N] [--format=text|json] [--exclude-file=...] [paths...]` successfully scans a Go module and reports packages in the Pain/Uselessness zones; `boy-scout go all` includes the abstractness check alongside existing checks.

## Spec
The `boy-scout go abstractness` subcommand:

### Module and package scope
- Scans exactly one Go module per invocation (same as instability check)
- Only first-party packages with at least one internal import edge are analyzed
- Packages with no internal edges (isolated, `Ca = Ce = 0`) are skipped — Instability undefined

### Abstractness formula
For a package P:
- `A` (abstractness) = (# exported `interface` type declarations) / (# exported `interface` + # exported `struct` type declarations)
  - Only capitalized (exported) names count
  - Other type kinds (aliases, func types, etc.) are ignored from both count and denominator
  - If a package declares zero exported interfaces/structs, `A = 0` (no abstraction)
- `I` (instability) = `Ce / (Ca + Ce)` (computed from the graph, same as instability check)
- `signedD = A + I - 1` (ranges roughly -1 to 1; ideal at 0)
- `Distance = |signedD|`

### Zone classification
A package is flagged as a violation if:
- **Zone of Pain**: `signedD < -minDistance` (concrete-and-stable: no interfaces, many dependents, few dependencies)
- **Zone of Uselessness**: `signedD > minDistance` (abstract-and-unstable: all interfaces, few dependents, many dependencies)

Otherwise no violation is reported (ideal combos: abstract-stable or concrete-unstable).

### Output and options
- `--min-distance=N` (float, default 0.5): flag packages with `|signedD| > minDistance` (packages must be clearly closer to a bad corner than the middle to get flagged)
- `--format=text|json` (default text): text format lists flagged packages line-by-line, JSON includes full report
- `--exclude-file` (comma-separated globs): skip files matching patterns (same as other checks)
- `--exclude-func` (unused, present for consistency with other checks)
- Default path `.` if none given
- Skips files that fail to parse with a warning, continues checking the rest
- Reports exit code 0 if clean, 1 if violations found, 2 if any file was skipped (takes priority)

### Report structure
The report contains:
- `Violations`: list of `{ImportPath, Abstractness, Instability, Distance, Zone}` tuples where `|signedD| > minDistance`
- `Skipped`: list of files that couldn't be parsed
- `TotalPackages`: count of packages with a computable Instability (those appearing in at least one edge)

The `boy-scout go all` subcommand includes abstractness alongside gofunclen, crap, filelen, and instability checks.

Dependencies: Go stdlib only. No third-party modules.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given package `legacydb` (6 exported structs, 0 interfaces → A=0.0; Ca=8, Ce=0 → I=0.0) - When `abstractness.Check` runs - Then `Report.Violations` has one entry, `Zone="Pain"`, `Distance=1.0`|Normal|unit test: `internal/abstractness/abstractness_test.go: TestCheck_FlagsZoneOfPain`|
|Given package `plugini` (3 exported interfaces, 0 structs → A=1.0; Ca=0, Ce=3 → I=1.0) - When `abstractness.Check` runs - Then `Report.Violations` has one entry, `Zone="Uselessness"`, `Distance=1.0`|Normal|unit test: `internal/abstractness/abstractness_test.go: TestCheck_FlagsZoneOfUselessness`|
|Given package `stableiface` (all exported types are interfaces → A=1.0; Ca=5, Ce=0 → I=0.0, `signedD=0`) - When checked - Then it is not in `Report.Violations` (ideal: abstract and stable)|Normal|unit test: `internal/abstractness/abstractness_test.go: TestCheck_NoViolationWhenAbstractAndStable`|
|Given package `leafimpl` (0 interfaces, some structs → A=0.0; Ca=0, Ce=4 → I=1.0, `signedD=0`) - When checked - Then it is not in `Report.Violations` (ideal: concrete and unstable)|Normal|unit test: `internal/abstractness/abstractness_test.go: TestCheck_NoViolationWhenConcreteAndUnstable`|
|Given a package with `signedD` exactly `-0.5` (default `--min-distance=0.5`) - When checked - Then it is not flagged (boundary: strictly greater distance required, exactly-at-limit is compliant)|Boundary|unit test: `internal/abstractness/abstractness_test.go: TestCheck_ExactlyAtMinDistanceIsCompliant`|
|Given a package with 0 exported interfaces and 0 exported structs (pure-function package) - When checked - Then `A=0` is used (no divide-by-zero), not skipped or errored|Boundary|unit test: `internal/abstractness/abstractness_test.go: TestCheck_ZeroExportedTypesUsesAZero`|
|Given a package with no internal edges (isolated, `Ca=Ce=0`) - When checked - Then it does not appear in `Report.Violations` or count toward `Report.TotalPackages` (Instability undefined, same rule as instability check)|Boundary|unit test: `internal/abstractness/abstractness_test.go: TestCheck_IsolatedPackageSkipped`|
|Given a Go file with invalid syntax in a package that would otherwise be a Pain-zone candidate - When checked - Then that file is recorded in `Report.Skipped` and the rest of the module is still checked|Exception|unit test: `internal/abstractness/abstractness_test.go: TestCheck_SkipsUnparsableFile`|
|Given the `legacydb` fixture - When run as `boy-scout go abstractness --format=json` via the CLI - Then stdout is valid JSON containing the Pain-zone entry and exit code is 1|Normal|unit test: `cmd/boy-scout/main_test.go: TestRunGoAbstractness_JSONOutput`|
|Given the `legacydb` fixture - When run as `boy-scout go all` - Then the combined report's `abstractness` field contains the Pain-zone entry and the combined exit code is 1|Normal|unit test: `cmd/boy-scout/main_test.go: TestRunGoAll_IncludesAbstractness`|
