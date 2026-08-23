---
type: Spec Story
title: Per-violation resolve-method reference files for the boy-scout skill
description: Replace the boy-scout auto-fix skill's inline one-line violation explanations with a small SKILL.md index plus one self-authored "why is it a problem / how to fix it" reference file per violation kind, closing the gap where instability/abstractness had no guidance at all
tags: [spec, claude-code-integration]
timestamp: 2026-08-23T00:00:00+09:00
---

# Per-violation resolve-method reference files for the boy-scout skill

## Value to user

The boy-scout auto-fix skill now ships dedicated reference files for each of the 5 violation kinds (funclen, crap, filelen, instability, abstractness). Instead of one-line inline explanations in SKILL.md, the skill routes you to the matching reference file and tells you to read both the "Why this is a problem" and "How to fix it" sections before editing. This closes the gap where instability and abstractness had zero guidance, provides deeper and more actionable explanations than inline one-liners, and follows deep-modules principles applied to the skill itself: SKILL.md is a small index, the actual depth lives in separate files.

## Completion criteria

The embedded SKILL.md now contains a table mapping each of the 5 violation kinds to its `references/<kind>.md` file, with instructions to read both sections before editing. The `references/` directory is shipped alongside SKILL.md with one `.md` file per violation kind, each self-authored (not copied from other projects) with two sections: "Why this is a problem" (concrete harm specific to that violation) and "How to fix it" (concrete resolution method). The three old inline bullets (gofunclen, crap, filelen one-liners) are removed from SKILL.md; the two new kinds (instability, abstractness) that previously had zero inline guidance now have full reference files. No CLI flag or code-logic change.

## Spec

The embedded skill template (`internal/setup/skill.md`) and 5 new reference files under `internal/setup/references/`:

- **Updated SKILL.md:** The three inline violation-explanation bullets are removed. In their place: a fix-order sentence covering all 5 kinds (funclen → crap → filelen → instability → abstractness, with "more disruptive fixes go last" rationale), and a table mapping each violation kind (as printed by `boy-scout go all`) to its reference file, with an instruction to read both the "Why this is a problem" and "How to fix it" sections before editing. Note: Go's CLI uses token `gofunclen` and C++'s uses `funclen`; both are explained in the same `references/funclen.md` file.

- **Five reference files, one per violation kind:**
  - `references/funclen.md`: Why — function doing more than one thing at one level of abstraction. How — extract each logical step into a well-named helper, so the original function reads like a table of contents.
  - `references/crap.md`: Why — high complexity plus low test coverage, so nobody can prove a change is safe. How — add a characterization test if 0% coverage, then refactor to reduce complexity.
  - `references/filelen.md`: Why — file mixing multiple concerns, hard to understand/test/reuse. How — split along natural seams with high cohesion (one clear job) and loose coupling (minimal knowledge of others' internals).
  - `references/instability.md`: Why — stable package importing something less stable, dragging stability into the unstable thing's change risk. How — invert the dependency, or extract a new small stable package both can depend on.
  - `references/abstractness.md`: Why — concrete package depended on by many (Zone of Pain) or abstract package with no dependents (Zone of Uselessness). How — extract the stable boundary into a small abstract deep module, move implementations elsewhere.

- **Embedding and write:** The `//go:embed` directive in `internal/setup/setup.go` is updated to include `references/*.md`. A new function `writeReferenceFiles()` creates `<skill-dir>/references/` and writes all 5 files, calling `assertNoMachineLocalPath()` on each one before write (same guards as SKILL.md).

- **Machine-local path guard:** Every embedded reference file is checked at runtime to ensure no `/Users/` or `~/workspace` path appears, keeping the explanations self-contained so the skill works when copied via `boy-scout setup --global`.

- **Out of scope:** No change to `internal/instability`, `internal/abstractness`, `internal/gofunclen`, `internal/crap`, `internal/filelen` check logic; no CLI rename of `gofunclen`; the CRAP-in-test-files note and TDD green-baseline gate paragraphs in SKILL.md are untouched.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given `Run()` after this change - When it writes the skill template - Then it also creates `references/funclen.md`, `references/crap.md`, `references/filelen.md`, `references/instability.md`, `references/abstractness.md` next to `SKILL.md`|Normal|unit test: `internal/setup/setup_test.go: TestRun_WritesAllReferenceFiles`|
|Given `references/funclen.md` - When read - Then it contains "one level of abstraction" (why) and "table of contents" (how)|Normal|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesExplainWhyAndHow/Funclen`|
|Given `references/crap.md` - When read - Then it contains "high complexity plus low test coverage" (why) and "characterization test" (how)|Normal|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesExplainWhyAndHow/Crap`|
|Given `references/filelen.md` - When read - Then it contains "mixing multiple concerns" (why) and both "high cohesion" and "loose coupling" (how)|Normal|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesExplainWhyAndHow/Filelen`|
|Given `references/instability.md` - When read - Then it contains "least-stable thing it leans on" (why) and "Point the dependency the other way" (how)|Normal|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesExplainWhyAndHow/Instability`|
|Given `references/abstractness.md` - When read - Then it contains "Zone of Pain" and "Zone of Uselessness" (why) and "deep module" (how)|Normal|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesExplainWhyAndHow/Abstractness`|
|Given `SKILL.md` after this change - When read - Then it contains a table row for each of the 5 violation kinds pointing at its `references/<kind>.md` file, and `gofunclen`'s row is the same file (`references/funclen.md`) as `cpp`'s `funclen`|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateMapsViolationsToReferenceFiles`|
|Given `SKILL.md` - When read - Then it states the fix order "then filelen, then instability, then abstractness"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateMapsViolationsToReferenceFiles` (same test, second assertion)|
|Given `SKILL.md` after this change - When read - Then it no longer contains the old inline markers ("**gofunclen violation:**", "**crap violation:**", "**filelen violation:**" as bullet text) — proving the explanation actually moved, not just got duplicated|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateNoLongerInlinesViolationExplanations`|
|Given every embedded reference file's content - When `Run()` writes it - Then none contains a machine-local path (`/Users/`, `~/workspace`)|Exception (regression lock)|unit test: `internal/setup/setup_test.go: TestRun_ReferenceFilesHaveNoMachineLocalPath`|
|Given the pre-existing `TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance` suite with its `FunclenExplainsCleanCodeRule`/`CrapExplainsCleanCodeRule` cases removed - When the remaining cases (`BaselineGreenGate`, `CharacterizationTestForZeroCoverage`, `AttemptCapIncludesCharacterizationTest`) re-run - Then they still pass unmodified|Exception (regression lock)|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance`|
|Given the pre-existing `TestRun_TemplateDeclaresFilelenGuidance` suite with its rule-explanation cases removed - When its remaining fix-order case is updated to the new 5-kind order text - Then it passes|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFilelenGuidance`|
|Given `boy-scout setup` run end-to-end through the real CLI dispatch (`run(["setup","agents"], ...)`) - When it completes - Then `./.agents/skills/boy-scout/references/` contains all 5 files, and `funclen.md` contains "one level of abstraction" while `abstractness.md` contains "Zone of Pain"|Normal|integration test (built CLI path): `cmd/boy-scout/main_setup_test.go: TestRun_SetupWritesReferenceFiles`|
|Given the full repo after this change - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (195 baseline + 7 new reference-file tests, no regressions)|Exception (regression lock)|build: `go build ./... && go test ./...`|
|Given `internal/setup/skill.md` and the 5 new reference files - When a human reads them top to bottom - Then they read as one coherent skill (a small index plus 5 focused explanations), with no paragraph duplicated verbatim between two files|Normal|manual review: open all 6 files, read once, confirm no contradiction or verbatim duplication|
