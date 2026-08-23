# Abstractness Violations

## Why this is a problem

A package lives in one of two healthy zones: either it's abstract (mostly interfaces and types, few or no concretions, depended on by many) or it's concrete (lots of implementations, few dependents). The Zone of Pain is being concrete and depended on by many — your concrete details become everyone else's problem. The Zone of Uselessness is being abstract with no dependents — abstraction with no purpose. An abstractness violation means you've wandered into one of these zones. Usually it's the Zone of Pain: you've built a concrete package (lots of implementation, few interfaces) that many other packages depend on, so changes to it ripple everywhere.

**Related concept:** `deep-modules.md` describes the ideal shape — small interface hiding deep implementation. When your package is depended on by many (has many dependents), being a deep module is essential: a small, stable interface lets you change implementation without breaking anyone. The Zone of Pain is exactly what happens when you have a shallow module (large interface, little hidden logic) that many packages depend on.

## How to fix it

Extract the stable boundary (the interfaces and types that your dependents actually use) into a separate small, abstract, deep module. Move the concrete implementations into a different package. Now the dependents import only the abstract boundary, not the concrete details. Changes to the implementations don't touch the boundary, so dependents don't need to change. If your package is in the Zone of Uselessness (abstract but unused), question whether it should exist — it may be premature abstraction. More often, it's in the Pain zone and needs the extraction fix above.

