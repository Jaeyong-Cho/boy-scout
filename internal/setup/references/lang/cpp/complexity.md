# Complexity Violations — C++ Example

## Problem

A function with many nested `if`/`for`/`switch` branches has too many independent paths for a reader (or a test suite) to reliably hold in mind at once.

## Solution

Extract each branch into its own well-named helper function, so the original function reads as a short sequence of named decisions: `validateInput()`, `computeDiscount()`, `applyDiscount()`. Each helper now carries only its own branch, so both are easier to read and to test independently.
