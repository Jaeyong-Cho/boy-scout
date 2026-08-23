# Sentence-Level Writing Rules

Read this when editing prose sentence by sentence.

## 5. Prefer Simple and Concrete Language

Use simple language unless technical terminology is necessary.

- Prefer concrete expressions over abstract expressions.
- Explain necessary technical terms when they first appear.
- Avoid unnecessary jargon.
- Replace vague expressions with observable facts.
- Do not use sophisticated wording merely to sound professional.

Avoid vague expressions such as: "appropriately", "efficiently", "significantly", "various", "somewhat", "there is a problem", "it seems", "needs improvement". Replace them with concrete information when possible.

Example:

Bad: "Performance is significantly degraded."
Better: "Average latency increased from 20 ms to 85 ms."

## 6. Keep Sentences Short and Focused

Prefer one main meaning per sentence.

Split a sentence when it contains multiple independent ideas such as: problem, cause, solution, exception, consequence.

Avoid unnecessary use of: long subordinate clauses, excessive commas, "of", nominalized expressions, redundant modifiers, repeated explanations.

Keep the subject and predicate reasonably close.

Do not make every sentence artificially short. Preserve necessary logical relationships.

## 7. Separate Facts, Opinions, and Hypotheses

Clearly distinguish:

- Fact: directly verified information
- Observation: what was observed
- Interpretation: explanation derived from observations
- Hypothesis: an unverified explanation
- Opinion: judgment or preference

Never present an assumption as a fact.

For technical documents, prefer evidence such as: source code, configuration, logs, metrics, test results, reproduction steps, screenshots, specifications, references.

If information cannot be verified, explicitly mark it as uncertain.

## 8. Never Invent Technical Facts

When working with a codebase or project:

- Inspect the relevant code before documenting implementation details.
- Inspect configuration before documenting configuration behavior.
- Inspect tests before claiming behavior is tested.
- Verify commands before documenting how to run them.
- Do not invent APIs, parameters, outputs, file paths, or behavior.
- Do not assume that an old README accurately describes the current implementation.

If the evidence is insufficient, say so.
