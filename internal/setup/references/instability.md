# Instability Violations

## Why this is a problem

An instability violation means a package is importing something less stable than itself — the least-stable thing it leans on. Stability is the inverse of changeability: a stable package has few reasons to change (other code depends on it, so changes are risky and rare); an unstable package changes often (it has few dependents, so changes are safe). When a stable package imports an unstable one, it gets dragged into that instability. A small change deep in the unstable package can force the stable package — and everything that depends on it — to change too. This breaks the principle that stable packages should protect their dependents from change.

**Related concepts:** `meta-pattern.md` explains that stable packages are those you've chosen to rely on — you pay the cost of their changes; unstable packages are still evolving. `deep-modules.md` shows how a stable, small interface lets implementation changes hide inside without forcing dependents to change.

## How to fix it

Point the dependency the other way. If `domain` (stable, depended on by everything) is importing `httpapi` (unstable, changes to add new routes), invert it: move the glue that lives in `domain` but calls `httpapi` into `httpapi` instead. `domain` no longer imports `httpapi`, only `httpapi` imports `domain`. Now when `httpapi` changes, it doesn't force `domain` to change. If a simple import inversion is impossible (it would create a cycle), you need to extract a new package that both can depend on — one small, stable, depended-on-by-both package that holds just the boundary interface, not the implementation.

## Problem example

Stable package importing unstable:

```go
// domain/order.go - stable (depended on by everything)
package domain

import "myapp/httpapi"  // INSTABILITY VIOLATION: importing unstable package

type Order struct {
  ID    string
  Total float64
}

func (o *Order) NotifyCustomer(ctx context.Context) error {
  // Stable domain logic calling unstable HTTP API
  return httpapi.SendNotification(ctx, o.ID, "order ready")
}

// httpapi/http.go - unstable (changes frequently)
package httpapi

// Frequently changed: routes added, parameters renamed, etc.
func SendNotification(ctx context.Context, orderID string, msg string) error {
  resp, err := http.PostForm("https://api.example.com/notify", url.Values{
    "order_id": {orderID},
    "message":  {msg},
  })
  // ... handle response
}
```

When `httpapi` changes (new required field, URL change, etc.), `domain` must change too. And everything depending on `domain` is affected.

## Good resolve example

Invert the dependency — move the glue into the unstable package:

```go
// domain/order.go - now stable and independent
package domain

type Order struct {
  ID    string
  Total float64
}

// No knowledge of how notifications are sent
type NotificationSender interface {
  Send(ctx context.Context, orderID string, msg string) error
}

func (o *Order) NotifyCustomer(ctx context.Context, sender NotificationSender) error {
  return sender.Send(ctx, o.ID, "order ready")
}

// httpapi/http.go - unstable, but now imports domain
package httpapi

import "myapp/domain"

type HTTPNotifier struct {
  endpoint string
}

func (h *HTTPNotifier) Send(ctx context.Context, orderID string, msg string) error {
  resp, err := http.PostForm(h.endpoint + "/notify", url.Values{
    "order_id": {orderID},
    "message":  {msg},
  })
  // ... handle response
}
```

Now `domain` is stable and doesn't import `httpapi`. Only `httpapi` imports `domain`. When `httpapi` changes, `domain` is unaffected.
