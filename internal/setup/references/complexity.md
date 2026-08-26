# Complexity Violations

## Why this is a problem

A function with high cyclomatic complexity has too many independent paths through it — every `if`, loop, case, and logical `&&`/`||` branches the execution one more way. Past a handful of branches, no reader can hold all the paths in their head at once, and no test suite can realistically cover every combination. The function becomes hard to change safely: a fix for one path can silently break another one nobody thought to check.

**Related concepts:**
- `functions.md` — The clean-code chapter on functions. Deeply nested branches are usually a sign a function is doing more than one thing; extracting a branch into its own function is often also the fix for complexity.
- `deep-modules.md` — describes good interfaces that hide complexity behind a simple contract. A function with many branches often means the branching logic belongs one level down, behind a smaller interface.

## How to fix it

Extract each branch (or a coherent group of branches, like a whole nested `if`) into its own well-named helper function. This doesn't remove the underlying logic, but it moves each independent path into its own scope, so the caller reads as a short sequence of named decisions instead of one large decision tree. After extracting, `boy-scout go complexity` should report the original function fixed and (usually) not create new violations in the newly extracted helpers, because each one now holds just one part of the decision.

## Examples

For a concrete before/after code example in your language:

- **Go:** See `references/lang/go/complexity.md`
- **C++:** Not yet supported for C++
