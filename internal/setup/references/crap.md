# CRAP Violations

## Why this is a problem

CRAP (Change Risk Analysis and Prediction) is high when a function combines high complexity plus low test coverage. A complex function without tests is risky: you can't prove a change is safe. Nobody knows what all its branches do, which ones are dead code, or whether your edit breaks something in an untested path.

**Related concept:** `deep-modules.md` describes good interfaces that are testable. Complex functions are often a sign that the interface is leaky — callers have to do too much setup, or the function has too many parameters. Refactoring to a simpler interface often reduces complexity and makes testing natural.

## How to fix it

First, if coverage is 0%, add one characterization test — a test that captures what the function does right now, not what it should do. The test doesn't have to be pretty; it just has to pass, so you have a safety net for the refactor. Run it once to confirm it passes, then refactor the function to reduce complexity (simpler logic, fewer branches, more locals extracted to helpers). After refactoring, re-run the characterization test to make sure it still passes, then run the full test suite. If coverage is already above 0%, you can skip the characterization test and go straight to refactoring — existing tests plus the full re-run are enough of a guard.

## Problem example

```go
// CRAP score: high complexity, 0% coverage
func calculateDiscount(customerType, season, amount, daysOld) float64 {
  if customerType == "vip" {
    if season == "holiday" {
      if amount > 1000 {
        if daysOld > 365 {
          return 0.25
        } else {
          return 0.20
        }
      } else {
        return 0.15
      }
    } else if season == "summer" {
      if amount > 500 {
        return 0.10
      } else {
        return 0.05
      }
    }
  } else if customerType == "regular" {
    if daysOld > 365 {
      return 0.05
    }
  }
  return 0.0
}
```

## Good resolve example

First, add a characterization test that pins down the current behavior:

```go
func TestCalculateDiscount(t *testing.T) {
  // Characterization test: just document what it does now
  testCases := []struct {
    customerType string
    season       string
    amount       float64
    daysOld      int
    expected     float64
  }{
    {"vip", "holiday", 1500, 400, 0.20},
    {"vip", "holiday", 1500, 500, 0.25},
    {"vip", "summer", 600, 0, 0.10},
    {"regular", "", 1000, 400, 0.0},
    {"regular", "", 1000, 400, 0.05},
  }
  for _, tc := range testCases {
    if got := calculateDiscount(tc.customerType, tc.season, tc.amount, tc.daysOld); got != tc.expected {
      t.Errorf("expected %f, got %f", tc.expected, got)
    }
  }
}
```

Then refactor to reduce nesting and extract complexity:

```go
func calculateDiscount(customerType, season, amount, daysOld) float64 {
  if customerType == "vip" {
    return vipDiscount(season, amount, daysOld)
  } else if customerType == "regular" {
    return regularDiscount(daysOld)
  }
  return 0.0
}

func vipDiscount(season, amount, daysOld) float64 {
  if season == "holiday" {
    return holidayVipDiscount(amount, daysOld)
  }
  if season == "summer" {
    return summerVipDiscount(amount)
  }
  return 0.0
}

func holidayVipDiscount(amount, daysOld) float64 {
  if amount > 1000 && daysOld > 365 {
    return 0.25
  }
  if amount > 1000 {
    return 0.20
  }
  return 0.15
}

func regularDiscount(daysOld) float64 {
  if daysOld > 365 {
    return 0.05
  }
  return 0.0
}
```

Now the test passes and the logic is simpler to understand.
