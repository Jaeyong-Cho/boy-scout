---
type: Spec Story
title: Fill in boy-scout's C++ reference docs for filelen, instability, abstractness
description: Document three C++ checks (filelen, instability, abstractness) that are already built and released but missing from reference files and tables
tags: [spec, claude-code-integration, cpp, docs]
timestamp: 2026-08-24T00:00:00+09:00
---

# Fill in boy-scout's C++ reference docs for filelen, instability, abstractness

## Value to user

Three boy-scout checks (`filelen`, `instability`, `abstractness`) were already shipped for C++ (v0.2.0 and earlier) but their documentation never caught up to production. An agent running boy-scout on a C++ repo, hitting one of these violations, reads the skill's reference files expecting a concrete C++ example and finds "not yet supported for C++" — wrong. This closes that documentation gap so the auto-fix skill stops misleading agents about C++ support.

## Completion criteria

The reference-file documentation now matches the actual CLI implementation: the SKILL.md table, the top-level references, and the C++ language index all document that `filelen`, `instability`, and `abstractness` are working C++ checks with concrete before/after code examples in C++ syntax.

## Spec

- **`internal/setup/references/lang/cpp/filelen.md` is created** with a C++ translation of the Go multi-file order-refactoring example: a single `order.cpp` mixing data model, business logic, and HTTP handler is split into focused `.hpp`/`.cpp` files.

- **`internal/setup/references/lang/cpp/instability.md` is created** with a C++ translation of the Go domain/httpapi dependency-inversion example: a `domain.hpp` including unstable `httpapi.hpp` is refactored so that `domain.hpp` defines a `NotificationSender` interface and only `httpapi.hpp` includes `domain.hpp`.

- **`internal/setup/references/lang/cpp/abstractness.md` is created** with a C++ translation of the Go cache interface example: a concrete `MemoryCache` class included by many is split into an abstract `Cache` base class (`cache_api.hpp`) and a `MemoryCache` implementation (`memcache.hpp`), so dependents include only the interface.

- **`internal/setup/SKILL.md`'s violation-kind table is updated:** the `filelen`, `instability`, `abstractness` rows' C++ columns now point to `references/lang/cpp/filelen.md`, `references/lang/cpp/instability.md`, `references/lang/cpp/abstractness.md` respectively (were `(not yet supported for C++)`).

- **`internal/setup/references/filelen.md`'s Examples section is updated:** C++ line now reads `See \`references/lang/cpp/filelen.md\`` (was `Not yet supported for C++`).

- **`internal/setup/references/instability.md`'s Examples section is updated:** C++ line now reads `See \`references/lang/cpp/instability.md\`` (was `Not yet supported for C++`).

- **`internal/setup/references/abstractness.md`'s Examples section is updated:** C++ line now reads `See \`references/lang/cpp/abstractness.md\`` (was `Not yet supported for C++`).

- **`internal/setup/references/lang/cpp/index.md`'s "Available Checks" list is updated** to list four items: `funclen`, `filelen`, `instability`, `abstractness`. The "Limitations" section remains unchanged (CRAP scoring and `boy-scout:ignore` are genuinely unsupported for C++, verified against `cppfunclen.go:229`).

- **Assertions:** None. This is a pure documentation change; no runtime behavior changes. The new unit tests (listed in AC below) serve as the check.

## AC

|AC|Category|Verification Method|
|--|--|--|
|Given `Run()` writes the skill bundle - When it completes - Then `references/lang/cpp/filelen.md`, `references/lang/cpp/instability.md`, `references/lang/cpp/abstractness.md` all exist|Normal|unit test: `internal/setup/setup_test.go: TestRun_WritesCppFilelenInstabilityAbstractnessReferences`|
|Given the 3 new cpp lang files - When read - Then `filelen.md` contains `#include`, `instability.md` contains `domain.hpp`, `abstractness.md` contains `pure virtual` (each a real C++-specific marker for that check's example)|Normal|unit test: `internal/setup/setup_test.go: TestRun_CppLangReferenceFilesContainCppExamples` (3 subtests)|
|Given `SKILL.md` after this change - When read - Then it contains the literal paths `references/lang/cpp/filelen.md`, `references/lang/cpp/instability.md`, `references/lang/cpp/abstractness.md`|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateNoLongerMarksCppFilelenInstabilityAbstractnessUnsupported`|
|Given `references/lang/cpp/index.md` after this change - When read - Then it mentions `filelen`, `instability`, and `abstractness` (not just `funclen`)|Normal|unit test: `internal/setup/setup_test.go: TestRun_CppIndexListsFilelenInstabilityAbstractness`|
|Given the top-level `references/{filelen,instability,abstractness}.md` - When read - Then each one's Examples section contains its matching `references/lang/cpp/<kind>.md` path (no `Not yet supported for C++` for these three)|Boundary|unit test: `internal/setup/setup_test.go: TestRun_TopLevelReferencesPointToCppExamples`|
|Given a built `boy-scout` binary run end-to-end (`run(["setup","agents"], ...)`) in a scratch dir - When it completes - Then `.agents/skills/boy-scout/references/lang/cpp/{filelen,instability,abstractness}.md` all exist on disk|Normal|integration test (built CLI path): `cmd/boy-scout/main_setup_test.go: TestRun_SetupWritesCppFilelenInstabilityAbstractnessReferences`|
|Given the full repo after this change - When `make check` runs - Then it builds, vets, and all tests pass (223+ baseline + 6 new, no regressions)|Exception (regression lock)|build: `make check`|
