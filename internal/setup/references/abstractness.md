# Abstractness Violations

## Why this is a problem

A package lives in one of two healthy zones: either it's abstract (mostly interfaces and types, few or no concretions, depended on by many) or it's concrete (lots of implementations, few dependents). The Zone of Pain is being concrete and depended on by many — your concrete details become everyone else's problem. The Zone of Uselessness is being abstract with no dependents — abstraction with no purpose. An abstractness violation means you've wandered into one of these zones. Usually it's the Zone of Pain: you've built a concrete package (lots of implementation, few interfaces) that many other packages depend on, so changes to it ripple everywhere.

## How to fix it

Extract the stable boundary (the interfaces and types that your dependents actually use) into a separate small, abstract, deep module. Move the concrete implementations into a different package. Now the dependents import only the abstract boundary, not the concrete details. Changes to the implementations don't touch the boundary, so dependents don't need to change. If your package is in the Zone of Uselessness (abstract but unused), question whether it should exist — it may be premature abstraction. More often, it's in the Pain zone and needs the extraction fix above.

## Problem example

Concrete package depended on by many (Zone of Pain):

```go
// cache/cache.go - concrete, lots of implementation, many dependents
package cache

import "sync"

type MemoryCache struct {
  mu    sync.RWMutex
  items map[string]interface{}
  ttl   map[string]time.Time
}

func NewMemoryCache() *MemoryCache {
  return &MemoryCache{items: make(map[string]interface{})}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
  c.mu.RLock()
  defer c.mu.RUnlock()
  if exp, exists := c.ttl[key]; exists && exp.Before(time.Now()) {
    delete(c.items, key)
    return nil, false
  }
  val, ok := c.items[key]
  return val, ok
}

func (c *MemoryCache) Set(key string, val interface{}, ttl time.Duration) {
  c.mu.Lock()
  defer c.mu.Unlock()
  c.items[key] = val
  c.ttl[key] = time.Now().Add(ttl)
}

// ... 100+ more lines of implementation details
```

Many packages depend on this: `auth`, `session`, `user`, `config` all import `cache.MemoryCache` directly. When you change the locking strategy, add eviction policies, or optimize memory, all dependents must recompile.

## Good resolve example

Extract the boundary into an abstract interface package, move implementation elsewhere:

```go
// cacheapi/interface.go - small, abstract, deep module (the boundary)
package cacheapi

type Cache interface {
  Get(key string) (interface{}, bool)
  Set(key string, val interface{}, ttl time.Duration)
}
```

```go
// memcache/impl.go - concrete implementation (fewer dependents)
package memcache

import (
  "sync"
  "myapp/cacheapi"
)

type MemoryCache struct {
  mu    sync.RWMutex
  items map[string]interface{}
  ttl   map[string]time.Time
}

func NewMemoryCache() cacheapi.Cache {  // Returns interface, not concrete type
  return &MemoryCache{items: make(map[string]interface{})}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
  // ... same implementation as before
}

func (c *MemoryCache) Set(key string, val interface{}, ttl time.Duration) {
  // ... same implementation as before
}
```

```go
// auth/auth.go - now imports only the interface
package auth

import "myapp/cacheapi"

type AuthService struct {
  cache cacheapi.Cache
}

func NewAuthService(cache cacheapi.Cache) *AuthService {
  return &AuthService{cache: cache}
}
```

Now `auth` depends only on the abstract `cacheapi.Cache` interface. Changes to `memcache` implementation don't touch the boundary, so `auth` doesn't need to recompile. The deep module (`cacheapi`) is small and stable; the concrete details live in `memcache` where they're less visible.
