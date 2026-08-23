# Frontmatter (OKF)

**MUST**, every new document written under `~/wiki/**/*.md` or a target repo's `spec/**/*.md` carry [Open Knowledge Format](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing?hl=en) frontmatter. **MUST** every new first-party source-code file in a target repo carry the same metadata in a language-safe comment header. Source code uses a comment header instead of bare YAML so the metadata cannot break the interpreter or compiler. Out of scope: `index.md`/`log.md`, generated files, dependencies, vendored skills like `teach`/`diagram-design` that keep their own format, and everything else outside `~/wiki`/`spec/` or first-party source code.

Add a YAML frontmatter block at the top of the file, all six fields, every time:

```yaml
---
type: <Kind of document>
title: <Human-readable name>
description: <One-line summary>
resource: <Canonical URI this doc is about, or omit the line if none>
tags: [<tag>, ...]
timestamp: <ISO 8601 datetime, when this content was last meaningfully written>
---
```

- `type` — a short, descriptive string; pick what fits, there's no central registry. Examples already implied by what writes into `~/wiki`/`spec/`: `Journal Entry`, `Handoff`, `Advisor Report`, `Research Explore`, `Research Experiment`, `Research Plan`, `Research Recon`, `Research Context`, `Roadmap Project`, `Roadmap Epic`, `Roadmap Story`, `Goal`, `Spec Story`.
- `title`, `description`, `tags` — what a search index or an `index.md` entry pulls from first.
- `resource` — the canonical URI the doc is *about* (a PR, a dashboard, a repo, an external doc). Most `~/wiki` entries describe work rather than a resource — omit the line rather than inventing one.
- `timestamp` — when the content was last meaningfully written, not the file's mtime.

`index.md` and `log.md` are OKF's reserved filenames — no frontmatter on those. `index.md` is what OKF calls progressive disclosure: a directory listing linking each document with its one-line description, letting an agent see what exists before opening any one file. This is already the convention throughout `~/wiki` and `spec/` (journal/research/roadmap index chains, `spec/index.md` → `spec/{epic-slug}/index.md`); nothing new to add there beyond keeping those links current.

## Reading documents or source code (progressive disclosure)

**Frontmatter/header first, always** — read it before the body, on every `.md` file under `~/wiki` or a target repo's `spec/`, and on every first-party source-code file, even when the path is already known. For documents, read the YAML frontmatter. For source code, read only the language-safe metadata header first. It confirms `type`/relevance/`timestamp`-freshness before paying for the rest of the file; skip the body read entirely if the metadata alone answers the question.

1. Walk the `index.md` chain down from the relevant document root (`journal/`, `research/`, `roadmap/`, or `spec/`) toward the likely date/project/EPIC — each `index.md` narrows which subdirectory to enter next. For source code, use the repository's package/module path and any local index.
2. Once in a candidate directory, `grep`/read just the metadata header (`type`/`title`/`description`/`tags`) of each candidate before opening its body.
3. Open the full body only for files whose metadata makes them relevant.

This is the whole point of carrying metadata: a read should resolve from the header alone whenever it can, and only pay for a full read on an actual candidate.

## Source-code header form

Use the same fields as the document block below, wrapped in the file's comment syntax. Strip the comment markers before interpreting the YAML. Keep a required shebang, encoding declaration, or doctype before the header; otherwise put the header immediately after it.

```bash
#!/usr/bin/env bash
# ---
# type: Shell Script
# title: Archive today's context
# description: Moves today's working context into its dated wiki location.
# tags: [wiki, archive]
# timestamp: 2026-08-20T00:00:00+09:00
# ---
```

Use `/* ... */`, `<!-- ... -->`, or the language's equivalent for other source files. Omit `resource` when there is no genuine canonical URI. Do not put bare YAML before a line that the runtime requires to be first.

## Fixing existing documents and source code (progressive, not a migration sweep)

No bulk migration — old `~/wiki`/`spec/` documents and existing source files keep whatever shape they were written in until something touches them. When a skill opens an existing `~/wiki/**/*.md`, `spec/**/*.md`, or first-party source-code file for reading or editing and finds:

- **No frontmatter/header** — add the six-field block in the file's native syntax before writing it back, inferring `type`/`title`/`description`/`tags`/`timestamp` from the content (`resource` only if one is genuinely named).
- **A path that no longer matches this repo's current convention** for a document (e.g. a roadmap STORY still filed as `{epic-slug}/{story-slug}/index.md` instead of `{epic-slug}/{story-slug}.md`, or a spec STORY filed outside `spec/{epic-slug}/{story-slug}.md`) — `mv` it to the current path and fix every `index.md` link that pointed at the old one, same as any other Fix in `@skills/roadmap`, or the "keep the spec in sync" step in `@skills/do-plan`.
- **A generated, dependency, or vendored source file** — do not add a header; leave it under its owning tool's format.

Do this as part of whatever touched the file, not as a separate pass. A directory nobody opens keeps its old shape indefinitely, and that's fine.

Skipped: OKF's provenance/trust/lifecycle families (`sources`, `verified`, `status`, `stale_after`) and Attested Computation concepts — built for a data catalog verifying BigQuery-grade claims, not a personal wiki. Add a field when a real need for it shows up (e.g. `status: deprecated` on a stale roadmap EPIC), not preemptively.
