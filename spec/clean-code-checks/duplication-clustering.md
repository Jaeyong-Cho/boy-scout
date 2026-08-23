---
type: Spec Story
title: duplication-clustering
description: Group N-way function duplicates into Clusters so a 12-way copy-paste reports as 1 fixable group instead of C(N,2) separate pairwise Violations
tags: [spec, clean-code-checks]
timestamp: 2026-08-24T00:00:00+09:00
---

# duplication-clustering

## Value to user

When `boy-scout go duplication` detects multiple copies of the same function (e.g. the same helper duplicated across 12 packages), instead of reporting ~66 pairwise violation lines (C(12,2)), it now groups all 12 into one Cluster entry, making the result scannable at scale and actionable: "12 functions form one duplicate group" is a single fix target, whereas 66 line pairs require manual effort to see the pattern.

## Completion criteria

`boy-scout go duplication` output gains a `Clusters` array in both text and JSON formats, grouping every `Violation` by connected component (union-find keyed by `file:line:func`); the existing pairwise `Violations` array stays unchanged for backward compatibility, but now represents the raw results that feed into `Clusters` for summarization.

## Spec

### Clustering model

- A `Cluster` is a maximal group of functions where every member duplicates at least one other member in the group (via the transitive closure of the pairwise `Violations` graph)
- Clustering is computed via union-find after pairwise violation detection, keyed by each function's `(file, line, func)` identity
- Every cluster is guaranteed to have >= 2 members (singleton groups are unreachable: every violation connects exactly 2 distinct functions)

### Cluster data structure

- `FuncRef`: a unique function identity: `{file, line, func}` (JSON fields same names)
- `Cluster`: 
  - `Members []FuncRef` — all unique functions in this group, sorted stable by file then line
  - `Pairs []Violation` — every pairwise violation whose both endpoints are in this group, keeping each pair's own `Type`/`DupLines`/`Similarity`
  - `DupLines int` — sum of every `Pairs[i].DupLines` (total removable debt if all duplicates collapse into one)
  - `CrossPackage bool` — true if members span multiple different directory paths (inferred from `filepath.Dir(member.File)` comparison)

### Report changes (additive only)

- `duplication.Report` gains new field `Clusters []Cluster` (computed from `Violations` after pairwise detection)
- `Report.Violations` remains **unchanged in format and content** (zero impact on the ~26 existing duplication tests; this is new data flow, not refactoring of existing data)
- Ordering: `Clusters` is sorted by `DupLines` descending (ties broken by first member's file then line) so consumers can take the first N without re-sorting

### Text output (`--format=text`)

- Every existing pairwise violation line stays **byte-for-byte unchanged**
- One new summary line per cluster is appended after all violations: 
  - Format: `[prefix]N functions clustered as one duplicate group (M duplicated lines total, cross-package: [true|false])`
  - Example: `[duplication] 12 functions clustered as one duplicate group (84 duplicated lines total, cross-package: true)`

### JSON output (`--format=json`)

- Top-level `"clusters"` array is added to the duplication report
- No changes to existing `"violations"` array
- Example cluster entry:
```json
{
  "members": [
    {"file": "pkg1/helper.go", "line": 5, "func": "assertf"},
    {"file": "pkg2/helper.go", "line": 10, "func": "assertf"},
    ...
  ],
  "pairs": [
    {"fileA": "pkg1/helper.go", "lineA": 5, "funcA": "assertf", "fileB": "pkg2/helper.go", "lineB": 10, "funcB": "assertf", "type": "Type-2", "dupLines": 6, "similarity": 1.0},
    ...
  ],
  "dupLines": 84,
  "crossPackage": true
}
```

## Acceptance criteria

|AC|Category|Verification Method|
|--|--|--|
|Given 3 functions where A≡B is Type-1 and B≈C is Type-2 (renamed), all in the same package - When `CheckWithSimilarity` runs - Then `Report.Clusters` has exactly 1 cluster with 3 `Members`, 3 `Pairs` (each keeping its own Type), and `DupLines` equal to the sum of all pairs' `DupLines`|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ReportsClusterForThreeMutuallyDuplicateFunctions`|
|Given the same 3-function fixture but with C moved to a different package directory - When checked - Then `Clusters[0].CrossPackage` is `true`|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ClusterCrossPackageFlagTrueWhenMembersSpanPackages`|
|Given the existing 2-function `CalculateTax`/`CalculateFee` fixture (unchanged from slice 1) - When checked - Then `Report.Clusters` has exactly 1 cluster with 2 `Members` and 1 `Pair`, matching today's single pairwise `Violation` exactly|Boundary|unit test: `internal/duplication/duplication_test.go: TestCheck_ClusterOfTwoDegeneratesToSinglePair`|
|Given two independent duplicate groups with different summed `DupLines` - When checked - Then `Report.Clusters` is ordered by `DupLines` descending|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ClustersSortedByDupLinesDescending`|
|Given `buildClusters`'s postcondition assert (every emitted cluster has >= 2 members) - When a normal multi-function repo is checked - Then no panic occurs (assert is defensive, unreachable via the public API by construction, same style as the existing `TestCheck_PairComparisonAssertion`)|Exception|unit test: `internal/duplication/duplication_test.go: TestCheck_ClusterMinimumMembersAssertion`|
|Given the 3-function fixture run through the built CLI with `--format=json` - When parsed - Then the `clusters` array has 1 entry whose `pairs` show both `Type-1` and `Type-2` labels|Normal|integration test (built CLI path): `cmd/boy-scout/main_test.go: TestRun_DuplicationJSONIncludesClustersField`|
|Given the existing pairwise CLI/JSON tests from slices 1-2 - When re-run unmodified after this change - Then `TestRun_DuplicationReportsRenamedFunctionAsType2Clone` and `TestRun_DuplicationJSONFormatOutputsValidSchema` still pass, no test edits needed|Exception (regression lock)|integration test: `cmd/boy-scout/main_test.go` (both existing tests, unmodified)|
|Given the full repo after this change - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (26+ duplication tests + new ones, no regressions)|Exception (regression lock)|build: `go build ./... && go test ./...`|

## Assertions

Each line below is a real `assert`-style guard (`assertf(cond, msg, ...)`) written directly in the implementation, not a comment or test-only check.

- `internal/duplication/duplication.go`, `buildClusters()`, postcondition: `assertf(len(c.Members) >= 2, "cluster %v has fewer than 2 members", c)` on every cluster it emits. Uncertainty: Low — every `Violation` connects exactly 2 distinct functions by construction, so union-find can never produce a singleton group; this is defensive insurance against a future refactor, same as the existing `i != j` assert in `reportDuplicates`.
- `internal/duplication/duplication.go`, `buildClusters()`, cross-package detection via `filepath.Dir(member.File)` comparison: Low impact, Low uncertainty — Go's package-per-directory convention makes this a reliable signal; no assert needed, just documented as an assumption (would need revisiting only if boy-scout ever supports a language where "package" isn't 1:1 with "directory").

## Known simplifications (ponytail marks)

- `ponytail:` Bubble-sort for cluster and member sorting on small slices; fine at duplication's typical cluster sizes. Upgrade path: use `sort.Slice` if profiling shows sorting is a bottleneck.
- `ponytail:` Union-find without path halving (only path compression); fine for small-to-medium repos. Upgrade path: add path halving if dogfood testing reveals slowness on repos with 10k+ functions.

## Release / Branch merge

Branch: `feat/duplication-detector` (continuing from slices 1-2). **MUST CONFIRM** with jaeyong.cho.97@gmail.com before merging into `main` or cutting a release — same gate as slices 1-2, now covering slices 1-2-A-B together as one merge.
