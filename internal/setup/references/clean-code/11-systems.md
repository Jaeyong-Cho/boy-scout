# Chapter 11: Systems

Core agent lesson: construction, wiring, runtime policy, and domain behavior should not be tangled.

Cover these concerns:

- separate system construction from system use
- keep main/wiring code distinct from domain logic
- use factories and dependency injection only when they clarify construction
- scale architecture incrementally rather than upfront
- isolate cross-cutting concerns
- test-drive architecture decisions where possible
- make decisions at the last responsible moment
- use standards only when they add demonstrable value
- build domain-specific language where it makes repeated intent clearer

Agent questions:

- Is wiring mixed into business behavior?
- Is the abstraction justified by real construction complexity?
- Does this framework feature buy clarity or only ceremony?
