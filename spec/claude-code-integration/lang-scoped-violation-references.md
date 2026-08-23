---
type: Spec Story
title: Generalize boy-scout's reference files by language
description: Split boy-scout's Go-only SKILL.md and reference files into a language-agnostic core plus per-language references/lang/{go,cpp}/ files, discovered by project language at skill-run time
tags: [spec, claude-code-integration, multi-language]
timestamp: 2026-08-23T00:00:00+09:00
---

# Generalize boy-scout's reference files by language

## Value to user

The boy-scout auto-fix skill now discovers your project's language (Go or C++) from version-control markers (`go.mod` vs `CMakeLists.txt`/`*.cpp`/`*.hpp`) instead of assuming Go. The SKILL.md becomes language-agnostic; each language gets a separate index file (`references/lang/{lang}/index.md`) documenting its test command, test-file suffix, available checks, and ignore-comment syntax. Violation-type reference files are split: the generic "why/how" sections stay at the top level (`references/{kind}.md`), while language-specific code examples move into `references/lang/{lang}/{kind}.md` (Go has all 5 kinds, C++ has funclen only). Projects containing both Go and C++ run both language's checks, merge the violations into one ranked list, and apply the existing 5-per-run cap unchanged. Projects with no matching language marker stop and refuse to run, preventing silent misidentification.

## Completion criteria

The embedded SKILL.md now contains a language-discovery step (look for `go.mod` or `CMakeLists.txt`/`*.cpp`/`*.hpp`), a per-language index-file read step, and a per-violation-kind table that points to both the generic reference file and the language-specific example file. The five top-level reference files (`references/{funclen,crap,filelen,instability,abstractness}.md`) contain only "Why this is a problem"/"How to fix it" sections; code examples are gone. Five new files live under `references/lang/go/`: an index file plus one per-kind file with the Go examples moved verbatim from the old top-level files. Two new files live under `references/lang/cpp/`: an index file plus `funclen.md` with a C++ translation of the Go example. The `//go:embed` directive in `internal/setup/setup.go` continues to work unchanged, since it already recursively includes `references/*` with zero code changes needed.

## Spec

The embedded skill template (`internal/setup/SKILL.md`) is rewritten; five top-level reference files are edited to remove language-specific examples; two new `lang` directory trees are added under `internal/setup/references/`:

- **Updated SKILL.md:** 
  - New "Discover Your Project's Language" section explaining the `go.mod`, `CMakeLists.txt`, `*.cpp`, `*.hpp` markers.
  - New "Per-Language Setup" section instructing the reader to read the language-specific index file before running checks.
  - New "Running Checks" section with language-specific commands and multi-language merge instruction.
  - Updated "Before Starting" and "Processing Violations" sections to be generic (no hardcoded `go test ./...`).
  - Updated violation-kind table with three columns: generic why/how (`references/<kind>.md`), language-specific Go example (`references/lang/go/<kind>.md`), language-specific C++ example (`references/lang/cpp/<kind>.md` for funclen only).

- **Five reference files edited (top-level, no code examples):**
  - `references/funclen.md`: Removed "Problem example" and "Good resolve example" sections; kept only "Why this is a problem" and "How to fix it".
  - `references/crap.md`: Removed code examples; kept only conceptual sections.
  - `references/filelen.md`: Removed code examples; kept only conceptual sections.
  - `references/instability.md`: Removed code examples; kept only conceptual sections.
  - `references/abstractness.md`: Removed code examples; kept only conceptual sections.

- **New language-specific files under `references/lang/go/`:**
  - `references/lang/go/index.md`: Test command (`go test ./...`), test-file suffix (`_test.go`), all 5 available check kinds, ignore-comment syntax (`// boy-scout:ignore`), CRAP-in-test-files note.
  - `references/lang/go/funclen.md`: Go `ProcessOrder` before/after example, moved verbatim from old `references/funclen.md`.
  - `references/lang/go/crap.md`: Go `calculateDiscount` before/after example, moved verbatim from old `references/crap.md`.
  - `references/lang/go/filelen.md`: Go multi-file order refactoring before/after example, moved verbatim from old `references/filelen.md`.
  - `references/lang/go/instability.md`: Go domain/httpapi inversion before/after example, moved verbatim from old `references/instability.md`.
  - `references/lang/go/abstractness.md`: Go cache interface extraction before/after example, moved verbatim from old `references/abstractness.md`.

- **New language-specific files under `references/lang/cpp/`:**
  - `references/lang/cpp/index.md`: Documents funclen as the only available check, states CRAP and `boy-scout:ignore` are not yet supported.
  - `references/lang/cpp/funclen.md`: C++ translation of the same `ProcessOrder` example (same shape as Go version, C++ syntax).

- **Assertions in `internal/setup/setup.go`:**
  - SKILL.md must not contain `"go test ./..."` (proves the generic skill has no baked-in single-language assumption).
  - Every embedded reference file must be non-empty.

- **Embedding unchanged:** The `//go:embed SKILL.md references/*` directive in `setup.go` continues to work as-is; the recursive `references/*` pattern already picks up `lang/` subdirectories with zero code changes needed.

- **Out of scope:** No change to `boy-scout go` or `boy-scout cpp` CLI dispatch logic; the binary already supports both languages. No change to any violation check implementation. The 5-per-run cap and priority ordering stay exactly as before; multi-language projects just merge the violations before applying the cap.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given `Run()` after this change - When it writes the skill template - Then it also creates `references/lang/go/index.md`, `references/lang/go/{funclen,crap,filelen,instability,abstractness}.md`, `references/lang/cpp/index.md`, `references/lang/cpp/funclen.md`|Normal|unit test: `internal/setup/setup_test.go: TestRun_WritesLangReferenceFiles`|
|Given `references/lang/go/funclen.md` - When read - Then it contains the Go `ProcessOrder` problem/resolve example moved verbatim from the old `references/funclen.md`|Normal|unit test: `internal/setup/setup_test.go: TestRun_LangReferenceFilesContainLanguageExamples/GoFunclen`|
|Given `references/lang/cpp/funclen.md` - When read - Then it contains a real C++ translation of the same `ProcessOrder` example (contains `ProcessOrder(` and `#include`)|Normal|unit test: `internal/setup/setup_test.go: TestRun_LangReferenceFilesContainLanguageExamples/CppFunclen`|
|Given the top-level `references/{kind}.md` files after this change - When read - Then none contains a "Problem example" or "Good resolve example" heading or Go code fence (why/how only, language-agnostic)|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TopLevelReferenceFilesNoLongerContainLanguageExamples`|
|Given `references/lang/go/index.md` - When read - Then it states `go test ./...`, `_test.go`, and lists all 5 available kinds|Normal|unit test: `internal/setup/setup_test.go: TestRun_LangIndexFilesDescribeLanguageSpecifics/Go`|
|Given `references/lang/cpp/index.md` - When read - Then it states only `funclen` is available and that CRAP / `boy-scout:ignore` are not supported for cpp|Normal|unit test: `internal/setup/setup_test.go: TestRun_LangIndexFilesDescribeLanguageSpecifics/Cpp`|
|Given `SKILL.md` after this change - When read - Then it instructs discovering the project's language from `go.mod` vs `CMakeLists.txt`/`*.cpp`/`*.hpp` before running any `boy-scout <lang>` command|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDiscoversProjectLanguage`|
|Given `SKILL.md` after this change - When read - Then it no longer contains the literal substring `go test ./...` or any cpp-only aside sentence outside the per-language index files|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateHasNoHardcodedLanguageAssumptions`|
|Given `SKILL.md`'s per-violation-kind table - When read - Then each row points at both `references/<kind>.md` (why/how) and `references/lang/{lang}/<kind>.md` (example)|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateRoutesToLangSpecificExample`|
|Given `SKILL.md` - When read - Then it states that a project matching more than one language marker runs every matching language's checks and merges all violations into one ranked list before the existing 5-per-run cap|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TemplateHandlesMultiLanguageProject`|
|Given `SKILL.md` - When read - Then it states that no matching `references/lang/{lang}/` directory means stop, run nothing|Exception|unit test: `internal/setup/setup_test.go: TestRun_TemplateHandlesUnsupportedLanguage`|
|Given `boy-scout setup` run end-to-end through the real CLI dispatch (`run(["setup","agents"], ...)`) - When it completes - Then `./.agents/skills/boy-scout/references/lang/go/funclen.md` and `./.agents/skills/boy-scout/references/lang/cpp/funclen.md` both exist and contain their respective language's example|Normal|integration test (built CLI path): `cmd/boy-scout/main_setup_test.go: TestRun_SetupWritesLangReferenceFiles`|
|Given every embedded reference file's content, including the new `lang/` ones - When `Run()` writes it - Then none contains a machine-local path (already-generic `assertNoMachineLocalPath`, no new code needed) and none is empty (new `assertf` in `writeReferenceFile`)|Exception (regression lock)|unit test: `internal/setup/setup_test.go: TestRun_LangReferenceFilesHaveNoMachineLocalPathOrEmptyContent`|
|Given the full repo after this change - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass (10 baseline + new tests, no regressions)|Exception (regression lock)|build: `go build ./... && go test ./...`|
