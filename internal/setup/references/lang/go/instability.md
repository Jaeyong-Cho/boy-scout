# Instability Violations — Go Example

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
