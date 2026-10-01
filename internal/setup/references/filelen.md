# Filelen Violations

## Why this is a problem

A file that's too large is mixing multiple concerns. It holds more than one job — maybe it defines the data model, the business logic, and the API transport layer all in one file. This makes the file hard to understand (readers must hold all concerns in mind at once), hard to test (one concern's test can't avoid importing the others), and hard to reuse (can't take one concern without the whole file).


## How to fix it

Split the file along natural seams, where each concern naturally separates. Keep each new file focused on one responsibility and keep dependencies narrow. After splitting, re-run the relevant filelen check and your test suite to confirm the original violation is fixed without changing behavior.

## Examples

For a concrete before/after code example in your language:

- **Go:** See `references/lang/go/filelen.md`
- **C++:** See `references/lang/cpp/filelen.md`
- **TypeScript:** See `references/lang/ts/filelen.md`
