# CRAP Violations

## Why this is a problem

CRAP (Change Risk Analysis and Prediction) is high when a function combines high complexity plus low test coverage. A complex function without tests is risky: you can't prove a change is safe. Nobody knows what all its branches do, which ones are dead code, or whether your edit breaks something in an untested path.

**Related concepts:**
- `unit-tests.md` — The clean-code chapter on tests. Tests are production assets that enable change safely; the FIRST principles (fast, independent, repeatable, self-validating, timely) keep tests trustworthy.
- `deep-modules.md` — describes good interfaces that are testable. Complex functions are often a sign that the interface is leaky — callers have to do too much setup, or the function has too many parameters. Refactoring to a simpler interface often reduces complexity and makes testing natural.

## How to fix it

First, if coverage is 0%, add one characterization test — a test that captures what the function does right now, not what it should do. The test doesn't have to be pretty; it just has to pass, so you have a safety net for the refactor. Run it once to confirm it passes, then refactor the function to reduce complexity (simpler logic, fewer branches, more locals extracted to helpers). After refactoring, re-run the characterization test to make sure it still passes, then run the full test suite. If coverage is already above 0%, you can skip the characterization test and go straight to refactoring — existing tests plus the full re-run are enough of a guard.

