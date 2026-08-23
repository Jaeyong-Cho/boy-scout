# Preference Format

Where a recorded preference lives, and how each entry is written. Used by `to-preference`.

## Style

Preference files are concise decision ledgers, not report documents.

- Apply `document-style.md` to user-facing preference confirmations: lead with the decision and use concise bullets for a list of candidates.
- Apply `document-style.md`'s bullet rules to persisted entries: one decision per bullet, decision first, direct wording, and no joined unrelated claims.
- Do not add an introduction, conclusion, table, or diagram to a preference topic file. A preference file is list-like content, so concise bullets are the correct structured form.

## Standing vs one-off

A **standing** preference is one you'd answer the same way next time, regardless of which feature or project prompted it — an API design convention, a testing habit, a naming rule. A **one-off** is specific to this task's own constraints (this feature uses Redis because this project already runs Redis). Only standing preferences get recorded; one-offs stay in the session.

## File location

- `../../preferences/{topic}.md` — true across any project. `{topic}` is a kebab-case slug of the rule's subject (e.g. `../../preferences/api-design.md`).
- `.context/preferences/{topic}.md` — true only in the current project (e.g. `.context/preferences/tech-stack.md`).

## Entry format

Each entry is the rule stated as a decision, one line: `- Use plural nouns for REST resource endpoints.` Append under the file's existing bullets. If it's the first entry on that topic, create the file with just that line. Split distinct decisions into separate bullets rather than joining them with “and.”
