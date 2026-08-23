---
type: STORY
title: Rename gardener to boy-scout
parent_epic: multi-language-variants
parent_epic_id: multi-language-variants
state: in-progress
date_started: 2026-08-23
---

# rename-gardener-boy-scout

## Value to user
Reinforces the tool's mission—leave code cleaner than you found it—by using a name that reflects its core behavior: auditing code like a Boy Scout auditing a campsite before handing it back. The tool checks off every concern before returning to a clean state.

## Completion criteria

|AC|Category|Verification Method|
|--|--|--|
|Given the repo after the module/import/marker rename - When `go build ./... && go test ./...` runs - Then it compiles and all tests pass, with zero remaining `gardener` (case-insensitive) string references in any `.go`, `.md`, `Makefile`, or `.gitignore` file under the repo|Normal|build: `go build ./... && go test ./...`, plus `grep -rli gardener --include="*.go" --include="*.md" --include="Makefile" --include=".gitignore" .` finds no matches|
|Given a Go source fixture using the new marker `// boy-scout:ignore:crap` - When `internal/funcignore.Reason` runs on it - Then the function is excluded with reason `"comment"`|Normal|unit test: `internal/funcignore/funcignore_test.go` (updated fixtures)|
|Given a Go source fixture still using the *old* marker `// gardener:ignore` - When `internal/funcignore.Reason` runs on it - Then the function is **not** excluded (proves the marker actually changed, not aliased)|Boundary|unit test: `internal/funcignore/funcignore_test.go` (new case added for the old marker)|
|Given a tmp baseDir with no `.agents/` yet - When `setup.Run(baseDir, binaryPath, ".agents")` runs - Then `baseDir/.agents/skills/boy-scout/SKILL.md` and `baseDir/.agents/bin/boy-scout` exist|Normal|unit test: `internal/setup/setup_test.go: TestRun_CreatesSkillFileAtBaseDir` (updated)|
|Given the embedded skill template - When read - Then it contains `name: boy-scout` and `boy-scout go all` (not `name: gardener` / `gardener go all`)|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresUserInvokedFixLoop` (updated)|
|Given `run(["setup"], ...)` from a cwd with no `.agents/` dir - When it runs - Then exit 0, stdout contains `boy-scout skill installed:`, and `./.agents/skills/boy-scout/SKILL.md` + `./.agents/bin/boy-scout` exist|Normal|unit test: `cmd/boy-scout/main_setup_test.go: TestRun_SetupLocalWritesRelativeSkillFile` (updated)|
|Given `run([], ...)` (no subcommand) - When it prints usage - Then stderr contains `boy-scout <lang> <command>` and `boy-scout setup` (not bare `gardener`)|Normal|unit test: `cmd/boy-scout/main_test.go: TestRun_NoSubcommandPrintsUsage` (updated)|
|Given the built `bin/boy-scout` binary - When `./bin/boy-scout setup --global` runs on this machine - Then `~/.agents/skills/boy-scout/SKILL.md` and `~/.agents/bin/boy-scout` exist, and the stale `~/.agents/skills/gardener`, `~/.agents/bin/gardener`, `~/.claude/skills/gardener` no longer exist|Normal|manual test: run the command, then `ls ~/.agents/skills/`, `ls ~/.agents/bin/`, `ls ~/.claude/skills/`|
|Given the previously-existing, unrelated `~/.claude/skills/boy-scout` (report-only reviewer) - When `./bin/boy-scout setup claude --global` runs - Then `~/.claude/skills/boy-scout/SKILL.md` now contains the gardener auto-fix content (`name: boy-scout`, `boy-scout go all`), not the old report-only content|Boundary|manual test: run the command, then read `~/.claude/skills/boy-scout/SKILL.md` and confirm its content is the renamed gardener skill|

## Spec changes

**STORY: rename-gardener-boy-scout**
- Go module path renamed: `gardener-go` → `boy-scout` (in `go.mod`), all `gardener-go/internal/...` imports updated to `boy-scout/internal/...`.
- CLI command renamed: `gardener` → `boy-scout` (built binary, `cmd/gardener/` → `cmd/boy-scout/`, all usage/help text, the `setup` success message).
- Claude Code skill renamed: `name: gardener` → `name: boy-scout` in `internal/setup/skill.md` frontmatter and body (`gardener go all` → `boy-scout go all`).
- Install paths renamed: `{prefix}/skills/gardener/` → `{prefix}/skills/boy-scout/`, `{prefix}/bin/gardener` → `{prefix}/bin/boy-scout` (covers every prefix: `.agents`, `.claude`, `.copilot`).
- `// gardener:ignore` and `// gardener:ignore:checker[,checker...]` suppression-comment markers renamed to `// boy-scout:ignore` / `// boy-scout:ignore:checker[,checker...]` — explicitly changed this time (opposite of the prior rename's decision).
- Repo folder renamed: `agent-tools/gardener` → `agent-tools/boy-scout`.
- Machine-installed copies refreshed: stale `~/.claude/skills/gardener`, `~/.agents/skills/gardener`, `~/.agents/bin/gardener` removed; `~/.claude/skills/boy-scout` (previously the unrelated report-only skill) and `~/.agents/skills/boy-scout` / `~/.agents/bin/boy-scout` installed in their place.
- Out of scope (record explicitly in the spec too): no rewriting of historical `~/wiki/journal/**` entries; no rewriting of the old `spec/multi-language-variants/rename-go-to-gardener-go.md` spec; no change to check behavior/thresholds; no attempt to preserve the old report-only boy-scout skill's functionality under a different name.
