---
type: Spec Story
title: Wire duplication into boy-scout's per-violation-kind reference docs
description: Add references/duplication.md + references/lang/go/duplication.md, extend SKILL.md's table/processing-order/cap/special-rules, and document the kind-scoped ignore comment, so `duplication` is documented like funclen/crap/filelen/instability/abstractness
tags: [spec, claude-code-integration, duplication]
timestamp: 2026-08-24T20:00:00+09:00
---

# Wire duplication into boy-scout's per-violation-kind reference docs

## Value to user

Boy-scout's duplication detector was released (v0.3.0 and earlier) with no documentation in the skill's reference files or SKILL.md processing order. An agent running boy-scout on a Go repo, hitting a duplication cluster, finds zero guidance on whether to fix it or why it matters. This Story closes that documentation gap, documenting duplication's fix strategy and tiering (same-package early, cross-package late) so agents can act on what the checker finds.

## Completion criteria

The duplication violation kind is now fully documented in boy-scout's reference hierarchy: `references/duplication.md` explains the problem and general fix strategy, `references/lang/go/duplication.md` shows a concrete before/after Go example, the SKILL.md table routes users to both, and the processing order tier is spelled out (same-package early with funclen, cross-package late after abstractness). The kind-scoped ignore directive `// boy-scout:ignore:duplication` is documented in the Go language guide. C++ remains unsupported (no `references/lang/cpp/duplication.md`), matching the existing CRAP precedent.

## Spec

- **`internal/setup/references/duplication.md` is created** with a language-agnostic "Why this is a problem" section (copy-pasted logic multiplies future fix costs), "Related concepts" cross-linking `meta-pattern.md` and `functions.md`, "How to fix it" section describing same-package vs. cross-package extraction strategies, and an Examples section pointing at `references/lang/go/duplication.md` and noting C++ is not yet supported.

- **`internal/setup/references/lang/go/duplication.md` is created** with a concrete Go before/after example: two functions `CalculateTax()` and `CalculateFee()` performing identical percentage calculations are refactored into one shared `calculatePercentage()` helper both callers use.

- **`internal/setup/references/lang/go/index.md` is updated:** "Available Checks" section lists `duplication` as item 4 (between filelen and instability); "Ignore Comments" section documents the kind-scoped `// boy-scout:ignore:duplication` directive to exclude just that function from duplication comparison.

- **`internal/setup/SKILL.md`'s violation-kind table is updated:** a `duplication` row is added between `filelen` and `instability`, with columns pointing to `references/duplication.md`, `references/lang/go/duplication.md`, and `(not yet supported for C++)` for the C++ column.

- **`internal/setup/SKILL.md`'s "Processing Violations" paragraph is updated:** same-package duplication clusters are inserted after `funclen` (cheap, local fix); cross-package duplication clusters (`CrossPackage: true`) are inserted after `abstractness` (most disruptive). Rationale extended to name duplication clusters as the concrete example of "package-boundary reshape" alongside instability/abstractness.

- **`internal/setup/SKILL.md`'s "Cap each run at 5 violations" paragraph is updated:** a duplication cluster counts as 1 against the 5-violation cap regardless of how many pairwise lines it resolves; clusters are ranked by `DupLines` (already summed by the checker), same as every other kind's "highest severity number first" rule.

- **`internal/setup/SKILL.md`'s "Special Rules by Violation Kind" section is updated:** a new `duplication` bullet instructs reading `--format=json` (not text) to inspect `members`, `pairs`, `dupLines`, `crossPackage` fields; fix the whole cluster in one atomic multi-file edit (extract shared helper, repoint every member, delete every duplicate body), then run the test suite once; default fix is always extract-a-helper, never delete-one-copy.

- **Assertions:** None. This is pure documentation (SKILL.md, references/**); consumed by an LLM at skill-run time, not executed code. The unit and integration tests (listed in AC below) verify the output matches the spec.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given `SKILL.md`'s Processing Violations order - When read - Then same-package duplication clusters are named in the early tier alongside `funclen`, and cross-package clusters (`CrossPackage: true`) are named in the late tier after `abstractness`|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateOrdersDuplicationBySamePackageVsCrossPackage`|
|Given `SKILL.md`'s cap paragraph - When read - Then it states a duplication cluster counts as 1 against the 5-violation cap regardless of how many pairwise lines it resolves|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateStatesClusterCountsAsOneAgainstCap`|
|Given `SKILL.md`'s Special Rules section - When read - Then it instructs reading `--format=json` for `members`/`pairs`/`dupLines`/`crossPackage`, fixing the whole cluster as one atomic edit, and using extract-a-helper as the default fix, never delete-one-copy|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateStatesDuplicationClusterFixRule`|
|Given `references/duplication.md` - When read - Then it contains "Why this is a problem" and "How to fix it" sections, cross-links `meta-pattern.md` and `functions.md`, and contains no Go/C++ code example|Normal|unit test: `internal/setup/setup_test.go: TestRun_DuplicationReferenceExplainsWhyAndHow`|
|Given `references/lang/go/duplication.md` - When read - Then it contains a concrete Go before/after example (two near-duplicate functions refactored into one shared helper both callers use)|Normal|unit test: `internal/setup/setup_test.go: TestRun_WritesGoDuplicationReferenceExample`|
|Given `references/lang/go/index.md` - When read - Then it lists `duplication` as an available check and documents the `// boy-scout:ignore:duplication` kind-scoped directive|Normal|unit test: `internal/setup/setup_test.go: TestRun_GoIndexListsDuplicationAndIgnoreDirective`|
|Given `SKILL.md`'s per-violation-kind table - When read - Then it has a `duplication` row pointing at `references/duplication.md` and `references/lang/go/duplication.md`, with the C++ column stating duplication isn't supported for C++ yet|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateTableRoutesDuplicationToReferences`|
|Given `references/lang/cpp/` - When listed after this change - Then it does NOT contain a `duplication.md` file, matching the existing CRAP-unsupported-for-C++ precedent|Boundary|unit test: `internal/setup/setup_test.go: TestRun_CppReferencesHaveNoDuplicationFile`|
|Given `boy-scout setup` run end-to-end through the real CLI dispatch (`run(["setup","agents"], ...)`) - When it completes - Then `./.agents/skills/boy-scout/references/duplication.md` and `./.agents/skills/boy-scout/references/lang/go/duplication.md` both exist and are non-empty|Normal|integration test (built CLI path): `cmd/boy-scout/main_setup_test.go: TestRun_SetupWritesDuplicationReferenceFiles`|
|Given the full repo after this change - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (293 total, no regressions)|Exception (regression lock)|build: `go build ./... && go test ./...`|
