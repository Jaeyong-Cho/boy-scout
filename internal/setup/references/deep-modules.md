# Deep Modules

A deep module has a **small interface** (few methods, simple parameters) hiding a **large, complex implementation** (lots of hidden logic). This is the ideal shape for infrastructure.

```
┌─────────────────────┐
│   Small Interface   │  ← Few methods, simple params
├─────────────────────┤
│                     │
│  Deep Implementation│  ← Complex logic hidden
│                     │
└─────────────────────┘
```

The opposite is a **shallow module**: many methods, complex parameters, but little actual logic. Shallow modules force callers to do the real work — they don't hide complexity, they just pass it around.

## Why deep modules matter for architecture

A good deep module:
- **Reduces caller burden** — callers don't need to understand internal details
- **Easier to change** — you can refactor the implementation without touching any call sites
- **Fewer dependencies** — simple interface = fewer things callers depend on, so changes ripple less

When many packages depend on you (instability violation), being a deep module means your changes don't break everyone. When you're abstract with many dependents (abstractness violation), a deep module design means the boundary is stable and implementations can change freely.

## How to achieve deep modules

- Ask: what is the **one job** this module does?
- Hide the complexity behind that job inside the module.
- Expose only the smallest interface needed to do that job.
- Refactor internal logic without changing the interface.

If you find yourself adding more and more methods to the interface just to support new callers, you've gone shallow. Extract a new module instead.
