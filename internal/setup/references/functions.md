# Chapter 3: Functions

Core agent lesson: functions should be small, focused, and readable top to bottom.

Cover these concerns:

- small blocks and clear indentation (Not conflit with `deep-modules.md`. The interface of others should narrow, not the private functions)
- one thing per function
- one abstraction level per function
- top-to-bottom stepdown flow
- careful handling of switch or selector logic
- descriptive function names
- few arguments, with related values grouped into real concepts
- avoiding flag arguments and output arguments
- avoiding hidden side effects
- separating commands from queries
- using idiomatic exceptions or error flows instead of ignorable status codes
- extracting error-handling blocks when they obscure the main path
- keeping error handling as one responsibility
- using structured control flow without clever jumps

Agent questions:

- Can I summarize this function without using "and"?
- Does each line sit at the same abstraction level?
- Does a boolean argument mean this is really two functions?
- Is the function secretly changing input, global state, or external systems?
