# Chapter 7: Error Handling

Core agent lesson: error handling must preserve clarity in the happy path and context in the failure path.

Cover these concerns:

- prefer idiomatic exceptions/results over ignorable codes
- design the failure path first: sketch the error skeleton for a risky operation before filling in the happy path
- design try/catch or error branches around the caller's needs
- preserve context and original cause
- distinguish normal alternative flows from true failures; model expected alternate outcomes (not found, empty, declined) as values, result types, or special-case objects so callers do not need exceptional control flow for ordinary cases
- avoid returning or passing null-like values when the language has safer options
- keep error handling localized and cohesive

Agent questions:

- Can the caller make a useful decision from this error?
- Did I hide the original cause?
- Did I replace a real failure with a silent default?
- Is nullability or absence explicit in the type or contract?
