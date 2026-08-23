# Claude Code Integration

## Business value
Enable Claude Code to use gardener for automatic code quality fixes, shipping pre-written skills that guide Claude through the fix-loop for both funclen and CRAP violations without human intervention beyond the initial invocation.

## Completion criteria
Every Story below shipped.

## Overview
The first Story in this EPIC is the `gardener setup` subcommand, which auto-installs a Claude Code skill (SKILL.md) that runs gardener, detects violations, and guides Claude through fixing them one at a time using a simple loop: run `gardener all`, identify violations, edit the function, re-run checks, repeat until clean or give up after 3 attempts per violation. Future Stories will add real-time fix hooks and interactive violation navigation.

## Stories
- [setup-command](./setup-command.md)
- [setup-multi-agent-targets](./setup-multi-agent-targets.md)
- [tdd-refactor-and-violation-guidance](./tdd-refactor-and-violation-guidance.md)
- [filelen-violation-guidance](./filelen-violation-guidance.md)
- [violation-resolve-references](./violation-resolve-references.md)
- [fix-run-cap-and-priority](./fix-run-cap-and-priority.md)
