# Cohesion Violations

## Why this is a problem

A class or struct with low cohesion has methods that touch disjoint sets of fields — they're not really related. Low cohesion is a symptom that the class is gluing together multiple unrelated responsibilities. One method might use fields A and B, another might use fields C and D, with no overlap. This means:

- The class is hard to understand (readers must hold separate mental models in their head).
- Changes to one concern (updating fields for responsibility 1) can accidentally break the other concern (responsibility 2).
- Tests for responsibility 1 can't stand alone; they're coupled to responsibility 2's code.

Boy-scout measures this using **LCOM4** (Lack of Cohesion of Methods), **TCC** (Tight Class Cohesion), or **LCC** (Loose Class Cohesion) — metrics that check how much overlap the methods share in their field accesses. A score near 1.0 (for TCC/LCC) or high LCOM4 means methods are truly independent.

**Note:** Boy-scout skips classes or structs with fewer than 2 methods; there's nothing to compare cohesion against with a single method.

## How to fix it

Split the class along the groups of methods that share the same fields. If methods A and B work together (both use fields 1, 2, and 3) and methods C and D work together (both use fields 4 and 5), create two classes: one holding methods A, B and fields 1–3; the other holding C, D and fields 4–5. The original class' public interface might delegate to both, or one might be used alone. The goal is high cohesion: each new class has one clear job and all its methods work closely together toward that job.

After splitting, `boy-scout go all` should report the original class fixed.

## Examples

For a concrete before/after code example in your language:

- **Go:** See `references/lang/go/cohesion.md`
- **C++:** See `references/lang/cpp/cohesion.md`
- **TypeScript:** See `references/lang/ts/cohesion.md`
