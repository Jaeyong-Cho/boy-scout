---
type: Spec Story
title: TDD refactor gate and clean-code violation explanations
description: Add a green-baseline check, characterization-test step for untested crap violations, and plain clean-code explanations per violation type to the gardener-go auto-fix skill
tags: [spec, claude-code-integration]
timestamp: 2026-08-22T00:00:00+09:00
---

# TDD refactor gate and clean-code violation explanations

## Value to user

The gardener-go auto-fix skill now refuses to edit code with a failing test suite (never refactor on RED), explains which clean-code rule each violation breaks in plain terms (not just "funclen violation" or "crap violation 43%"), and adds one minimal characterization test before refactoring untested crap violations — ensuring the AI can tell its edits from pre-existing breakage and understands why the fix matters.

## Completion criteria

The embedded `SKILL.md` template instructs a `go test ./...` check before any edit and stops on red; explains funclen as "one level of abstraction" and crap as "high complexity plus low test coverage"; adds a characterization test for 0%-coverage crap violations before refactoring; and states the 3-attempt cap includes the characterization-test step. No CLI flag, scoring formula, or `gardener-go` Go logic changes — only skill prose changes.

## Spec

The embedded skill template (`internal/setup/skill.md`, written by `gardener-go setup`):

- **Green-baseline gate:** Instructs running `go test ./...` once before starting the fix loop. If it fails, stop immediately and report that tests were already failing before any gardener-go edit — never refactor on top of a red suite.
- **Violation explanations:** Explains in plain language which clean-code rule each violation breaks, self-contained (no external file read):
  - funclen violation: the function is too big to hold one level of abstraction — it's doing more than one thing. Fix by extracting sub-steps into well-named helpers.
  - crap violation: high complexity plus low test coverage — nobody can prove a change to it is safe. Fix by simplifying the logic, backed by a real test.
- **Characterization test for 0%-coverage crap:** For a crap violation whose reported coverage is 0%, add one minimal characterization test (pins current behavior, not desired) and confirm it passes before refactoring. Skip this step when coverage is already above 0%.
- **Attempt cap includes characterization test:** The existing 3-attempts-then-unresolved cap covers the whole fix for a violation, including any characterization test written for it — not an unlimited separate phase.
- **Machine-local path guard:** The embedded skill content is checked at runtime to ensure it contains no machine-local path (`/Users/`, `~/workspace`), keeping the violation explanations self-contained so the skill works when copied to another machine via `gardener-go setup --global`.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given the embedded skill template after this change - When `Run()` writes it - Then it instructs a `go test ./...` check before any edit and to stop on red|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/BaselineGreenGate`|
|Given the embedded skill template - When read - Then it explains a funclen violation as breaking "one level of abstraction"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/FunclenExplainsCleanCodeRule`|
|Given the embedded skill template - When read - Then it explains a crap violation as "high complexity plus low test coverage"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/CrapExplainsCleanCodeRule`|
|Given the embedded skill template - When read - Then it instructs adding a characterization test for 0%-coverage crap violations before refactoring|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/CharacterizationTestForZeroCoverage`|
|Given the embedded skill template - When read - Then the 3-attempt cap is stated to include the characterization-test step, not exempt from it|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/AttemptCapIncludesCharacterizationTest`|
|Given the embedded skill template content - When `Run()` writes it - Then it contains no machine-local path (`/Users/`, `~/workspace`) that would break the skill on another machine|Exception|unit test: `internal/setup/setup_test.go: TestRun_TemplateHasNoMachineLocalPath`|
|Given the pre-existing setup-package test suite (`TestRun_CreatesSkillFileAtBaseDir`, `TestRun_OverwritesExistingSkillFile`, `TestRun_TemplateDeclaresUserInvokedFixLoop`, `TestRun_ReturnsErrorWhenDirUnwritable`) - When re-run after this change - Then all pass unmodified|Exception (regression lock)|unit test: `internal/setup/setup_test.go` (existing suite, unchanged)|
|Given the pre-existing `gardener-go setup` CLI test suite in `cmd/gardener-go/main_test.go` - When re-run after this change - Then all pass unmodified (no CLI behavior changed, only skill prose)|Exception (regression lock)|unit test: `cmd/gardener-go/main_test.go` (existing `TestRun_Setup*` suite, unchanged)|
|Given `internal/setup/skill.md` after this change - When a human reads it top to bottom - Then it reads as one coherent skill with no contradictory instructions|Normal|manual review: open the file, read it once, confirm no contradiction|
