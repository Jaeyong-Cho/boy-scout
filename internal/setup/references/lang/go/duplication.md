# Duplication Violations — Go Example

## Problem example

```go
func CalculateTax(baseAmount float64) float64 {
	taxRate := 0.08
	return baseAmount * taxRate
}

func CalculateFee(baseAmount float64) float64 {
	feeRate := 0.05
	return baseAmount * feeRate
}
```

Both functions perform the same percentage calculation: multiply the base by a rate. The duplication is the shared algorithm — `base * rate` — appearing in two places.

## Good resolve example

Extract the shared calculation into one helper:

```go
func calculatePercentage(base float64, rate float64) float64 {
	return base * rate
}

func CalculateTax(baseAmount float64) float64 {
	return calculatePercentage(baseAmount, 0.08)
}

func CalculateFee(baseAmount float64) float64 {
	return calculatePercentage(baseAmount, 0.05)
}
```

Now the core algorithm — `base * rate` — lives in one place. If the calculation ever needs to change (e.g., to handle edge cases, apply rounding, or log calls), one edit fixes both callers. Adding a new percentage-based calculation (discount, rebate, etc.) just calls `calculatePercentage` again.

The extracted helper is private (`calculatePercentage` with a lowercase initial) because it's internal to this package. If similar calculations appear in other packages, the solution escalates: extract to a shared package (e.g., `internal/math`), export the helper, and import it in both packages.

After extraction, run your test suite to confirm `CalculateTax` and `CalculateFee` still produce the same results. Boy-scout will no longer report this cluster as a violation.
