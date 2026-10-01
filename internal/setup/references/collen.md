# Collen Violations

## Why this is a problem

A line that's too long is hard to read without horizontal scrolling. Very long lines usually indicate deeply nested logic or an unbroken chain of method calls — the same "hard to hold in your head" problem as having a function or file that's too large. The reader must mentally parse the entire expression to understand what it does, and small changes to the middle of the line require retyping the whole thing.

## How to fix it

Extract the sub-expression into a named local variable (one intermediate step per variable) or break the call chain across multiple lines. The goal is to make each line short enough to read without scrolling: typically 80–120 characters depending on team norms. After splitting, `boy-scout go all` should report the violation fixed.

## Examples

The same rule applies to Go, C++, TypeScript, HTML, and CSS files selected by the language command. The checker counts Unicode characters, not terminal display columns; a tab counts as one character. The default limit is 100 (`--max-chars`). Overflow is exempt when the line fits after quoted strings are removed. Split expressions at syntactically valid boundaries, then run the checker and your test suite.
