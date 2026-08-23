# Writing Good Tests

Tests verify behavior through public interfaces — not implementation details.

## Good test

```typescript
// Describes what the system does from the outside
it("user can checkout with a valid cart", async () => {
  const cart = Cart.withItems([item("book", 10)])
  const result = await checkout(cart, validPayment())
  expect(result.status).toBe("confirmed")
  expect(result.orderId).toBeDefined()
})
```

- Tests user-facing outcome
- Uses only public interface (`checkout`)
- Survives any internal refactor

## Bad test

```typescript
// Coupled to implementation
it("calls PaymentGateway.charge", async () => {
  const gateway = jest.fn()
  await checkout(cart, gateway)
  expect(gateway).toHaveBeenCalledWith(10)
})
```

- Tests how, not what
- Breaks when you rename or restructure internals
- Passes even if behavior is wrong

## Rule

If test breaks after refactor but behavior hasn't changed, test was wrong.
Good test reads like specification — says what system can do, not how it does it.
