# Chapter 4: Comments

Core agent lesson: comments are useful when they explain intent, constraints, warnings, or public contracts; they are harmful when they compensate for unclear code.

Keep comments for:

- legal requirements when needed
- useful context that code cannot express directly
- intent behind a non-obvious decision
- clarification of unavoidable ambiguity
- warnings about consequences
- TODOs with actionable context
- amplification of a subtle point
- public API documentation where the ecosystem expects it

Avoid comments that are:

- redundant with code
- misleading or stale
- mandated noise
- journal logs
- decorative markers
- closing-brace labels caused by oversized functions
- attributions that belong in version control
- commented-out code
- too much nonlocal background
- function headers for obvious private helpers

Agent questions:

- Can clearer names or extraction remove this comment?
- Does the comment explain why, or merely restate what?
- Could this comment become false when nearby code changes?
