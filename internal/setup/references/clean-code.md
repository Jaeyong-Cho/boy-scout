# Clean-Code Chapter Map

This is an agent-oriented synthesis of the full clean-code source structure: 17 chapters, the deeper concurrency appendix, the SerialDate appendix, and the cross-reference appendix. It is not a replacement for the source text. Use it as a checklist so no major clean-code area is skipped during writing, refactoring, or review.

Each chapter is its own file under `clean-code/`, loaded only when needed — read this index first, then open just the chapter file the task calls for.

## How To Use This Map

- For a narrow task, open the one chapter file that matches it.
- For a review, open chapters 2-13 plus `clean-code/17-smells-and-heuristics.md`.
- For refactoring, open chapters 3, 5, 10, 12, 14, 16, and 17. For whole-project cleanup, follow `project-refactor.md` and use this map inside each batch.
- For concurrent code, open chapter 13 and `clean-code/appendix-a-concurrency-ii.md`.
- For tests, open chapter 9 and the T-group in `clean-code/17-smells-and-heuristics.md`.
- For "where does this code or file belong" questions, combine chapter 10 (cohesion), G6, G17, G24, and the Where Code Lives section of the skill.

## Chapters

| File | Core agent lesson |
| --- | --- |
| `clean-code/01-clean-code.md` | Clean code is a professional obligation; mess compounds cost. |
| `clean-code/02-meaningful-names.md` | Names are the primary documentation layer. |
| `clean-code/03-functions.md` | Functions should be small, focused, readable top to bottom. |
| `clean-code/04-comments.md` | Comments help explaining intent; harm compensating for unclear code. |
| `clean-code/05-formatting.md` | Formatting communicates structure before the code is understood. |
| `clean-code/06-objects-and-data-structures.md` | Objects hide data behind behavior; data structures expose it. |
| `clean-code/07-error-handling.md` | Preserve clarity in the happy path, context in the failure path. |
| `clean-code/08-boundaries.md` | External boundaries should be wrapped, learned, and tested. |
| `clean-code/09-unit-tests.md` | Tests are production assets that enable change. |
| `clean-code/10-classes.md` | Classes/modules stay small, cohesive, one reason to change. |
| `clean-code/11-systems.md` | Construction, wiring, policy, and domain behavior stay untangled. |
| `clean-code/12-emergence.md` | Clean design emerges from four rules in strict priority. |
| `clean-code/13-concurrency.md` | Concurrency needs explicit ownership, scope, lifecycle, testing. |
| `clean-code/14-successive-refinement.md` | Ship a rough version, then refine in small verified steps. |
| `clean-code/15-junit-internals.md` | Even respected frameworks improve through naming and decomposition. |
| `clean-code/16-refactoring-serialdate.md` | Legacy cleanup: protect behavior first, then names, structure, tests. |
| `clean-code/17-smells-and-heuristics.md` | Smells are review prompts, not automatic rewrite permission. |
| `clean-code/appendix-a-concurrency-ii.md` | Concurrency correctness: paths, library guarantees, locks, deadlocks, tools. |
| `clean-code/appendix-b-serialdate-source.md` | Legacy example for characterization and behavior-preserving cleanup. |
| `clean-code/appendix-c-cross-references.md` | Review is interconnected; cross-check related smells together. |

## Coverage Pressure Scenarios

`clean-code/coverage-pressure-scenarios.md` — self-test table mapping common tasks to the chapters they must consult.
