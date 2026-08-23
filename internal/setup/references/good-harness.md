# Good Harness

A harness is the process that turns a natural-language constraint (a job description, a completion criterion, an acceptance criterion) into a local, executable, pass/fail check. Used when validating a custom agent or skill built via `create-agent` or `skill-creator`: does it actually do the job, not just have the right shape.

## Axes

Classify the constraint before picking a harness:

| Axis | Question |
|---|---|
| **Layer** | Structural — checkable from the file alone, without running anything (frontmatter, schema, file shape). Behavioral — only checkable by actually invoking the agent/skill and looking at what it does. |
| **Determinism** | Objective — one correct answer (exact match, regex, exit code, field present). Judgment-based — correctness is a matter of quality or taste (clarity, tone, "did it use good judgment"). |

## Harness by shape

| Layer | Determinism | Harness | Example |
|---|---|---|---|
| Structural | Objective | grep/awk/field check against the schema reference | `grep -c "^name:"` must print 1 |
| Structural | Judgment | Rare — usually means the constraint is actually behavioral. Re-classify before building a check here. | — |
| Behavioral | Objective | A `test-loop.md` scenario: `run` invokes the agent/skill with a fixed prompt, `verify` checks the output exactly or structurally | Agent must always end its output with a line starting `Completion criterion:` |
| Behavioral | Judgment | skill-creator's eval/benchmark tooling — an LLM-judge scored against a rubric, run repeatedly to check for variance | Agent's explanations must be clear and free of unexplained jargon |

## When the constraint won't harness

If nothing in the table produces a check, the problem isn't the harness — it's the constraint. Rewrite it into SMART / Given-When-Then form (same bar as `../template/requirements.md`'s Acceptance Criteria table) before attempting the gate again. A constraint that can't be phrased as a Given-When-Then can't be harnessed.

## Anti-Patterns

- **Can't fail** — the check passes regardless of whether the behavior is correct (e.g. asserting a file exists but never checking its content). If you can't picture the failing case it would catch, it isn't a harness.
- **Wrong layer** — a structural check standing in for a behavioral constraint. "The field is present" is not the same as "the field's value is correct for this job."
- **Unexecutable "harness"** — a checklist for a human to eyeball. If it isn't a command with an exit code, it's not local executable, it's a hope.

## Worked example

Job: "The `commit-message-writer` agent must produce conventional-commit-format messages."

1. **Layer?** Behavioral — you can't tell from the agent's frontmatter whether its output follows the format; you have to run it on a real diff.
2. **Determinism?** Objective — conventional commits are a fixed grammar (`type(scope): subject`), not a matter of taste.
3. **Harness** → behavioral + objective → a `test-loop.md` scenario: `run` invokes the agent on a fixed sample diff, `verify` regex-matches the output against the conventional-commit grammar and fails loud with the actual string if it doesn't match.
