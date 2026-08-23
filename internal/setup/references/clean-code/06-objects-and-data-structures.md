# Chapter 6: Objects And Data Structures

Core agent lesson: objects hide data behind behavior; data structures expose data for external behavior. Mixing both casually creates confusion.

Cover these concerns:

- data abstraction instead of leaking representation
- object/data tradeoff: adding new types vs adding new operations
- Law of Demeter and avoiding train-wreck navigation
- avoiding hybrids that expose fields while pretending to protect invariants
- DTOs for plain transport data
- Active Record patterns when the framework uses them, with domain behavior kept clear

Agent questions:

- Is this value just data, or does it protect behavior and invariants?
- Am I reaching through object internals instead of asking for behavior?
- Is a framework model becoming a dumping ground for unrelated logic?
