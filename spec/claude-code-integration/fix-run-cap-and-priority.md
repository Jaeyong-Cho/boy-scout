---
type: Spec Story
title: Cap the boy-scout auto-fix skill at 5 violations per run, worst-first, test files last
description: Add a hard 5-violation cap per run to the boy-scout auto-fix skill, ordering by severity (worst-first) within each existing violation kind and deferring test-file violations to the end of the list
tags: [spec, claude-code-integration]
timestamp: 2026-08-23T00:00:00+09:00
---

# Cap the boy-scout auto-fix skill at 5 violations per run, worst-first, test files last

## Value to user

The boy-scout auto-fix skill now processes at most 5 violations per run, ordered by severity (worst-offending first within each violation kind) and deferring test-file violations to the end. This bounds how much work and how large a diff one run produces, preventing runaway sessions and keeping the feedback loop tight for rapid iteration. The skill reports how many violations were skipped by the cap (never attempted) separately from unresolved (attempted, failed 3 times), so you know at a glance whether to re-run or investigate a stuck violation.

## Completion criteria

The embedded SKILL.md now contains a paragraph (inserted after the existing fix-order sentence and before the reference-file table) explaining the 5-violation cap, the worst-first severity ordering within each kind, test-file deferral, and the distinction between cap-skipped and unresolved violations in the final report. No CLI flag, no code-logic change to the checks themselves — prose only.

## Spec

**Cap each run at 5 violations.** Within each kind (funclen → crap → filelen → instability → abstractness, unchanged), process the worst violations first: rank by the severity number `boy-scout go all` already prints — for funclen/gofunclen and filelen, lines minus the limit; for crap, the CRAP score; for instability, the Gap; for abstractness, the Distance — highest number first. Regardless of kind or severity, violations in test files (`*_test.go`) are deferred to the end of the list, after every non-test violation. Stop once 5 violations total have been processed (fixed or marked unresolved) this run, including any characterization test step — same accounting as the existing 3-attempts-per-violation cap. At the end, report fixed vs. unresolved as before, plus how many remaining violations were skipped by the cap (never attempted) — distinct from unresolved (attempted, failed 3 times).

The new paragraph is inserted directly after the existing fix-order paragraph (the one ending "...so instability and abstractness go last.") and before the "Before editing each violation, read the corresponding reference file below." paragraph. No other paragraph in `skill.md` changes.

**Out of scope:** No CLI flag or Go check-logic change; no git usage; no configurable cap; no change to the 3-attempts-per-violation retry cap or the existing kind order.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given `internal/setup/skill.md` after this change - When `Run()` writes it - Then it states the cap: "Cap each run at 5 violations"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFixCapAndPriority/CapAtFive`|
|Given the skill template - When read - Then it states worst-first ordering within a kind: "highest number first"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFixCapAndPriority/SeverityWorstFirst`|
|Given the skill template - When read - Then it states test-file violations are deferred to the end: "deferred to the end of the list"|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFixCapAndPriority/TestFilesDeferredLast`|
|Given the skill template - When read - Then it distinguishes cap-skipped from unresolved: "skipped by the cap"|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresFixCapAndPriority/CapSkippedDistinctFromUnresolved`|
|Given the skill template after this change - When read - Then the pre-existing fix-order sentence ("then filelen, then instability, then abstractness") and the reference-file table are still present and unmodified|Exception (regression lock)|unit test: `internal/setup/setup_test.go: TestRun_TemplateMapsViolationsToReferenceFiles` (existing, unchanged)|
|Given `boy-scout setup` run end-to-end through the real CLI dispatch (`run(["setup","agents"], ...)`) - When it completes - Then the written `SKILL.md` contains all 4 new markers above|Normal|integration test (built CLI path): `cmd/boy-scout/main_setup_test.go: TestRun_SetupWritesCapAndPriorityGuidance`|
|Given the full repo after this change - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (202 baseline + new tests, no regressions)|Exception (regression lock)|build: `go build ./... && go test ./...`|
|Given `internal/setup/skill.md` after this change - When a human reads it top to bottom - Then it reads as one coherent skill, the new paragraph doesn't contradict or duplicate the existing fix-order/report sentences|Normal|manual review: open the file, read once, confirm no contradiction|
