---
type: Spec Story
title: setup-command
description: Add gardener setup [--global] subcommand to write a Claude Code skill
tags: [spec, claude-code-integration]
timestamp: 2026-08-22T00:00:00+09:00
---

# setup-command

## Value to user
Users can run `gardener setup` to install a Claude Code skill and the gardener binary. The skill runs `gardener all`, finds violations, and guides Claude through fixing them one at a time (funclen violations first), re-running checks after each fix, giving up on a violation after 3 failed attempts, and reporting the final state. The skill and binary are project-local (shareable via git) by default or agent-global with `--global`.

## Completion criteria
`gardener setup [--global]` successfully writes a user-invoked SKILL.md file and copies the gardener binary to `./.agents/skills/gardener/SKILL.md` and `./.agents/bin/gardener` (project-local) or `~/.agents/skills/gardener/SKILL.md` and `~/.agents/bin/gardener` (global), creating parent directories as needed, overwriting silently if files exist, and instructing Claude to run the auto-fix loop.

## Spec

The `gardener setup` subcommand:
- Takes no required arguments, zero or one positional argument flag `--global`
- No flag → writes `./.agents/skills/gardener/SKILL.md` and `./.agents/bin/gardener` (relative to cwd, project-local, git-shareable)
- `--global` → writes `~/.agents/skills/gardener/SKILL.md` and `~/.agents/bin/gardener` (agent-global location, shareable across tools)
- Creates any missing parent directories with `os.MkdirAll`
- Copies the current executable to the `.agent/bin/` location
- Overwrites silently if the files already exist (safe to re-run after upgrading gardener)
- Extra positional args after `setup` is a usage error (stderr + exit 2), matching `funclen`/`crap`/`all`
- Write failure (permission denied, path is wrong type, etc.) is stderr + exit 2, no partial files
- Prints `gardener skill installed: <path>` to stdout on success, exits 0
- On error, prints the error to stderr, exits 2

Generated SKILL.md:
- Frontmatter: `name: gardener`, `description: Auto-fix code quality violations from the gardener lint checker`, `disable-model-invocation: true` (user-invoked only, never fires itself)
- Body instructs Claude to: run `gardener all`; for each violation found, fix funclen violations before crap violations one at a time; after each edit, re-run `gardener all` + `go test ./...`; on green (all checks pass), move to the next violation; after 3 failed attempts on one violation mark it unresolved and continue; at the end report fixed vs. unresolved count; if `gardener all` finds zero violations, report clean and make no edits; never commit changes with git.

Bare `gardener` with no subcommand prints usage to stderr listing `funclen, crap, all, setup` and exits non-zero.

Dependencies: Go stdlib only. No third-party modules. Embeds the SKILL.md template at compile time.

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a tmp dir with no `.agents/` yet, used as baseDir - When `setup.Run(baseDir, binaryPath)` runs - Then `baseDir/.agents/skills/gardener/SKILL.md` and `baseDir/.agents/bin/gardener` exist, skill is non-empty, and the returned path is the skill path|Normal|unit test: `internal/setup/setup_test.go: TestRun_CreatesSkillFileAtBaseDir`|
|Given `baseDir/.agents/skills/gardener/SKILL.md` already exists with stale content - When `setup.Run(baseDir, binaryPath)` runs again - Then both files are overwritten, no error, no `--force` needed|Normal|unit test: `internal/setup/setup_test.go: TestRun_OverwritesExistingSkillFile`|
|Given the embedded skill template - When read - Then it contains `name: gardener`, `description:`, `disable-model-invocation: true`, `gardener all`, and `go test`|Normal|unit test: `internal/setup/setup_test.go: TestRun_TemplateDeclaresUserInvokedFixLoop`|
|Given `gardener setup` run from a cwd with no `.agents/` dir - When `run(["setup"], ...)` runs - Then exit code 0, stdout contains `gardener skill installed: `, and both `./.agents/skills/gardener/SKILL.md` and `./.agents/bin/gardener` exist relative to that cwd|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupLocalWritesRelativeSkillFile`|
|Given `HOME` set to a tmp dir - When `run(["setup", "--global"], ...)` runs - Then exit code 0 and both `$HOME/.agents/skills/gardener/SKILL.md` and `$HOME/.agents/bin/gardener` exist|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupGlobalWritesToHomeDir`|
|Given `gardener setup extra-arg` (stray positional arg) - When run - Then stderr contains usage text, exit code 2, no files written|Exception|unit test: `cmd/gardener/main_test.go: TestRun_SetupRejectsExtraArgs`|
|Given a baseDir the process cannot write to (e.g. a file created at the `.agents` path, blocking `MkdirAll`) - When `setup.Run(baseDir, binaryPath)` runs - Then it returns a non-nil error and `run(["setup"], ...)` exits 2 with the error on stderr|Exception|unit test: `internal/setup/setup_test.go: TestRun_ReturnsErrorWhenDirUnwritable` + `cmd/gardener/main_test.go: TestRun_SetupWriteFailureExitsTwo`|
|Given `gardener all` finds zero violations (per the SKILL.md instructions, not code) - When the skill is invoked by a human in a real repo - Then it reports clean and makes no edits|Boundary|manual test: run the installed skill against a clean gardener checkout; not unit-testable, the loop logic is prose Claude executes|
