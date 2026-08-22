---
type: Spec Story
title: rename-go-to-gardener-go
description: Rename Go gardener tool from 'gardener' to 'gardener-go', first of per-language naming scheme
tags: [spec, multi-language-variants]
timestamp: 2026-08-22T19:18:00+09:00
---

# rename-go-to-gardener-go

## Value to user
Users can now clearly distinguish the Go gardener tool as `gardener-go` instead of the ambiguous `gardener`, supporting future per-language tool variants (e.g., `gardener-py`, `gardener-ts`) without naming collisions.

## Completion criteria
Go module path, binary name, CLI command, skill name, and install paths all renamed from `gardener` to `gardener-go`. The `// gardener:ignore` suppression-comment keyword remains unchanged (shared across all future language versions). All tests pass with zero remaining `go-gardener` string references in `.go` files.

## Spec

### Module and binary rename
- Go module path renamed: `go-gardener` → `gardener-go` (in `go.mod`)
- CLI command renamed: `gardener` → `gardener-go` (built binary, `cmd/gardener-go/`, all usage/help text, success message)
- `cmd/gardener/` directory → `cmd/gardener-go/`

### Skill name
- Claude Code skill renamed: `name: gardener` → `name: gardener-go` in the embedded `SKILL.md` frontmatter and body
- All references to `gardener all` in the skill template → `gardener-go all`

### Install paths
- Project-local paths: `.agents/skills/gardener/` → `.agents/skills/gardener-go/`, `.agents/bin/gardener` → `.agents/bin/gardener-go`
- Global paths (`--global` flag): `~/.agents/skills/gardener/` → `~/.agents/skills/gardener-go/`, `~/.agents/bin/gardener` → `~/.agents/bin/gardener-go`

### Suppression comment
- `// gardener:ignore` keyword is explicitly NOT renamed (remains `gardener:ignore`, not `gardener-go:ignore`)

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given the repo after the module/import rename - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass, with zero remaining `go-gardener` string references in any `.go` file|Normal|build: `go build ./... && go test ./...`, plus `grep -rl "go-gardener" --include="*.go" .` finds no matches|
|Given a tmp baseDir with no `.agents/` yet - When `setup.Run(baseDir, binaryPath)` runs - Then `baseDir/.agents/skills/gardener-go/SKILL.md` and `baseDir/.agents/bin/gardener-go` exist|Normal|unit test: `internal/setup/setup_test.go: TestRun_CreatesSkillFileAtBaseDir` (updated)|
|Given the embedded skill template - When read - Then it contains `name: gardener-go` and `gardener-go all` (not `name: gardener` / bare `gardener all`)|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresUserInvokedFixLoop` (updated)|
|Given `gardener-go setup` run from a cwd with no `.agents/` dir - When `run(["setup"], ...)` runs - Then exit 0, stdout contains `gardener-go skill installed:`, and `./.agents/skills/gardener-go/SKILL.md` + `./.agents/bin/gardener-go` exist|Normal|unit test: `cmd/gardener-go/main_test.go: TestRun_SetupLocalWritesRelativeSkillFile` (updated)|
|Given `HOME` set to a tmp dir - When `run(["setup", "--global"], ...)` runs - Then `$HOME/.agents/skills/gardener-go/SKILL.md` and `$HOME/.agents/bin/gardener-go` exist|Normal|unit test: `cmd/gardener-go/main_test.go: TestRun_SetupGlobalWritesToHomeDir` (updated)|
|Given `run([], ...)` (no subcommand) - When it prints usage - Then stderr contains `gardener-go` (not bare `gardener`)|Normal|unit test: `cmd/gardener-go/main_test.go: TestRun_NoSubcommandPrintsUsage` (updated with added assertion)|
|Given the `internal/crap/crap.go` / `internal/funclen/funclen.go` ignore-directive logic - When their existing tests run unmodified - Then `// gardener:ignore` still works exactly as before (string not touched by this rename)|Boundary|unit test: `internal/crap/crap_test.go` + `internal/funclen/funclen_test.go` (existing ignore-directive tests, run as-is, must stay green)|
|Given the repo root - When `go build -o bin/gardener-go ./cmd/gardener-go` runs - Then it builds with no errors and produces `bin/gardener-go`|Normal|build: `go build -o bin/gardener-go ./cmd/gardener-go`|
|Given the built `bin/gardener-go` binary - When `./bin/gardener-go setup --global` runs on this machine - Then `~/.agents/skills/gardener-go/SKILL.md` and `~/.agents/bin/gardener-go` exist|Normal|manual test: run the command and verify files exist|
