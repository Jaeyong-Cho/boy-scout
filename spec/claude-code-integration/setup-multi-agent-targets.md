---
type: Spec Story
title: setup-multi-agent-targets
description: Extend gardener setup to take a target agent (claude/copilot/pi/agents), with an interactive picker when none is given
tags: [spec, claude-code-integration]
timestamp: 2026-08-22T00:00:00+09:00
---

# setup-multi-agent-targets

## Value to user
Users can run `gardener setup <claude|copilot|pi|agents>` to install the gardener skill and binary under that agent's project-local skill directory, supporting per-agent configurations. When run without a target, an interactive prompt (on real terminals) helps the user choose, while scripts and CI get a clear error listing valid targets.

## Completion criteria
`gardener setup [claude|copilot|pi|agents] [--global]` successfully writes the same SKILL.md content and copies the same binary to the correct directory prefix for each target (`.claude`, `.copilot`, `.pi/agent`, or `.agents`). When no target is given and stdin is interactive, the user sees a numbered picker. When no target is given and stdin is non-interactive (piped/CI), the command lists valid targets and exits 2. The `--global` flag composes in either position.

## Spec

The `gardener setup` subcommand:
- Takes an optional positional `<target>` argument: `claude`, `copilot`, `pi`, or `agents` (the last is today's default, kept for back-compat)
- Optional `--global` flag, works in either position: `gardener setup claude --global` or `gardener setup --global claude`
- Each target writes the same embedded SKILL.md content and copies the same binary, only the directory prefix changes:
  - claude: `.claude/skills/gardener/SKILL.md` and `.claude/bin/gardener`
  - copilot: `.copilot/skills/gardener/SKILL.md` and `.copilot/bin/gardener`
  - pi: `.pi/agent/skills/gardener/SKILL.md` and `.pi/agent/bin/gardener` (extra `agent/` segment)
  - agents: `.agents/skills/gardener/SKILL.md` and `.agents/bin/gardener`
- No target and stdin is a real terminal → print a numbered list (`1) claude  2) copilot  3) pi  4) agents`), read one line, accept either the number or the name
- No target and stdin is not a terminal (piped/CI) → print the valid target names to stderr and exit 2, no files written, no hang
- Unknown target name (typed explicitly or via the picker) → usage error to stderr listing the 4 valid names, exit 2, no files written
- More than one positional argument after `setup` → usage error, exit 2 (unchanged from before, just re-validated against the new "one positional = target" shape)
- Codex is **not** a valid target and is not mentioned anywhere — out of scope
- SKILL.md content itself is unchanged — still assumes a bare `gardener` on `$PATH`; the per-target binary copy stays purely for parity with the existing agents convention

Flag scanning:
- Manual scan for `--global`/`-global` flags (do not use `flag.NewFlagSet` for setup's boolean, which would silently ignore trailing flags after positional args)
- This fixes the ordering bug where `flag.Parse` would stop at the first positional arg

## AC
|AC|Category|Verification Method|
|--|--|--|
|Given a tmp dir with no `.claude/` yet, used as baseDir - When `setup.Run(baseDir, binaryPath, ".claude")` runs - Then `baseDir/.claude/skills/gardener/SKILL.md` and `baseDir/.claude/bin/gardener` exist and the returned path is the skill path|Normal|unit test: `internal/setup/setup_test.go: TestRun_UsesGivenPrefixForSkillAndBinPaths`|
|Given `gardener setup claude` run from a cwd with no `.claude/` dir - When `run(["setup", "claude"], ...)` runs - Then exit code 0 and both `./.claude/skills/gardener/SKILL.md` and `./.claude/bin/gardener` exist|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupClaudeWritesClaudeSkillPath`|
|Given `gardener setup copilot` - When run - Then `./.copilot/skills/gardener/SKILL.md` and `./.copilot/bin/gardener` exist|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupCopilotWritesCopilotSkillPath`|
|Given `gardener setup pi` - When run - Then `./.pi/agent/skills/gardener/SKILL.md` and `./.pi/agent/bin/gardener` exist (the extra `agent/` segment)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupPiWritesPiAgentSkillPath`|
|Given `gardener setup agents` (today's pre-existing default, now explicit) - When run - Then `./.agents/skills/gardener/SKILL.md` and `./.agents/bin/gardener` exist, same as before this change|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupLocalWritesRelativeSkillFile` (updated to pass `agents` explicitly)|
|Given `HOME` set to a tmp dir - When `run(["setup", "agents", "--global"], ...)` runs - Then `$HOME/.agents/skills/gardener/SKILL.md` and `$HOME/.agents/bin/gardener` exist|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupGlobalWritesToHomeDir` (updated to pass `agents` explicitly)|
|Given `gardener setup bogus` (unknown target name) - When run - Then stderr lists the 4 valid target names, exit code 2, no directory created|Exception|unit test: `cmd/gardener/main_test.go: TestRun_SetupUnknownTargetIsUsageError`|
|Given `gardener setup agents extra-arg` (two positional args) - When run - Then stderr contains usage text, exit code 2, no directory created|Exception|unit test: `cmd/gardener/main_test.go: TestRun_SetupRejectsExtraArgs` (updated to include the target)|
|Given no target and stdin that is not a terminal (a `strings.Reader`, as in any script/CI invocation) - When `runSetup(nil, stdin, stdout, stderr)` runs - Then stderr lists the 4 valid target names, exit code 2, no files written, no hang|Boundary|unit test: `cmd/gardener/main_test.go: TestRun_SetupNoTargetNonInteractiveIsUsageError`|
|Given stdin text `"2\n"` - When `promptForTarget(stdin, stdout)` runs - Then it returns `"copilot"` and stdout shows the numbered list|Normal|unit test: `cmd/gardener/main_test.go: TestPromptForTarget_NumericSelection`|
|Given stdin text `"pi\n"` - When `promptForTarget(stdin, stdout)` runs - Then it returns `"pi"`|Normal|unit test: `cmd/gardener/main_test.go: TestPromptForTarget_NameSelection`|
|Given stdin text `"nope\n"` or empty/EOF stdin - When `promptForTarget(stdin, stdout)` runs - Then it returns a non-nil error|Exception|unit test: `cmd/gardener/main_test.go: TestPromptForTarget_InvalidSelectionReturnsError`|
|Given a baseDir the process cannot write to - When `setup.Run(baseDir, binaryPath, prefix)` runs with any prefix - Then it returns a non-nil error and `run(["setup", "agents"], ...)` exits 2 with the error on stderr|Exception|unit test: `internal/setup/setup_test.go: TestRun_ReturnsErrorWhenDirUnwritable` + `cmd/gardener/main_test.go: TestRun_SetupWriteFailureExitsTwo` (both updated to pass `.agents`/`agents` explicitly)|
|Given a real terminal running `gardener setup` with no target - When a human picks an option from the numbered list - Then the corresponding target's files are written, matching the manual QA step|Boundary|manual test: QA step 7|
|Given `run(["setup", "claude", "--global"], ...)` and separately `run(["setup", "--global", "claude"], ...)` - When both run - Then both succeed, writing under `$HOME/.claude/...` (confirms flag composability regardless of position)|Normal|unit test: `cmd/gardener/main_test.go: TestRun_SetupGlobalFlagComposesInEitherOrder`|
