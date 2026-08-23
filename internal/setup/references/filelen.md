# Filelen Violations

## Why this is a problem

A file that's too large is mixing multiple concerns. It holds more than one job — maybe it defines the data model, the business logic, and the API transport layer all in one file. This makes the file hard to understand (readers must hold all concerns in mind at once), hard to test (one concern's test can't avoid importing the others), and hard to reuse (can't take one concern without the whole file).

## How to fix it

Split the file along natural seams, where each concern naturally separates. high cohesion: each new file has one clear job, and its functions work closely together toward that job. loose coupling: each file knows the minimum it needs to know about the others — import from the other file only the small public interface, not internal details. After splitting, `boy-scout go all` should report the original file fixed. If the split creates new violations (e.g., a new instability violation because the seam you chose created a dependency cycle), fix those in later steps.

## Problem example

One file mixing data model, business logic, and HTTP API:

```go
// order.go (too big, mixing concerns)
package order

type Order struct {
  ID     string
  Total  float64
  Status string
}

func (o *Order) Save(db *Database) error {
  // Business logic: validate
  if o.Total <= 0 {
    return errors.New("invalid total")
  }
  // Data persistence: save
  _, err := db.Exec("UPDATE orders SET status = ? WHERE id = ?", o.Status, o.ID)
  return err
}

func (o *Order) Refund() error {
  // Business logic: process refund
  if o.Status != "paid" {
    return errors.New("can only refund paid orders")
  }
  o.Status = "refunded"
  // Data persistence: save
  return o.Save(db)
}

// HTTP API layer mixed in
func HandleOrderAPI(w http.ResponseWriter, r *http.Request) {
  // Parse request
  id := r.URL.Query().Get("id")
  
  // Load order
  order := &Order{}
  row := db.Query("SELECT * FROM orders WHERE id = ?", id)
  if err := row.Scan(&order.ID, &order.Total, &order.Status); err != nil {
    http.Error(w, err.Error(), http.StatusNotFound)
    return
  }
  
  // Process refund
  if err := order.Refund(); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  
  // Return response
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(order)
}
```

## Good resolve example

Split into three focused files:

```go
// model.go - just the data structure
package order

type Order struct {
  ID     string
  Total  float64
  Status string
}

// business.go - business logic (high cohesion, loose coupling with storage)
package order

func (o *Order) ValidateForSave() error {
  if o.Total <= 0 {
    return errors.New("invalid total")
  }
  return nil
}

func (o *Order) CanRefund() bool {
  return o.Status == "paid"
}

// storage.go - data persistence only (loose coupling with business logic)
package order

func (o *Order) Save(db *Database) error {
  if err := o.ValidateForSave(); err != nil {
    return err
  }
  _, err := db.Exec("UPDATE orders SET status = ? WHERE id = ?", o.Status, o.ID)
  return err
}

func LoadOrder(db *Database, id string) (*Order, error) {
  order := &Order{}
  row := db.Query("SELECT * FROM orders WHERE id = ?", id)
  if err := row.Scan(&order.ID, &order.Total, &order.Status); err != nil {
    return nil, err
  }
  return order, nil
}

// handler.go - HTTP API only (imports just what it needs)
package handler

import "myapp/order"

func HandleRefund(w http.ResponseWriter, r *http.Request) {
  id := r.URL.Query().Get("id")
  
  o, err := order.LoadOrder(db, id)
  if err != nil {
    http.Error(w, err.Error(), http.StatusNotFound)
    return
  }
  
  if !o.CanRefund() {
    http.Error(w, "cannot refund", http.StatusBadRequest)
    return
  }
  
  o.Status = "refunded"
  if err := o.Save(db); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(o)
}
```

Now each file has one clear job, tests can import just what they need, and changes to storage don't force API changes.
