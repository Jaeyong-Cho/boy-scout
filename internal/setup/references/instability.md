# Instability Violations

## Why this is a problem

An instability violation means a package is importing something less stable than itself — the least-stable thing it leans on. Stability is the inverse of changeability: a stable package has few reasons to change (other code depends on it, so changes are risky and rare); an unstable package changes often (it has few dependents, so changes are safe). When a stable package imports an unstable one, it gets dragged into that instability. A small change deep in the unstable package can force the stable package — and everything that depends on it — to change too. This breaks the principle that stable packages should protect their dependents from change.

**Related concepts:** `meta-pattern.md` explains that stable packages are those you've chosen to rely on — you pay the cost of their changes; unstable packages are still evolving. `deep-modules.md` shows how a stable, small interface lets implementation changes hide inside without forcing dependents to change.

## How to fix it

Point the dependency the other way. If `domain` (stable, depended on by everything) is importing `httpapi` (unstable, changes to add new routes), invert it: move the glue that lives in `domain` but calls `httpapi` into `httpapi` instead. `domain` no longer imports `httpapi`, only `httpapi` imports `domain`. Now when `httpapi` changes, it doesn't force `domain` to change. If a simple import inversion is impossible (it would create a cycle), you need to extract a new package that both can depend on — one small, stable, depended-on-by-both package that holds just the boundary interface, not the implementation.

