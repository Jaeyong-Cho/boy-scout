# TODO

## Now

## Next

## Later

- Type-3 near-miss clone detection (LCS similarity threshold) — tracked as its own plan, `02-duplication-detector-type3.md`
- Block/statement-level sub-function clone detection (v2 granularity, below whole-function)
- `--include-tests` flag to opt `_test.go` files back into duplication scanning
- C++/TS duplication support (mirrors the existing per-language rollout order)
- Hashing/bucketing to cut the O(n²) pair-scan cost on large repos
- Type-4 (functionally-equivalent, syntactically-different) clone detection — needs a program-dependence-graph/behavior-equivalence engine, not a text-normalize-and-compare extension
