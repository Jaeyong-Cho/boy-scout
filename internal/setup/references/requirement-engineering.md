# Requirements Engineering

**Requirements Engineering** is the process of turning "what stakeholders want" into requirements that are clear, testable, and traceable to a verification method — deciding *what* to build, not *how*.

## Why it matters

Most software projects fail not from a lack of technical skill, but because needs were misunderstood, requirements stayed ambiguous, or nobody checked the built thing against what was actually asked for. The activity that catches all three is Validation — an ambiguous requirement survives elicitation and analysis just fine; it only breaks when someone tries to write a testable acceptance criterion for it and can't.

## Key activities

1. **Elicitation** — customer interviews, surveys, workshops, user observation, brainstorming.
2. **Analysis** — identifying conflicting requirements, prioritization, feasibility.
3. **Specification** — writing requirements into a document (SRS), clear enough that developers and customers share the same understanding.
4. **Validation** — confirming requirements match intent by converting each one into a testable acceptance criterion. The highest-leverage activity — detailed below.
5. **Management** — tracking requirements as they change, maintaining traceability.

## Functional vs. Non-functional Requirements

| Category | Description | Example |
|---|---|---|
| Functional Requirement | What the system must do | Sign-up, login, payment |
| Non-functional Requirement | How the system must behave | Performance, security, availability, usability |

## Validation: writing Acceptance Criteria

A requirement isn't done until it's an acceptance criterion someone can check without a judgment call. Each row is SMART and phrased Given–When–Then.

**SMART:**

- **Specific** — one condition per row; "and" is a sign to split it into two.
- **Measurable** — a number, a state, or a boolean, checkable without interpretation.
- **Achievable** — realistic under the constraints already fixed in Decision and Out of Scope.
- **Relevant** — traces back to a line in Requirements or the User Scenario; a criterion that doesn't trace to either belongs in Out of Scope, not here.
- **Time-bound** — bounded to when it's checked ("within 2 seconds", "on the first request") — never "eventually".

**Given–When–Then:** Given {starting state} is the precondition, not the action. When {action} is one trigger. Then {expected result} is observable from outside, not internal state.

**Classify every row:**

- **Normal** — expected input, happy path.
- **Exception** — invalid input, failure, or unavailability — this is where the failure modes surfaced during elicitation belong.
- **Boundary** — the edges: zero, max, empty, first-of-its-kind, exactly-at-the-limit.

**Verification Method:** pick the cheapest one that can't lie. Prefer something automatable (unit test, e2e test, query) over manual; reserve manual for checks that genuinely need human judgment (visual review, UX feel).

## Example

Customer says: "I'd like login to be convenient." Not a requirement yet.

**Functional**
- The user must be able to log in with email and password.
- The user must be able to log in with a Google account.

**Non-functional**
- Login response time must be 2 seconds or less.
- Login credentials must be encrypted via HTTPS.
- Login success rate must be 99.9% or higher.

**Acceptance Criteria**

|AC|Category|Verification Method|
|--|--|--|
|Given a registered user on the login page - When they submit a correct email and password - Then they're authenticated and redirected within 2 seconds|Normal|e2e test: `login.spec.ts`|
|Given a registered user - When they submit the wrong password 3 times - Then the account locks and shows a clear error|Exception|unit test: `AuthService.lockout`|
|Given a new user with no existing account - When they sign in with Google for the first time - Then an account is created and linked without a password|Boundary|manual test: first-time OAuth flow|
