# Spec Document Convention

A target project's spec documents live under `spec/**/*.md`, not a single `SPEC.md`/`docs/spec.md`/`requirements.md` file.

- One directory per EPIC: `spec/{epic-slug}/`, one file per STORY inside it: `spec/{epic-slug}/{story-slug}.md`.
- `spec/{epic-slug}/index.md` follows `../template/spec-epic-index.md` — a Business value section and a Completion criteria section (the observable state that proves the EPIC is done), each conclusion-first and as long as it warrants per `document-style.md` (never truncated to a fixed sentence count), plus an Overview section and a list linking to its STORY files.
- Each STORY file follows `../template/spec.md`'s format — an OKF frontmatter block per `document-style/frontmatter.md`, a Value to user section (what the user can do, or gets, once this ships) and a Completion criteria section, plus a Spec section, ending in an AC (acceptance criteria) table, one row per criterion, per `requirement-engineering.md`. Completion criteria is the coarse, checkable-now line; AC is the detailed Given/When/Then breakdown, as many rows as the STORY needs, filled in when the STORY is picked up.
- `spec/index.md` follows `../template/spec-index.md` — one heading per top-level area, linking to each `spec/{epic-slug}/index.md`, plus optional Not yet specified / Out of scope sections for EPIC/STORY candidates that aren't sharp yet or were ruled out.
- `spec/overview.md` is the project's what/why in one short page — a paragraph, not a spec — linking into `spec/index.md` for detail rather than restating any requirement or AC.
- If `spec/` doesn't exist yet in the target project, create it and note that in the plan/report.
- Adding or updating a STORY file always updates its `spec/{epic-slug}/index.md` and the top-level `spec/index.md` in the same change — an unlisted file is treated as not done.
- An existing STORY file found missing frontmatter, or filed outside `spec/{epic-slug}/{story-slug}.md`, gets fixed in place per `document-style/frontmatter.md`'s "Fixing existing documents" — as part of whatever touched it, not a dedicated sweep.
