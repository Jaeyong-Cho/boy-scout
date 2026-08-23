# Mocking Guidelines

## Mock at system boundaries only

Mock things external to system — not internal.

| Mock | Don't mock |
|---|---|
| HTTP clients, external APIs | Internal services in same process |
| Database I/O in unit tests | Domain logic collaborators |
| Email/SMS senders | Entity methods |
| Clock / randomness | Method-layer workflows |

## Why

Mocking internal collaborators ties tests to implementation structure. When you restructure internals, tests break even though behavior is unchanged. Defeats purpose of tests.

## When integration tests hit a real boundary

Prefer test double at outermost boundary (e.g. in-memory database adapter, fake HTTP server) rather than mocking individual internal calls.

```typescript
// Good — mock at the boundary
const fakeEmailer: Emailer = { send: jest.fn() }
await registerUser(email, password, fakeEmailer)
expect(fakeEmailer.send).toHaveBeenCalledWith(email, "welcome")

// Bad — mock an internal collaborator
jest.mock("./UserRepository")
jest.mock("./PasswordHasher")
```

## Rule

If removing mock would require network call or disk write, mock it.
If removing mock would just run more of your own code, don't mock it.
