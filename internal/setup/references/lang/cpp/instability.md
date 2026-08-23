# Instability Violations — C++ Example

## Problem example

Stable file including unstable:

```cpp
// domain.hpp - stable (included by everything, changes rarely)
#pragma once
#include "httpapi.hpp"  // INSTABILITY VIOLATION: including unstable file
#include <string>

struct Order {
  std::string id;
  double total;
};

class OrderService {
public:
  void NotifyCustomer(const Order& order) {
    // Stable domain logic calling unstable HTTP API
    SendNotification(order.id, "order ready");
  }
};
```

```cpp
// httpapi.hpp - unstable (changes frequently: routes added, params renamed)
#pragma once
#include <string>

// Frequently changed: routes added, parameters renamed, etc.
bool SendNotification(const std::string& orderID, const std::string& msg);
```

When `httpapi.hpp` changes (new required field, endpoint change, etc.), `domain.hpp` must change too. And everything that includes `domain.hpp` is affected.

## Good resolve example

Invert the dependency — move the glue into the unstable file:

```cpp
// domain.hpp - now stable and independent
#pragma once
#include <string>

struct Order {
  std::string id;
  double total;
};

// No knowledge of how notifications are sent
class NotificationSender {
public:
  virtual ~NotificationSender() = default;
  virtual bool Send(const std::string& orderID, const std::string& msg) = 0;
};

class OrderService {
public:
  explicit OrderService(NotificationSender& sender) : sender_(sender) {}

  void NotifyCustomer(const Order& order) {
    sender_.Send(order.id, "order ready");
  }

private:
  NotificationSender& sender_;
};
```

```cpp
// httpapi.hpp - unstable, but now includes domain.hpp
#pragma once
#include "domain.hpp"
#include <string>

class HTTPNotifier : public NotificationSender {
public:
  bool Send(const std::string& orderID, const std::string& msg) override {
    // ... POST to https://api.example.com/notify
    return true;
  }
};
```

Now `domain.hpp` is stable and doesn't include `httpapi.hpp`. Only `httpapi.hpp` includes `domain.hpp`. When `httpapi.hpp` changes, `domain.hpp` is unaffected.
