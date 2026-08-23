# Appendix A: Concurrency II

Core agent lesson: concurrency correctness depends on execution paths, library guarantees, lock strategy, throughput tradeoffs, deadlock prevention, and testing tools.

Cover these concerns:

- client/server threading tradeoffs
- number of possible execution paths
- library support such as executors, nonblocking solutions, and thread-safe collections
- non-thread-safe classes
- dependencies between methods that break under parallel calls
- client-side vs server-side locking tradeoffs
- throughput calculations and bottlenecks
- deadlock conditions: mutual exclusion, hold-and-wait, no preemption, circular wait
- strategies for breaking deadlock conditions
- tools and instrumentation for forcing timing failures

Agent questions:

- Can two valid calls interleave into invalid state?
- Which component owns locking?
- What throughput gain justifies the added correctness risk?
- How can tests force the rare interleaving?
