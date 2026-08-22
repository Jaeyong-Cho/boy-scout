---
type: Spec Story
title: rename-funclen-gofunclen
description: Rename the Go-specific funclen checker's CLI/package/JSON surface to gofunclen, for symmetry with cppfunclen ahead of the language-agnostic filelen checker
tags: [spec, multi-language-variants, rename]
timestamp: 2026-08-22T00:00:00+09:00
---

# rename-funclen-gofunclen

## Value to user
`funclen` is the only Go-specific checker whose CLI/package name doesn't say "Go" — `cppfunclen` already does. This story ensures the Go checker's name is explicitly marked as Go-specific (`gofunclen`) before building the new language-agnostic `filelen` checker and its skill guidance. Pure rename — no behavior change.

## Completion criteria
`gardener go gofunclen` replaces `gardener go funclen` with identical flags, output shape, and semantics. All 99 tests pass (unchanged count from baseline). The `// gardener:ignore:funclen` suppression comment is renamed to `// gardener:ignore:gofunclen`.

## Spec

### CLI surface
- CLI subcommand `gardener go funclen` renamed to `gardener go gofunclen` (dispatch map key, `flag.NewFlagSet` name).
- All flags and output format (text/JSON) identical to the old `funclen` — no new flags, no breaking changes to the Report structure except the JSON top-level key name and struct field names.
- JSON field `funclen` → `gofunclen` (struct field `Funclen` → `Gofunclen`).
- Text output prefix `[funclen] ` → `[gofunclen] ` when combined with other checks in `gardener go all`.

### Internal surface
- Go package `internal/funclen` renamed to `internal/gofunclen` (directory, `package` clause, import path, all `funclen.`-qualified call sites in `cmd/gardener`).
- Suppression-comment keyword `// gardener:ignore:funclen` becomes `// gardener:ignore:gofunclen` (the literal string passed to the checker's own CLI name).
- Skill documentation `internal/setup/skill.md` updated: "funclen violation" → "gofunclen violation", and violation-priority order text updated.
- C++ `funclen` checker name unchanged (only the Go variant is renamed; the C++ variant stays as `gardener cpp funclen` with no `gardener:ignore` support yet).

### Implementation details
- All Go-specific logic moves with the package: violations, exclusion, test expectations.
- The `funcignore.Reason()` call in the gofunclen checker passes `"gofunclen"` as the checker name literal (line 59 in gofunclen.go).
- Test fixtures in `internal/funcignore` and `internal/crap` keep their example "funclen" strings unchanged (they're generic test data, not references to the renamed checker).

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given `gardener go gofunclen <path>` on a fixture with a 105-line function - When run - Then it reports the violation exactly as the old `gardener go funclen` did (same file/line/func/length/limit)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_GofunclenRespectsMaxLinesFlag` (renamed from `TestRun_FunclenRespectsMaxLinesFlag`)|
|Given `gardener go gofunclen` with no path argument - When run - Then it defaults to scanning `.`|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_GofunclenDefaultsToCurrentDir` (renamed)|
|Given `gardener go all --format=json` on a fixture with a gofunclen violation - When run - Then the JSON has a top-level `"gofunclen"` key (not `"funclen"`) containing that violation|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllCombinesGofunclenAndCrap` (renamed from `TestRun_AllCombinesFunclenAndCrap`)|
|Given `gardener go all` text output with a gofunclen violation present - When run - Then the printed line is prefixed `[gofunclen] ` (not `[funclen] `)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_AllCombinesGofunclenAndCrap` (same test, prefix assertion updated)|
|Given `--exclude-file`/`--exclude-func` flags passed to `gardener go gofunclen` - When run - Then exclusion still works exactly as before|Normal|unit test: `cmd/gardener/main_test.go: TestRun_ExcludeFlagsFilterGofunclenOutput` (renamed from `TestRun_ExcludeFlagsFilterFunclenOutput`)|
|Given a Go source function annotated `// gardener:ignore:gofunclen` - When `gardener go gofunclen` runs - Then that function is excluded from violations (old `// gardener:ignore:funclen` comment on existing user code stops working — documented, not silently broken)|Normal|unit test: `internal/gofunclen/gofunclen_test.go` (renamed from `internal/funclen/funclen_test.go`, comment strings updated)|
|Given the embedded skill template after this change - When read - Then it says "gofunclen violation" (not "funclen violation") in its clean-code explanation bullet|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance/FunclenExplainsCleanCodeRule` (kept — the marker text `"one level of abstraction"` doesn't change, so this existing test passes unmodified; no new test needed for this row)|
|Given the full repo after this rename - When `grep -rnE '\bfunclen\b\|\bFunclen\b' --include='*.go' cmd internal \| grep -viE 'cppfunclen\|CppFunclen'` runs - Then it returns zero matches|Exception (regression lock)|manual/build-time check: the grep command above, run as the last step before `go build`|
|Given the renamed repo - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (same 99-test count as baseline, just renamed)|Exception (regression lock)|build: `make check`|
