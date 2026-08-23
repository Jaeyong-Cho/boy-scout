---
type: Spec Story
title: crap-default-threshold-fix
description: Fix boy-scout go crap's default threshold from 6.0 to 30.0 (sized for CRAP score, not cyclomatic complexity)
tags: [spec, clean-code-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# crap-default-threshold-fix

## Value to user

The CRAP score formula (`comp² × (1−cov)³ + comp`) grows much faster than cyclomatic complexity alone. The current default threshold of 6.0 is sized for cyclomatic complexity metrics, making the `--threshold` flag too strict by default. Changing it to 30.0 aligns with the CRAP4J tool's established convention and reduces false positives from overly aggressive defaults.

## Completion criteria

`boy-scout go crap` with no `--threshold` flag defaults to 30.0 (not 6.0), matching the CRAP4J convention. The flag remains fully overridable and backward-compatible — users can still pass `--threshold=6.0` to get the old behavior if desired.

## Spec

The CRAP score default threshold lives in one location: `cmd/boy-scout/main_runners.go:49`. Changing the hardcoded default from `6.0` to `30.0` in the flag definition propagates to every caller of `crap.Check`. This is a single-line, single-constant fix with no new behavior or branch logic.

### Spec changes

- `cmd/boy-scout/main_runners.go:49`: Default value in `fs.Float64("threshold", ...)` changes from `6.0` to `30.0`.
- CLI behavior: `boy-scout go crap <path>` (no `--threshold` flag) now uses 30.0 as the default.
- Override behavior: `--threshold=6.0` (or any other value) still works exactly as before.
- Exit codes and output format: unchanged.
- `crap.Check` function signature and internal logic: unchanged (it receives the threshold as a parameter).

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given a function with cyclomatic complexity 8 and 70% test coverage (CRAP ≈ 9.7) - When `crap.Check` runs with no explicit threshold override (default 30.0) - Then it reports 0 violations (9.7 < 30)|Normal|unit test: `internal/crap/crap_test.go: TestCrapDefaultThreshold_BorderlineCaseNotFlaggedAtNewDefault`|
|Given the same function - When `crap.Check` runs with an explicit `threshold=6.0` - Then it reports 1 violation (9.7 > 6)|Boundary|unit test: `internal/crap/crap_test.go: TestCrapDefaultThreshold_OldDefaultStillOverridable`|
|Given a built `boy-scout` binary and a Go fixture file containing a function with complexity 8 and 70% coverage (CRAP ≈ 9.7) - When `boy-scout go crap testdata/borderline.go` runs with no flags - Then stdout says "0 violations" (or empty violation list in JSON mode) and exit code is 0|Normal|integration test: `cmd/boy-scout/main_test.go: TestRun_CrapDefaultThresholdIs30`|
|Given the same fixture - When `boy-scout go crap --threshold=6.0 testdata/borderline.go` runs - Then exit code is 1 (violation) and output shows the function as a violation|Normal|integration test: implicit in TestRun_CrapRespectsThresholdFlag (already exists, confirms override still works)|
|Given a `_test.go` file with the same complexity/coverage (part of existing fixture from other AC) - When `boy-scout go crap <path>` runs (new default 30.0) - Then test file is excluded by default (per crap-ignores-test-files story) and no false extra violations appear|Regression (lock)|implicit: same fixture, same exclude logic; test passes if no new violations surface|
|Given `boy-scout go all <path>` run against the same fixture - When run - Then the `[crap]` section of combined output shows 0 violations (new default applies to `all` subcommand too)|Normal|integration test: implicit in TestRun_AllCombinesGofunclenAndCrap and related `all` tests (already exist, verify regression)|
