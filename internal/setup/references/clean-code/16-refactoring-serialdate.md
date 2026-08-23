# Chapter 16: Refactoring SerialDate

Core agent lesson: legacy code cleanup should first protect behavior, then improve names, structure, tests, and responsibility.

Apply this as:

- First make behavior observable with tests or characterization checks.
- Then make names accurate and domain-specific.
- Remove misleading comments and dead code only when covered or clearly unused.
- Move misplaced constants, calculations, or responsibilities to clearer homes.
- Keep refactor steps small enough to review.

Agent questions:

- Do I know current behavior before changing it?
- Is this rename behavior-preserving?
- Am I cleaning legacy code or silently changing its contract?
