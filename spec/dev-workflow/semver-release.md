---
type: Spec Story
title: semver-release
description: Add a SemVer release tool (internal/release + cmd/release + make release)
tags: [spec, dev-workflow, release, semver]
timestamp: 2026-08-23T17:00:00+09:00
---

# Add a SemVer release tool (internal/release + cmd/release + make release)

## Value to user
Releases are automated and repeatable: developers run `make release`, and the tool computes the next version from commit history (using the Conventional Commits convention enforced by Story 1), tags the release, rebuilds with the version baked in, installs the binary, updates CHANGELOG.md, and pushes a GitHub Release — all gated behind an explicit human confirmation, never automatic.

## Completion criteria
- `internal/release.NextVersion(lastTag string, commitSubjects []string) (next string, ok bool)` computes the next SemVer tag from Conventional Commit types: `fix` → patch, `feat` → minor, breaking (`!` suffix or `BREAKING CHANGE:` footer) → minor while `lastTag`'s major is `0`, else major; highest-priority bump wins when multiple types are present; returns `ok=false` when no bump-worthy commit exists.
- When `lastTag == ""` (first-ever release) and at least one bump-worthy commit exists, `NextVersion` returns `v0.1.0`.
- `internal/release.ChangelogEntry(version string, commitSubjects []string) string` renders a Markdown section (`## vX.Y.Z`) grouping `feat`/`fix` commit subjects under "Features"/"Fixes" headers; `docs`/`style`/`refactor`/`perf`/`test`/`chore` subjects are excluded from the changelog body.
- `cmd/boy-scout` gains a lang-less `version` subcommand (`boy-scout version`) printing `boy-scout <version>`, where `version` is overridable via `-ldflags -X main.version=...` (defaults to `"dev"`).
- `cmd/release` is a thin CLI that reads the latest git tag and commit subjects since it, calls `internal/release.NextVersion`, and prints the computed version (or the literal string `none`) to stdout; supports a `-changelog` flag to print the changelog entry instead.
- `make release` refuses to run against a dirty working tree (`git status --porcelain` non-empty); computes the next version via `cmd/release`; exits cleanly with a message when the answer is `none`; otherwise builds the binary with the version baked in, copies it to `~/.agents/bin/boy-scout` and `~/.claude/bin/boy-scout`, prepends a `CHANGELOG.md` entry, commits `CHANGELOG.md`, tags the release commit, pushes `main` and the tag, and creates a GitHub Release via `gh release create <tag> --generate-notes`.

## Spec

### NextVersion function
- Parse `lastTag` (empty or `vMAJOR.MINOR.PATCH` via regexp); if parsing fails, treat as "no tag"
- Classify each subject with Conventional Commits pattern (same as `.githooks/commit-msg`); skip Merge/Revert subjects
- Pick the highest-priority bump: breaking (bumpMajor) > feat (bumpMinor) > fix (bumpPatch)
- Apply the pre-1.0 rule: if major version is 0, breaking changes bump minor, not major, per SemVer spec
- If `lastTag == ""` and at least one bump-worthy commit, return `v0.1.0`
- If no bump-worthy commits, return `("", false)`
- Assert: if major == 0, bump must never be bumpMajor (pre-1.0 invariant)
- Assert: if tag matches the pattern, parsed major/minor/patch must be non-negative

### ChangelogEntry function
- Given a version string and commit subjects, return a Markdown section: `## vX.Y.Z\n\n### Features\n- ...\n\n### Fixes\n- ...\n`
- Include only `feat` subjects under Features and `fix` subjects under Fixes
- Exclude `docs`, `style`, `refactor`, `perf`, `test`, `chore` subjects from output
- Each subject's description (everything after `type(scope)?!?: `) is listed as a bullet point

### boy-scout version subcommand
- Add a package-level var `version = "dev"` in `cmd/boy-scout/main.go`
- Add a lang-less branch in `run()`: if `args[0] == "version"`, print `boy-scout <version>` to stdout and return 0
- The version can be overridden at build time via `-ldflags -X main.version=vX.Y.Z`

### cmd/release CLI
- Read the latest git tag via `git describe --tags --abbrev=0` (tolerate error as "no tag")
- Read commit subjects since the tag via `git log <tag>..HEAD --format=%s` (or all commits if no tag)
- Call `internal/release.NextVersion` with the tag and subjects
- If ok is false (no bump-worthy commits), print `none` and exit 0
- Otherwise, print the version string and exit 0
- Support a `-changelog` flag: if set, call `internal/release.ChangelogEntry` and print that instead of the version

### Makefile release target
- Check for dirty working tree: `git status --porcelain` must be empty; exit 1 with message if not
- Run `go run ./cmd/release` to compute the next version
- Exit cleanly with a message if the version is `none`
- Build the binary with the version baked in: `go build -ldflags "-X main.version=$(NEXT)" -o $(BINARY) $(MAIN_PKG)`
- Copy to `~/.agents/bin/boy-scout` and `~/.claude/bin/boy-scout`
- Compute the changelog entry via `go run ./cmd/release -changelog`
- Prepend the entry to `CHANGELOG.md`
- Commit as `chore(release): $(NEXT)`
- Tag the release commit: `git tag $(NEXT)`
- Push main and the tag: `git push origin main && git push origin $(NEXT)`
- Create a GitHub Release: `gh release create $(NEXT) --generate-notes`

## Acceptance criteria

| AC | Category | Verification Method |
| -- | -- | -- |
| Given lastTag `v0.4.0` and commit subjects `["fix: x", "feat: add abstractness flag"]` - When `NextVersion` runs - Then it returns `("v0.5.0", true)` | Normal | unit test: `internal/release/release_test.go: TestNextVersion_FeatBeatsFix` |
| Given lastTag `v0.4.0` and commit subjects `["fix: x"]` - When `NextVersion` runs - Then it returns `("v0.4.1", true)` | Normal | unit test: `internal/release/release_test.go: TestNextVersion_FixOnlyIsPatch` |
| Given lastTag `""` and commit subjects `["feat: initial CLI"]` - When `NextVersion` runs - Then it returns `("v0.1.0", true)` | Boundary | unit test: `internal/release/release_test.go: TestNextVersion_NoPriorTagStartsAtZeroOneZero` |
| Given lastTag `v0.4.1` and commit subjects `["feat!: rename flag"]` - When `NextVersion` runs - Then it returns `("v0.5.0", true)` (minor, not major, because the tag's major is still `0`) | Boundary | unit test: `internal/release/release_test.go: TestNextVersion_BreakingPre1_0BumpsMinorNotMajor` |
| Given lastTag `v0.4.1` and commit subjects `["docs: fix typo", "chore: cleanup"]` - When `NextVersion` runs - Then it returns `("", false)` | Exception | unit test: `internal/release/release_test.go: TestNextVersion_NoBumpWorthyCommitsReturnsNotOK` |
| Given commit subjects `["Merge branch 'x' into main", "fix: y"]` and lastTag `v0.4.1` - When `NextVersion` runs - Then the merge subject contributes no bump on its own and the result equals `("v0.4.2", true)`, same as if it weren't present | Boundary | unit test: `internal/release/release_test.go: TestNextVersion_IgnoresMergeAndRevertSubjects` |
| Given a temp git fixture repo with seeded tags/commits - When the built `cmd/release` binary runs with that repo as its working directory - Then it prints a version string matching `^v[0-9]+\.[0-9]+\.[0-9]+$` or the literal `none`, never crashes | Normal | integration test: `cmd/release/main_test.go: TestMain_PrintsVersionOrNone` |
| Given a working tree with an uncommitted change - When `make release` runs - Then it exits non-zero with a "working tree not clean" message and creates no tag | Exception | e2e: manual run against a scratch dirty copy (QA step 5) |
| Given a binary built with `-ldflags "-X main.version=v0.5.0"` - When run as `boy-scout version` - Then it prints `boy-scout v0.5.0` | Normal | integration test: `cmd/boy-scout/main_test.go: TestRun_VersionPrintsBuiltVersion` |
| Given the real repo on `main`, clean tree, at least one bump-worthy commit since the last tag - When `make release` runs - Then a new tag exists locally and on `origin`, `gh release view <tag>` succeeds, `CHANGELOG.md` has a new entry, and both installed binaries report the new version via `version` | Normal | e2e: manual run of the real `make release` (QA step 6, requires explicit release confirmation) |
