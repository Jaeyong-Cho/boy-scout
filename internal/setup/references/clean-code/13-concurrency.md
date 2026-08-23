# Chapter 13: Concurrency

Core agent lesson: concurrency creates correctness risks that require explicit ownership, data scope, lifecycle, and testing.

Cover these concerns:

- myths that concurrency is simple or only a performance concern
- separating concurrency policy from business logic
- limiting shared data scope
- using copies or immutable data when useful
- keeping threads/tasks independent when possible
- knowing library concurrency primitives
- understanding execution models such as producer-consumer and readers-writers
- avoiding dependencies between synchronized methods
- keeping critical sections small
- designing shutdown carefully
- treating sporadic failures as possible concurrency bugs
- testing with stress, instrumentation, different platforms, and varied schedules

Agent questions:

- Who owns this state?
- What can run at the same time?
- How does cancellation or shutdown complete safely?
- Can retries or duplicate events corrupt data?
