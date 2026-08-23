# Funclen Violations — Go Example

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
