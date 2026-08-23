# Chapter 14: Successive Refinement

Core agent lesson: good code is often produced by making a rough version work, then refining in small verified steps.

Apply this as:

- Start with a simple passing implementation.
- Stop when code starts resisting change and clean it immediately.
- Refine incrementally, not by giant rewrite.
- Keep tests green through each refinement.
- Add argument or input variants one at a time with tests.

Agent questions:

- Am I attempting the final architecture before the behavior is proven?
- Can this refactor be split into smaller green steps?
- Did I preserve the tests that let refinement continue safely?
