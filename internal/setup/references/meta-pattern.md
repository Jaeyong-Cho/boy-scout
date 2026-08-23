# Meta Pattern: When to Split Code

When code grows, you eventually have to choose: keep it together for simplicity, or split it into separate modules for clarity. The right choice depends on what forces are pushing you.

## Cohesion forces (keep it together)

- **Single team, early stage** — Speed wins. Splitting adds coordination overhead.
- **Data consistency** — Shared state is simpler than synchronized copies.
- **Debuggability** — One process is easier to trace than multiple moving parts.

## Decoupling forces (split it up)

- **Scale** — Different parts need different resources or independent scaling.
- **Variability** — Conflicting requirements demand multiple implementations.
- **Multiple teams** — Conway's Law: team boundaries force code boundaries.
- **Location** — Some parts must run on different machines.

## The rule

Only pay for decoupling when a real decoupler is present and active. Cohesion is the default. As your codebase grows, you pay more in confusion and merge conflicts; at that point, splitting becomes cheaper than staying together.

Typical progression: **Monolith → Layers → Services** as pressures accumulate.
