# Funclen Violations — C++ Example

## Problem example

```cpp
#include <string>
#include <stdexcept>

struct Order {
  std::string id;
  double total;
  std::string status;
};

class Database {
public:
  Order query(const std::string& sql, const std::string& id);
  bool execute(const std::string& sql, const std::string& status, const std::string& id);
};

class Logger {
public:
  void error(const std::string& msg, const std::string& detail);
  void info(const std::string& msg, const std::string& detail);
};

bool processPaymentAPI(double amount);

bool ProcessOrder(const std::string& orderID, Database& db, Logger& logger) {
  // Fetch order
  try {
    Order order = db.query("SELECT * FROM orders WHERE id = ?", orderID);
    
    // Validate
    if (order.total <= 0) {
      logger.error("invalid total", std::to_string(order.total));
      return false;
    }
    if (order.status != "pending") {
      logger.error("invalid status", order.status);
      return false;
    }
    
    // Process payment
    if (!processPaymentAPI(order.total)) {
      logger.error("payment failed", orderID);
      return false;
    }
    
    // Update database
    if (!db.execute("UPDATE orders SET status = ? WHERE id = ?", "paid", orderID)) {
      logger.error("update failed", orderID);
      return false;
    }
    
    logger.info("order processed", orderID);
    return true;
  } catch (const std::exception& e) {
    logger.error("fetch failed", e.what());
    return false;
  }
}
```

## Good resolve example

```cpp
#include <string>
#include <stdexcept>

struct Order {
  std::string id;
  double total;
  std::string status;
};

class Database {
public:
  Order query(const std::string& sql, const std::string& id);
  bool execute(const std::string& sql, const std::string& status, const std::string& id);
};

class Logger {
public:
  void error(const std::string& msg, const std::string& detail);
  void info(const std::string& msg, const std::string& detail);
};

bool processPaymentAPI(double amount);

bool FetchOrder(const std::string& orderID, Database& db, Logger& logger, Order& order) {
  try {
    order = db.query("SELECT * FROM orders WHERE id = ?", orderID);
    return true;
  } catch (const std::exception& e) {
    logger.error("fetch failed", e.what());
    return false;
  }
}

bool ValidateOrder(const Order& order, Logger& logger) {
  if (order.total <= 0) {
    logger.error("invalid total", std::to_string(order.total));
    return false;
  }
  if (order.status != "pending") {
    logger.error("invalid status", order.status);
    return false;
  }
  return true;
}

bool ProcessPayment(const Order& order, Logger& logger) {
  if (!processPaymentAPI(order.total)) {
    logger.error("payment failed", order.id);
    return false;
  }
  return true;
}

bool MarkOrderPaid(const std::string& orderID, Database& db, Logger& logger) {
  if (!db.execute("UPDATE orders SET status = ? WHERE id = ?", "paid", orderID)) {
    logger.error("update failed", orderID);
    return false;
  }
  return true;
}

bool ProcessOrder(const std::string& orderID, Database& db, Logger& logger) {
  Order order;
  
  if (!FetchOrder(orderID, db, logger, order)) {
    return false;
  }
  
  if (!ValidateOrder(order, logger)) {
    return false;
  }
  
  if (!ProcessPayment(order, logger)) {
    return false;
  }
  
  if (!MarkOrderPaid(orderID, db, logger)) {
    return false;
  }
  
  logger.info("order processed", orderID);
  return true;
}
```

Now `ProcessOrder` reads like a table of contents of the steps, and each step lives in its own helper function.
