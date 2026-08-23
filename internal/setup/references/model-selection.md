# Model Selection

Which model tier an agent should run at is a property of the job's shape, not the agent's identity. Judge it on three axes:

| Axis | Question |
|---|---|
| **Ambiguity** | Is the task well-specified, or does it require judgment about what "right" even means? |
| **Mistake cost** | If the output is wrong, is that cheap to notice and retry, or does it fail silently? |
| **Verifiability** | Is there a fast automatic check (tests, compiler, linter) that catches errors, or does correctness rest on judgment alone? |

**Rule of thumb:**

- High ambiguity + no automatic check → **Powerful** (`opus`). Getting it wrong is expensive and nothing else will catch it.
- Well-specified task with a fast feedback loop → **Medium** (`sonnet`). The spec plus tests/compiler bound the risk of a wrong answer.
- Mechanical, low-ambiguity, cheaply re-generated → **Mini** (`haiku`). Errors are cheap to spot and cheap to redo.

## Job categories

| Job | Tier | Model | Why |
|---|---|---|---|
| Requirement / ambiguity analysis, architecture decisions | Powerful | `opus` | High ambiguity; no test verifies "did I understand the right problem" |
| Code / security review | Powerful | `opus` | A missed bug is a silent failure; judgment quality is the whole job |
| Coding / implementation | Medium | `sonnet` | Spec is defined; compiler and tests provide a fast feedback loop |
| Refactoring (behavior-preserving) | Medium | `sonnet` | Tests must stay green; scoped, mechanical judgment |
| Debugging / root-cause investigation | Medium | `sonnet` | Reasoning is needed but scoped to a specific failing test or error |
| Test-case writing (given an existing spec/implementation) | Mini | `haiku` | Mechanical enumeration of cases from a known, already-decided spec |
| Documentation generation from existing code | Mini | `haiku` | Source of truth already exists; low ambiguity |
| Commit messages, PR descriptions, changelog/release notes drafting | Mini | `haiku` | Facts are fixed by the diff/history; job is accurate summarization, not judgment |
| Release readiness / semver version-bump decisions | Powerful | `opus` | A misjudged breaking-vs-minor call breaks downstream consumers silently; no automated check catches it |
| Formatting, boilerplate, mechanical transforms | Mini | `haiku` | Deterministic transformation, trivially checked |
| Fact-finding / narrow lookup (locate a definition, check a config value, list callers, confirm a version) | Mini | `haiku` | Well-specified locate task; a wrong or incomplete answer is cheap to notice and cheap to re-run |
| Open-ended exploration / architecture reconnaissance (assess a pattern, survey trade-offs, judge risk before a design decision) | Medium | `sonnet` | High ambiguity in what's relevant, but exploration only gathers evidence — it doesn't make the risky call itself, so `sonnet` covers the reasoning without escalating to `opus` for a research step |
| Anything not listed above | Medium | `sonnet` | Default to the middle tier until the axes above say otherwise |

## Escalation

If an agent at a given tier gets stuck — repeated failed attempts, or the job turns out to need judgment beyond what its scoped ambiguity band assumed — escalate to the next tier up. Don't keep retrying at the same tier expecting a different result; a `haiku` job that can't converge is evidence it wasn't actually mechanical, and a `sonnet` job that can't converge is evidence the ambiguity was underestimated.

`fable` is a separate, style-tuned model and isn't part of this power ladder — don't substitute it into a tier slot above.
