---
type: Spec Story
title: Add filelen violation guidance to the gardener auto-fix skill
description: Prose-only addition to internal/setup/skill.md explaining a filelen violation as low cohesion/high coupling and instructing the AI to split the file, fixed last after gofunclen and crap
tags: [spec, claude-code-integration]
timestamp: 2026-08-23T00:00:00+09:00
---

# Add filelen violation guidance to the gardener auto-fix skill

## Value to user

The gardener auto-fix skill now explains which clean-code rule a filelen violation breaks in plain terms (not just "filelen violation"), instructs to split files along natural seams with high cohesion and loose coupling, and specifies the fix order: filelen violations are fixed last, after gofunclen and crap, since splitting a file shifts every other violation's line number in it.

## Completion criteria

The embedded `SKILL.md` template explains filelen as "mixing multiple concerns" and instructs splitting into files with "high cohesion" and "loose coupling"; states violations are fixed in the order gofunclen → crap → filelen; and specifies that file-level reorganization goes last to avoid invalidating line numbers mid-run. No CLI flag, scoring formula, or `gardener` Go logic changes — only skill prose changes.

## Spec

The embedded skill template (`internal/setup/skill.md`, written by `gardener setup`):

- **Filelen violation explanation:** Explains in plain language which clean-code rule a filelen violation breaks, self-contained (no external file read):
  - filelen violation: the file has grown too big to hold one responsibility — it's mixing multiple concerns. Fix by splitting into separate files along natural seams, each with high cohesion (one clear job) and loose coupling (minimal knowledge of the others' internals).
- **Fix order with rationale:** States that violations are fixed in order: gofunclen first, then crap, then filelen — file-level reorganization goes last, since splitting a file shifts every other violation's line number in it; fixing the small stuff first, then re-running `gardener go all` before reorganizing, avoids invalidating violation locations mid-run. This ordering rationale is worth a one-line note directly in `skill.md` itself, not just this spec, since it's non-obvious to a reader of the skill.
- **Machine-local path guard:** The embedded skill content is checked at runtime to ensure it contains no machine-local path (`/Users/`, `~/workspace`), keeping the violation explanations self-contained so the skill works when copied to another machine via `gardener setup --global`.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given the embedded skill template after this change - When `Run()` writes it - Then it contains a bullet explaining a filelen violation as the file "mixing multiple concerns"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFilelenGuidance/FilelenExplainsCleanCodeRule`|
|Given the embedded skill template - When read - Then it instructs splitting into files with "high cohesion" and "loose coupling"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFilelenGuidance/FilelenExplainsCohesionAndCoupling`|
|Given the embedded skill template - When read - Then it states filelen violations are fixed last, after gofunclen and crap|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFilelenGuidance/FilelenFixedLast`|
|Given the embedded skill template content - When `Run()` writes it - Then it still contains no machine-local path (`/Users/`, `~/workspace`)|Exception (regression lock)|unit test: `internal/setup/setup_test.go: TestRun_TemplateHasNoMachineLocalPath` (existing test, unchanged, re-run after this edit)|
|Given the pre-existing `internal/setup/setup_test.go` suite (`TestRun_TemplateDeclaresUserInvokedFixLoop`, `TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance`, etc.) - When re-run after this change - Then all pass unmodified|Exception (regression lock)|unit test: `internal/setup/setup_test.go` (existing suite, unchanged)|
|Given `internal/setup/skill.md` after this change - When a human reads it top to bottom - Then it reads as one coherent skill with no contradictory instructions (three violation types explained, one fix order stated once)|Normal|manual review: open the file, read it once, confirm no contradiction|
