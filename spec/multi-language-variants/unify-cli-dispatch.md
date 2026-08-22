---
type: Spec Story
title: unify-cli-dispatch
description: Restructure gardener CLI into gardener {lang} {command} dispatch, reversing per-language binary naming
tags: [spec, multi-language-variants]
timestamp: 2026-08-22T21:36:00+09:00
---

# unify-cli-dispatch

## Value to user
One `gardener` tool with `gardener {lang} {command}` dispatch is simpler to install and matches how a solo user actually wants to run checks across repos of different languages, instead of maintaining a separate binary per language. This design reverses the per-binary-per-language naming scheme decided in `rename-go-to-gardener-go`.

## Completion criteria
CLI shape changes from `gardener-go <subcommand>` to `gardener <lang> <subcommand> [flags] [paths...]`. Binary renamed back to `gardener`, skill renamed back to `gardener`, and install paths use `gardener` (not `gardener-go`). The only language wired in this STORY is `go`; `funclen`, `crap`, and `all` subcommands all work identically to before, just via `gardener go <subcommand>`. Unknown language and unknown subcommand both exit 2 with clear stderr messages. All tests pass (97 baseline + 2 new tests for error cases = 99).

## Spec

### CLI dispatch shape
- Binary name: `gardener-go` → `gardener`
- CLI shape changes from `gardener-go <subcommand>` to `gardener <lang> <subcommand> [flags] [paths...]`
- Available languages in this STORY: `go`
- Available subcommands for `go`: `funclen`, `crap`, `all`
- Unknown language (e.g. `gardener rust funclen`) exits 2, stderr: `"unknown language: rust"`
- Unknown subcommand for a known language (e.g. `gardener go bogus`) exits 2, stderr: `"unknown subcommand for go: bogus"`
- `gardener setup [--global]` remains lang-less (top-level subcommand)

### Module path (intentionally unchanged)
- Go module path stays `gardener-go` in `go.mod` and import statements (internal, user-invisible)
- Only the CLI surface and install paths change, not the module path

### Repo directory
- Directory `gardener-go/` → `gardener/` (plain `mv`, doesn't touch git history)
- Subdirectory `cmd/gardener-go/` → `cmd/gardener/` (file `main.go` inside stays unchanged)

### Skill and setup
- Skill name: `gardener-go` → `gardener` (in embedded `SKILL.md` frontmatter `name:`)
- Skill body: all references to `gardener-go all` → `gardener go all`, all references to `gardener-go <subcommand>` → `gardener go <subcommand>`
- Success message: `"gardener-go skill installed: %s"` → `"gardener skill installed: %s"`
- Usage text: `"usage: gardener-go setup [--global]"` → `"usage: gardener setup [--global]"`
- Install paths: `.agents/skills/gardener-go/` → `.agents/skills/gardener/`, `.agents/bin/gardener-go` → `.agents/bin/gardener`
- Help text: `"install to ~/.agents/skills/gardener-go instead of ./.agents/skills/gardener-go"` → `"install to ~/.agents/skills/gardener instead of ./.agents/skills/gardener"`

### Command implementation
- Remove flat `subcommands` map
- Replace with `langSubcommands map[string]map[string]func...]` keyed by language then subcommand
- Rename functions: `runFunclen` → `runGoFunclen`, `runCrap` → `runGoCrap`, `runAll` → `runGoAll` (bodies unchanged, only names)
- Dispatch logic: if `args[0] == "setup"`, call `runSetup` (lang-less); else treat `args[0]` as language, look up in `langSubcommands`, exit 2 with "unknown language" if missing; then look up `args[1]` as subcommand, exit 2 with "unknown subcommand for {lang}" if missing or `args[1]` absent
- Add runtime assertion before calling subcommand handler: `assertf(fn != nil, "registered subcommand handler for %s/%s is nil", lang, subcommand)`

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given the renamed repo - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass with zero remaining `gardener-go` string references in CLI-facing code (usage text, skill template, install paths)|Normal|build: `go build ./... && go test ./...` (expect 99 tests: 97 baseline + 2 new), plus `grep -rl "gardener-go" --include="*.go" --include="*.md" cmd internal` finds no CLI-facing matches (package-internal Go import path `gardener-go/internal/...` is the one intentional exception)|
|Given `run(["go","funclen","--max-lines=10"], ...)` on a fixture dir - When it runs - Then it behaves identically to today's `run(["funclen","--max-lines=10"], ...)` (same violations, same exit code)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_FunclenRespectsMaxLinesFlag` (updated to prepend `"go"`)|
|Given `run(["go","crap", ...], ...)` and `run(["go","all", ...], ...)` - When they run - Then they behave identically to today's `run(["crap",...])`/`run(["all",...])`|Normal|unit test: every existing `TestRun_Crap*`/`TestRun_All*` case in `cmd/gardener/main_test.go`, updated to prepend `"go"`|
|Given `run(["setup"], ...)` and `run(["setup","--global"], ...)` - When they run - Then they still work exactly as today, just installing under the `gardener` name instead of `gardener-go`|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupLocalWritesRelativeSkillFile`, `TestRun_SetupGlobalWritesToHomeDir` (updated paths)|
|Given `run(["rust","funclen"], ...)` (unknown language) - When it runs - Then stderr contains "unknown language: rust" and exit code is 2|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_UnknownLanguagePrintsUsage` (new)|
|Given `run(["go","bogus"], ...)` (unknown subcommand for a known language) - When it runs - Then stderr contains "unknown subcommand for go: bogus" and exit code is 2|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_UnknownSubcommandForLangPrintsUsage` (new)|
|Given `run([], ...)` (no args at all) - When it runs - Then stderr's usage line mentions `gardener <lang> <command>` and lists at least `go` as an available language, exit code 2|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_NoSubcommandPrintsUsage` (updated assertion)|
|Given the embedded skill template - When read - Then it contains `name: gardener` (not `name: gardener-go`) and instructs `gardener go all` (not bare `gardener-go all` or `gardener all`)|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresUserInvokedFixLoop` (updated)|
|Given the built `bin/gardener` binary - When `./bin/gardener setup --global` runs on this machine - Then `~/.agents/skills/gardener/SKILL.md` and `~/.agents/bin/gardener` exist, and the stale `~/.agents/skills/gardener-go`, `~/.agents/bin/gardener-go`, `~/.claude/skills/gardener-go` no longer exist|Normal|manual test: run the command, then `ls ~/.agents/skills/`, `ls ~/.agents/bin/`, `ls ~/.claude/skills/`|

