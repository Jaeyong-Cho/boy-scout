---
type: Spec Story
title: duplication-near-miss
description: Extend duplication checker with Type-3 near-miss clone detection using LCS-based similarity (NiCad-style threshold)
tags: [spec, clean-code-checks]
timestamp: 2026-08-24T00:00:00+09:00
---

# duplication-near-miss

## Value to user
Users can run `boy-scout go duplication --min-similarity=0.70` to identify functions that are copy-pasted and then edited (not just exact/renamed copies), catching more duplication than Type-1/Type-2 alone; this uses NiCad's LCS-based near-miss mechanism to find structurally similar code.

## Completion criteria
`boy-scout go duplication [--min-lines=N] [--min-similarity=F] [--format=text|json] [--exclude-file=...] [--exclude-func=...] [paths...]` successfully extends slice 1's Type-1/Type-2 detection with Type-3 (near-miss) classification using LCS-based similarity ratio; pairs exceeding the threshold are reported with their similarity percentage; backward compatibility is maintained: running without `--min-similarity` uses 0.70 as the default.

## Spec

### Command: `boy-scout go duplication` (Type-3 extension to slice 1)

- Backward compatible: existing Type-1 (exact) and Type-2 (renamed identifier) behavior unchanged
- Adds `--min-similarity=F` flag (default `0.70`, valid range `0.0` to `1.0` inclusive)
- For each pair of eligible functions:
  - If raw sequences match → Type-1, similarity 1.0 (as before)
  - Else if blind sequences match → Type-2, similarity 1.0 (as before)
  - Else compute LCS-based similarity: `similarity = 2*LCS(blindA, blindB) / (len(blindA) + len(blindB))` — standard normalized-LCS ratio, in range [0.0, 1.0]
    - If `similarity >= --min-similarity` → Type-3 violation reported with computed similarity
    - Else → no violation
- Invalid `--min-similarity` values (non-numeric, `< 0`, or `> 1`) result in exit code 2 with a clear error message; no scan is performed
- Text output shows Type-3 violations as: `fileA:lineA: function funcA is Type-3 duplicate of fileB:lineB function funcB (similarity% similar, N duplicated lines)`
- JSON output includes `"similarity"` field in each violation object (always set for all types, 1.0 for Type-1/Type-2)
- All slice 1 behavior (test-file exclusion, ignore comments, cross-file scanning, `boy-scout go all` integration) is unchanged

## Acceptance criteria

|AC|Category|Verification Method|
|--|--|--|
|Given two identical token sequences - When LCS similarity is computed - Then the ratio is exactly 1.0|Normal|unit test: `internal/duplication/duplication_test.go: TestSimilarity_IdenticalSequencesRatioIsOne`|
|Given two token sequences sharing no tokens - When LCS similarity is computed - Then the ratio is 0.0|Normal|unit test: `internal/duplication/duplication_test.go: TestSimilarity_DisjointSequencesRatioIsZero`|
|Given function B is function A plus one added statement (similarity ~0.85 by construction) - When `duplication.CheckWithSimilarity` runs with the default 0.70 threshold - Then one violation is reported with Type `Type-3` and the computed similarity|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_ReportsType3NearMissAboveThreshold`|
|Given two functions with similarity below 0.70 (by construction, e.g. only the same 2-line boilerplate wrapper shared) - When checked with the default threshold - Then no violation is reported for that pair|Normal|unit test: `internal/duplication/duplication_test.go: TestCheck_NoViolationBelowSimilarityThreshold`|
|Given a constructed pair with similarity at/above 0.70 - When checked with `--min-similarity=0.70` - Then the pair IS reported (inclusive boundary)|Boundary|unit test: `internal/duplication/duplication_test.go: TestCheck_SimilarityExactlyAtThresholdIsIncluded`|
|Given the same pair but similarity 0.69 - When checked with `--min-similarity=0.70` - Then the pair is NOT reported (exclusive below)|Boundary|unit test: `internal/duplication/duplication_test.go: TestCheck_SimilarityJustBelowThresholdIsExcluded`|
|Given `--min-similarity=150` or `--min-similarity=notanumber` on the CLI - When `boy-scout go duplication` is run - Then it exits with code 2 and a clear error, no scan performed|Exception|integration test: `cmd/boy-scout/main_test.go: TestRun_DuplicationInvalidMinSimilarityErrors`|
|Given a near-miss pair (similarity 0.80) and `--min-similarity=0.85` on the CLI - When run - Then the pair is not reported (flag overrides the 0.70 default)|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_DuplicationMinSimilarityFlagIsRespected`|
|Given all slice-1 fixtures and tests - When re-run after this slice's changes - Then every slice-1 AC still passes unchanged|Normal|regression: `go test ./internal/duplication/... ./cmd/boy-scout/...` (existing slice-1 tests, all passing)|

## Assertions

Each line below is a real `assert`-style guard written directly in the implementation:

- `internal/duplication/duplication.go`, `lcsSimilarity`: precondition `assertf(len(a) > 0 || len(b) > 0, "lcsSimilarity called with two empty sequences")` — an empty/empty pair is meaningless; Uncertainty: Low
- `internal/duplication/duplication.go`, `lcsSimilarity`: postcondition `assertf(ratio >= 0.0 && ratio <= 1.0, "similarity ratio %f out of [0,1] range", ratio)` — catches LCS formula off-by-one immediately; Uncertainty: Low
- `cmd/boy-scout/main_runners.go`, `runGoDuplication`'s `--min-similarity` validation: explicit check `if minSimilarity < 0.0 || minSimilarity > 1.0 return error` (not an assert, since this is user input) to catch invalid flag values before scanning; Uncertainty: Low
- **The 0.70 default threshold itself — Uncertainty: High.** Carried from NiCad's literature, not validated against boy-scout's own code. QA step 6 (dogfood against `internal/`) is the validation experiment.

## Known simplifications (ponytail marks)

- `ponytail:` LCS over token sequences of length N is O(N²) time/space per pair on top of the already-O(n²) pair count — fine at function-sized sequences (tens to low hundreds of tokens) and boy-scout's own repo size; upgrade path is capping compared sequence length or switching to n-gram Jaccard if future dogfood on a much larger repo is too slow.
- `ponytail:` 0.70 default is an assumption from NiCad literature, not tuned to boy-scout's codebase. Upgrade path: adjust default if dogfood finds it too loose (false positives) or too tight (missing real near-misses).

## QA Procedure

1. Precondition: slice 1's QA steps all still pass after this slice's changes (quick re-run, not re-documented here).
2. In a scratch module, write function A (7-8 lines) and function B = A with one extra `if` guard inserted (not just renamed variables) — run `boy-scout go duplication .` with defaults — expect one violation labeled `Type-3` with a similarity below 100% shown.
3. Trim function B further so it only shares a couple of lines with A — re-run — expect zero violations (below the 0.70 default).
4. Re-run step 3's case with `--min-similarity=0.3` — expect it now IS reported.
5. Run `boy-scout go duplication --min-similarity=150 .` — expect exit code 2 and a clear "invalid" error message, no report printed.
6. **Dogfood**: run `bin/boy-scout go duplication internal/` against boy-scout's own repo. Manually confirm `internal/gofunclen/gofunclen.go`'s `Check` function and `internal/filelen/filelen.go`'s `Check` function are flagged (Type-2 or Type-3 — whichever the actual similarity computes to).
