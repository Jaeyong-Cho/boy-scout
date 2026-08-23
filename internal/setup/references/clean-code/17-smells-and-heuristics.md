# Chapter 17: Smells And Heuristics

Core agent lesson: smells are review prompts, not automatic rewrite permission.

Use these groups as a review scan. The IDs follow the standard clean-code heuristic numbering so findings can cite a stable code (for example "G17 misplaced responsibility" or "N7 name hides side effects").

### Comment Smells (C)

- C1: comment carries background that belongs elsewhere (tickets, history, metadata)
- C2: obsolete comment that no longer matches the code
- C3: redundant comment that restates the code
- C4: sloppy or unclear comment
- C5: commented-out code

### Environment Smells (E)

- E1: build requires more than one step
- E2: tests require more than one step

### Function Smells (F)

- F1: too many arguments
- F2: output arguments that mutate parameters
- F3: flag arguments selecting behaviors
- F4: dead, never-called functions

### General Smells (G)

- G1: mixed languages or paradigms in one file without need
- G2: obvious expected behavior left unimplemented
- G3: incorrect behavior at boundaries and edge cases
- G4: disabled or overridden safeguards (ignored warnings, skipped tests, silenced linters)
- G5: duplication of knowledge
- G6: code at the wrong abstraction level
- G7: dependency direction problems (foundations depending on details)
- G8: too much exposed information; wide interfaces
- G9: dead code
- G10: poor vertical separation; related code far apart
- G11: inconsistency; same idea done different ways
- G12: clutter that earns no keep
- G13: artificial coupling between things that do not belong together
- G14: feature envy; code operating on another module's internals
- G15: selector arguments that switch behavior
- G16: obscured intent
- G17: misplaced responsibility; code living where it does not belong
- G18: inappropriate static/global behavior
- G19: missing explanatory variables
- G20: function names that do not say what the function does
- G21: algorithm not understood before changing it
- G22: logical dependency not represented physically
- G23: repeated conditionals that want a single dispatch structure (polymorphism, handler map)
- G24: ignored standard conventions
- G25: magic values without domain names
- G26: imprecision in assumptions, types, or comparisons (money in floats, naive time math)
- G27: relying on convention where explicit structure is needed
- G28: unencapsulated complex conditionals
- G29: negative conditionals where positive ones read clearer
- G30: functions doing more than one thing
- G31: hidden temporal coupling
- G32: arbitrary, unjustified structural choices
- G33: boundary conditions not encapsulated in one place
- G34: functions descending more than one abstraction level
- G35: configurable data buried at low levels instead of the top
- G36: transitive navigation through object graphs (train wrecks)

### Language-Specific Smells (J and equivalents)

- imports, constants, and enum-like concepts handled against local language idioms
- rules from one language translated into another blindly instead of idiomatically

### Naming Smells (N)

- N1: non-descriptive names
- N2: names at the wrong abstraction level
- N3: missing standard nomenclature the team or ecosystem already uses
- N4: ambiguous names
- N5: short names for long scopes, long names for short scopes inverted
- N6: unnecessary encodings and prefixes
- N7: names that hide side effects

### Test Smells (T)

- T1: insufficient tests; untested reachable behavior
- T2: no coverage signal where coverage would reveal gaps
- T3: skipped trivial tests that would document behavior
- T4: ignored tests that encode unresolved ambiguity
- T5: missing boundary tests
- T6: no extra coverage near recent bugs
- T7: failure patterns not investigated
- T8: coverage patterns not inspected
- T9: slow tests that discourage frequent runs

Agent questions:

- Is this smell in the scope of the requested task?
- Does it create immediate risk?
- Did my change introduce it?
- Can I fix it safely with current tests?
