# Duplication Violations — Go Example

## Problem

The same algorithm or logic appears in multiple places. If the algorithm needs to change (for correctness, edge cases, rounding), all copies must be updated consistently.

## Solution

Extract the shared logic into a single helper function. If the duplication is within a package, make it private. If it spans packages, extract to a shared package, export it, and import it in both places. This way, changes to the algorithm happen once and automatically apply everywhere it's used.
