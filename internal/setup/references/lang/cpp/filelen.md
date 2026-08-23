# Filelen Violations — C++ Example

## Problem example

One file mixing data model, business logic, and HTTP API:

```cpp
// order.cpp (too big, mixing concerns)
#include <string>
#include <stdexcept>

struct Order {
  std::string id;
  double total;
  std::string status;
};

bool SaveOrder(Order& order, Database& db) {
  // Business logic: validate
  if (order.total <= 0) {
    throw std::runtime_error("invalid total");
  }
  // Data persistence: save
  return db.execute("UPDATE orders SET status = ? WHERE id = ?", order.status, order.id);
}

bool RefundOrder(Order& order, Database& db) {
  // Business logic: process refund
  if (order.status != "paid") {
    throw std::runtime_error("can only refund paid orders");
  }
  order.status = "refunded";
  // Data persistence: save
  return SaveOrder(order, db);
}

// HTTP API layer mixed in
void HandleOrderAPI(const Request& req, Response& res, Database& db) {
  // Parse request
  std::string id = req.query("id");

  // Load order
  Order order = db.query("SELECT * FROM orders WHERE id = ?", id);

  // Process refund
  try {
    RefundOrder(order, db);
  } catch (const std::exception& e) {
    res.status(400).send(e.what());
    return;
  }

  // Return response
  res.header("Content-Type", "application/json").send(order.toJSON());
}
```

## Good resolve example

Split into focused files:

```cpp
// order.hpp - just the data structure
#pragma once
#include <string>

struct Order {
  std::string id;
  double total;
  std::string status;
};
```

```cpp
// order_business.hpp - business logic (high cohesion, loose coupling with storage)
#pragma once
#include "order.hpp"

bool CanRefund(const Order& order);
void ValidateForSave(const Order& order);  // throws on invalid
```

```cpp
// order_storage.hpp - data persistence only (loose coupling with business logic)
#pragma once
#include "order.hpp"
#include "order_business.hpp"

bool SaveOrder(Order& order, Database& db) {
  ValidateForSave(order);
  return db.execute("UPDATE orders SET status = ? WHERE id = ?", order.status, order.id);
}

Order LoadOrder(Database& db, const std::string& id) {
  return db.query("SELECT * FROM orders WHERE id = ?", id);
}
```

```cpp
// order_handler.hpp - HTTP API only (includes just what it needs)
#pragma once
#include "order_storage.hpp"
#include "order_business.hpp"

void HandleRefund(const Request& req, Response& res, Database& db) {
  std::string id = req.query("id");
  Order order = LoadOrder(db, id);

  if (!CanRefund(order)) {
    res.status(400).send("cannot refund");
    return;
  }

  order.status = "refunded";
  SaveOrder(order, db);
  res.header("Content-Type", "application/json").send(order.toJSON());
}
```

Now each file has one clear job, tests can include just what they need, and changes to storage don't force API changes.
