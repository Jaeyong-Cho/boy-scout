# Filelen Violations

## Why this is a problem

A file that's too large is mixing multiple concerns. It holds more than one job — maybe it defines the data model, the business logic, and the API transport layer all in one file. This makes the file hard to understand (readers must hold all concerns in mind at once), hard to test (one concern's test can't avoid importing the others), and hard to reuse (can't take one concern without the whole file).

**Related concepts:**
- `classes.md` — The clean-code chapter on modules. Covers Small Responsibility Principle, cohesion (do methods use the same state?), and when to split a class/file.
- `meta-pattern.md` — explains the forces that decide when to split code. A file that mixes concerns has conflicting decouplers — the urge to separate business logic from data persistence from API concerns — but they're all forced into one place. This creates pain that gets more expensive to ignore than the cost of splitting.

## How to fix it

Split the file along natural seams, where each concern naturally separates. high cohesion: each new file has one clear job, and its functions work closely together toward that job. loose coupling: each file knows the minimum it needs to know about the others — import from the other file only the small public interface, not internal details. After splitting, `boy-scout go all` should report the original file fixed. If the split creates new violations (e.g., a new instability violation because the seam you chose created a dependency cycle), fix those in later steps.

