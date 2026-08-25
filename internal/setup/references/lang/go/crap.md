# CRAP Violations — Go Example

## Problem

High complexity combined with low or zero test coverage creates technical debt. The function is hard to understand, test, and refactor safely.

## Solution

First, write a characterization test that pins down current behavior, then refactor to reduce nesting and extract complexity into smaller, well-tested helpers. Each helper should be simple enough to test and reason about independently.
