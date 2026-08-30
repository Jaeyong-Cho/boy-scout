# Linelen Violations

## Why this is a problem

A line that's too long is hard to read without horizontal scrolling. Very long lines usually indicate deeply nested logic or an unbroken chain of method calls — the same "hard to hold in your head" problem as having a function or file that's too large. The reader must mentally parse the entire expression to understand what it does, and small changes to the middle of the line require retyping the whole thing.

## How to fix it

Extract the sub-expression into a named local variable (one intermediate step per variable) or break the call chain across multiple lines. The goal is to make each line short enough to read without scrolling: typically 80–120 characters depending on team norms. After splitting, `boy-scout go all` should report the violation fixed.

## Examples

For a concrete before/after code example in your language, see the corresponding language guide — each language's approach to breaking long lines differs (Python uses backslash continuation, JavaScript uses implicit semicolon insertion, Go and C++ prefer breaking before operators).
