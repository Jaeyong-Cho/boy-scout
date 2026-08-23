# Funclen Violations

## Why this is a problem

A function that's too large is doing more than one thing at one level of abstraction. It's mixing distinct concerns — setup, computation, cleanup, error handling, logging — that belong in separate steps. This forces readers to hold too much state in their head at once and makes it hard to test, reuse, or refactor individual steps without affecting others.

**Related concepts:** 
- `functions.md` — The clean-code chapter on functions. Covers one-thing-per-function, abstraction levels, stepdown flow, and when to extract.
- `meta-pattern.md` — explains when code should stay together vs. split. A single function doing one thing well is a form of cohesion; when a function accumulates unrelated responsibilities, it's time to split.
- `naming.md` — good naming of extracted functions is essential; the name should say what each step does.

## How to fix it

Extract each logical step into its own well-named helper function. The original function should read like a table of contents: call `validateInput()`, then `computeResult()`, then `recordMetrics()`, with each helper carrying one level of responsibility. Name each helper so its purpose is obvious without reading its body. Don't just trim lines by removing comments or inlining constants; actually separate the concerns. After extracting, `boy-scout go all` should report the original function fixed and (usually) not create new violations in the newly extracted helpers, because each one now holds just one thing.

## Problem example

```go
func ProcessOrder(orderID string, db *Database, logger Logger) error {
  // Fetch order
  row := db.Query("SELECT * FROM orders WHERE id = ?", orderID)
  order := &Order{}
  if err := row.Scan(&order.ID, &order.Total, &order.Status); err != nil {
    logger.Error("fetch failed", err)
    return err
  }
  
  // Validate
  if order.Total <= 0 {
    logger.Error("invalid total", order.Total)
    return errors.New("total must be positive")
  }
  if order.Status != "pending" {
    logger.Error("invalid status", order.Status)
    return errors.New("order not pending")
  }
  
  // Process payment
  paymentResp, err := callPaymentAPI(order.Total)
  if err != nil {
    logger.Error("payment failed", err)
    return err
  }
  
  // Update database
  _, err = db.Exec("UPDATE orders SET status = ? WHERE id = ?", "paid", orderID)
  if err != nil {
    logger.Error("update failed", err)
    return err
  }
  
  logger.Info("order processed", orderID)
  return nil
}
```

## Good resolve example

```go
func ProcessOrder(orderID string, db *Database, logger Logger) error {
  order, err := fetchOrder(orderID, db, logger)
  if err != nil {
    return err
  }
  
  if err := validateOrder(order, logger); err != nil {
    return err
  }
  
  if err := processPayment(order, logger); err != nil {
    return err
  }
  
  if err := markOrderPaid(orderID, db, logger); err != nil {
    return err
  }
  
  logger.Info("order processed", orderID)
  return nil
}

func fetchOrder(orderID string, db *Database, logger Logger) (*Order, error) {
  row := db.Query("SELECT * FROM orders WHERE id = ?", orderID)
  order := &Order{}
  if err := row.Scan(&order.ID, &order.Total, &order.Status); err != nil {
    logger.Error("fetch failed", err)
    return nil, err
  }
  return order, nil
}

func validateOrder(order *Order, logger Logger) error {
  if order.Total <= 0 {
    logger.Error("invalid total", order.Total)
    return errors.New("total must be positive")
  }
  if order.Status != "pending" {
    logger.Error("invalid status", order.Status)
    return errors.New("order not pending")
  }
  return nil
}

// ... similar helpers for processPayment, markOrderPaid
```

Now `ProcessOrder` reads like a table of contents of the steps, and each step lives in its own helper.
