# Abstractness Violations — C++ Example

## Problem example

Concrete class included by many (Zone of Pain):

```cpp
// cache.hpp - concrete, lots of implementation, many dependents
#pragma once
#include <string>
#include <unordered_map>
#include <mutex>

class MemoryCache {
public:
  bool Get(const std::string& key, std::string& outValue) {
    std::lock_guard<std::mutex> lock(mu_);
    auto it = items_.find(key);
    if (it == items_.end()) return false;
    outValue = it->second;
    return true;
  }

  void Set(const std::string& key, const std::string& value) {
    std::lock_guard<std::mutex> lock(mu_);
    items_[key] = value;
  }

  // ... 100+ more lines of implementation details

private:
  std::mutex mu_;
  std::unordered_map<std::string, std::string> items_;
};
```

Many files depend on this: `auth.cpp`, `session.cpp`, `user.cpp`, `config.cpp` all `#include "cache.hpp"` and use `MemoryCache` directly. `MemoryCache` has 0 pure virtual methods (fully concrete) and many dependents — the Zone of Pain. When you change the locking strategy, add eviction policies, or optimize memory, every dependent recompiles.

## Good resolve example

Extract the boundary into an abstract interface header, move implementation elsewhere:

```cpp
// cache_api.hpp - small, abstract, deep module (the boundary)
#pragma once
#include <string>

class Cache {
public:
  virtual ~Cache() = default;
  virtual bool Get(const std::string& key, std::string& outValue) = 0;
  virtual void Set(const std::string& key, const std::string& value) = 0;
};
```

```cpp
// memcache.hpp - concrete implementation (fewer dependents)
#pragma once
#include "cache_api.hpp"
#include <unordered_map>
#include <mutex>

class MemoryCache : public Cache {
public:
  bool Get(const std::string& key, std::string& outValue) override {
    // ... same implementation as before
  }

  void Set(const std::string& key, const std::string& value) override {
    // ... same implementation as before
  }

private:
  std::mutex mu_;
  std::unordered_map<std::string, std::string> items_;
};
```

```cpp
// auth.hpp - now includes only the interface
#pragma once
#include "cache_api.hpp"

class AuthService {
public:
  explicit AuthService(Cache& cache) : cache_(cache) {}

private:
  Cache& cache_;
};
```

Now `auth.hpp` depends only on the abstract `Cache` interface (1 pure virtual class = 100% abstract). Changes to `MemoryCache`'s implementation don't touch `cache_api.hpp`, so `auth.hpp` doesn't need to recompile. The deep module (`cache_api.hpp`) is small and stable; the concrete details live in `memcache.hpp` where they're less visible.
