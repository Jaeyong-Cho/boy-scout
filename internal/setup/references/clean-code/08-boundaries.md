# Chapter 8: Boundaries

Core agent lesson: external boundaries should be wrapped, learned, and tested so third-party change does not leak everywhere.

Cover these concerns:

- isolating third-party APIs behind local adapters
- learning tests for unfamiliar libraries
- contract tests for boundary behavior
- interfaces for code that does not exist yet
- keeping framework/vendor types out of core domains when possible
- validating serialization, time, encoding, and units at the edge

Agent questions:

- How many files know this third-party API shape?
- Is there a narrow local interface around the boundary?
- What happens when the vendor changes, times out, or returns malformed data?
