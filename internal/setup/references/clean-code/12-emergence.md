# Chapter 12: Emergence

Core agent lesson: clean design emerges through four simple rules applied in strict priority order.

Cover these concerns, in order of importance:

1. runs all tests — correctness outranks every aesthetic concern
2. contains no duplication — one authoritative home per piece of knowledge
3. expresses the intent of the programmer — a reader can tell what and why
4. minimizes classes, methods, and moving parts — subject to the first three

Use the order as a tie-breaker: never remove duplication in a way that breaks tests, and never add elements that rules 1-3 do not require.

Agent questions:

- Are tests strong enough to allow cleanup?
- What duplication represents shared knowledge rather than accidental similarity?
- Can I remove a construct without losing clarity or behavior?
