# Multi-language variants

## Business value
One `gardener` tool supports every language through a `gardener {lang} {command}` dispatch, instead of a separate binary per language.

## Completion criteria
Every Story below shipped.

## Overview
This EPIC establishes a multi-language dispatch architecture: one `gardener` binary with language as a runtime argument. Initial work ([rename-go-to-gardener-go](./rename-go-to-gardener-go.md)) renamed the tool to `gardener-go`, but that per-binary-per-language naming scheme was reversed within the same day via [unify-cli-dispatch](./unify-cli-dispatch.md) — a single tool with `gardener {lang} {command}` dispatch is simpler to install and matches how a solo user actually wants to run checks across repos of different languages. Future Stories will add other language-specific implementations (Python, C++, JavaScript, etc.) as implementation demand arises; those are out of scope for this EPIC.

## Stories
- [rename-go-to-gardener-go](./rename-go-to-gardener-go.md)
- [unify-cli-dispatch](./unify-cli-dispatch.md)
- [cpp-funclen-support](./cpp-funclen-support.md)
- [rename-funclen-gofunclen](./rename-funclen-gofunclen.md)
- [rename-gardener-boy-scout](./rename-gardener-boy-scout.md)
