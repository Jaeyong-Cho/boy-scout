# TODO

## Now

## Next

## Later

- Type-3 near-miss clone detection (LCS similarity threshold) — tracked as its own plan, `02-duplication-detector-type3.md`
- Block/statement-level sub-function clone detection (v2 granularity, below whole-function)
- `--include-tests` flag to opt `_test.go` files back into duplication scanning
- C++/TS duplication support (mirrors the existing per-language rollout order)
- Hashing/bucketing to cut the O(n²) pair-scan cost on large repos
- Cross-repo/cross-service duplication is out of scope — this checker and its docs only ever scan one repo's file tree, same as every existing check
- Auto-choosing the destination package name/location for an extracted cross-package shared helper is left to the agent/human at fix time, not hardcoded into the generic reference doc — codebases differ too much (`internal/common` isn't universal) to bake in one answer
- Type-4 (functionally-equivalent, syntactically-different) clone detection — needs a program-dependence-graph/behavior-equivalence engine, not a text-normalize-and-compare extension
