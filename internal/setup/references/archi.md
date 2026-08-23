# Architecture Layers

| Layer | Question | DDD equivalent |
|-------|----------|----------------|
| **Objects** | What domain things exist? | Entity / Value Object |
| **Logics** | How does the domain compute and decide? | Domain Service |
| **Usecase** | What user goals are worth automating? | Application Service |
| **External** | How does the outside world interact? | Controller / CLI / UI / Gateway |

Each class belongs to exactly one layer. If a class is hard to place, the boundary is probably wrong.

## Dependency Rule

**Inner layers never depend on outer layers.**

```
Objects  ←  Logics  ←  Usecase  ←  External
(inner)                              (outer)
```

| Layer | May depend on |
|-------|--------------|
| Objects | nothing |
| Logics | Objects |
| Usecase | Logics, Objects |
| External | Usecase, Logics, Objects |

A dependency pointing outward is a design error. Stop and redesign before continuing.

## Objects — What exists

Domain entities with unique identity, mutable state, a lifecycle, and their own invariants. Objects protect their own business rules — never anemic data containers.

```
// Rich — owns its invariant
class Order {
    cancel() {
        if (status == SHIPPED) throw new Error("Cannot cancel shipped order");
        status = CANCELLED;
    }
}

// Anemic — invariant leaked into Logics
class Order { status; }
class OrderLogic {
    cancel(order) { ... }  // ← belongs inside Order
}
```

**Ask**: "Whose business rule is this?" If it depends on the state of one object, it belongs inside that object.
**Invariance principle**: an object must remain the same regardless of which outer layer uses it. If the object changes shape for a specific use case, it has leaked upward.
**No combined objects**: never merge two objects into one to express a relationship. Keep them separate and let Logics or Usecase coordinate them. `ArtifactRepo` combining `Artifact` and `Repo` is a design error — `Artifact` and `Repo` are two objects; their relationship belongs in a Logic.

## Logics — How the domain computes

Stateless domain services that coordinate multiple objects or encapsulate algorithms that don't belong to any single object. No identity, no lifecycle.

```
// Transfer spans two Accounts — belongs to neither
class TransferLogic {
    transfer(source, destination, amount) {
        source.withdraw(amount);
        destination.deposit(amount);
    }
}
```

Use Logics only when the behavior genuinely spans multiple objects. If it only touches one, push it into the object.

Two sub-roles:
- **Algorithm** — workflow, strategy, or computation realizing a domain rule
- **Lens** — cross-cutting concern (auth, billing, audit) that reads from objects without modifying their shape

## Usecase — What the user wants

Encodes what the end user needs. Defines which goals are worth satisfying, what success looks like, and which logics deliver the result. Orchestrates Logics and Objects to fulfill one user goal per use case.
If a user need only lives in a comment or ticket, it hasn't been encoded yet.

## External — How the outside world interacts

Adapters between the application and the outside world: HTTP controllers, CLI commands, UI event handlers, message consumers, external API gateways, files. Translates external input into Usecase calls and Usecase output into external responses.
Contains no business logic. If logic appears here, move it inward.

## Design Smells

| Smell | What it looks like | Layer |
|-------|-------------------|-------|
| Combined object | `ArtifactRepo` merges two objects; split into `Artifact`, `Repo`, coordinate in Logics | Objects |
| Anemic object | Object has state but no behavior; rules live in Logics | Objects |
| Object shaped for one use case | `user.authContext`, `order.billingView` | Objects |
| God object | Evaluates, executes, and models domain | Objects |
| Logic that touches only one object | Should be a method on that object | Logics |
| Logic calling a Usecase | Inner layer depending on outer | Logics |
| Lens holding domain state | `AuthLogic` stores `user.plan` internally | Logics |
| Usecase containing business rules | Rules belong in Objects or Logics | Usecase |
| Usecase skipping Logics | Command directly checks `user.role === 'admin'` | Usecase |
| Business logic in External | Controller deciding domain outcomes | External |
| External calling Objects directly | Bypassing Usecase and Logics | External |
