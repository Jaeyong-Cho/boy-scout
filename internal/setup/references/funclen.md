# Funclen Violations

## Why this is a problem

A function that's too large is doing more than one thing at one level of abstraction. It's mixing distinct concerns — setup, computation, cleanup, error handling, logging — that belong in separate steps. This forces readers to hold too much state in their head at once and makes it hard to test, reuse, or refactor individual steps without affecting others.

## How to fix it

Extract each logical step into its own well-named helper function. The original function should read like a table of contents: call `validateInput()`, then `computeResult()`, then `recordMetrics()`, with each helper carrying one level of responsibility. Name each helper so its purpose is obvious without reading its body. Don't just trim lines by removing comments or inlining constants; actually separate the concerns. After extracting, `boy-scout go all` should report the original function fixed and (usually) not create new violations in the newly extracted helpers, because each one now holds just one thing.
