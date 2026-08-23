# Chapter 10: Classes

Core agent lesson: classes and modules should be small, cohesive, and organized around one reason to change.

Cover these concerns:

- organization of public surface, internals, and helpers
- encapsulation without hiding important design facts
- small classes or modules
- Single Responsibility Principle
- cohesion among fields and methods
- splitting classes when cohesion drops
- organizing for change
- isolating from change through narrow dependencies

Agent questions:

- Why would this class or module change?
- Do its methods use the same state and concepts?
- Is it a real domain role or a vague manager bucket?
- Is it easy to test without unrelated collaborators?
