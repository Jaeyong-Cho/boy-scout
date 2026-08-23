# Duplication Violations

## Why this is a problem

Code duplication multiplies the cost of every future fix. When the same logic appears in multiple places, a bug fix, security patch, or behavior change must be applied everywhere it's copied. This creates three risks: forgetting a copy (inconsistent behavior), applying the fix differently in each place (subtle bugs), or spending hours hunting down every location when one needs updating. Copy-pasted code also makes the codebase harder to understand — readers can't tell if two similar functions intentionally diverge or if one copy drifted accidentally.

**Related concepts:**
- `functions.md` — The clean-code chapter on functions. Covers extraction, naming, and the principle that a piece of logic should live in one place.
- `meta-pattern.md` — Explains when code should stay together vs. split, and why extraction is always safer than deletion when consolidating duplicates (a shared helper can evolve independently; a deleted copy can't be recovered if the copies turn out to serve different purposes).

## How to fix it

Read the duplication cluster using `--format=json` to see all the members, the line ranges they span, and whether the cluster crosses package boundaries. Then extract one shared helper function that both members reduce to a single-line call:

- **Same-package clusters:** Extract the shared helper into the same package, usually as a private (unexported) function. All members call the shared helper from their original locations.
- **Cross-package clusters:** Create a new shared package (or extend an existing `internal/common`-like package) with an exported helper, and have both packages import it. This is more disruptive than same-package extraction, but necessary when the code lives at different levels or in modules that shouldn't depend on each other.

Always extract a helper rather than delete one copy. A coincidental match today (same algorithm for two unrelated business rules) can become two independently-evolving functions tomorrow — extraction is reversible; deletion isn't.

After extracting, run your test suite once to confirm the shared helper's behavior matches all the original copies. The duplication cluster is fixed when boy-scout finds zero violations in its members and you've removed the duplicate bodies.

## Examples

For a concrete before/after code example in your language:

- **Go:** See `references/lang/go/duplication.md`
- **C++:** not yet supported. Boy-scout's duplication checker runs on Go codebases only.
