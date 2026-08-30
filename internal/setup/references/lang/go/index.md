# Go Language Guide

This guide covers how to use boy-scout for Go codebases.

## Running Boy-Scout

Run `go test ./...` once before making any changes to ensure tests are green. Then run:

```bash
boy-scout go all
```

This command checks for all five violation kinds in your Go code.

## Test Files

Go test files use the `_test.go` suffix. For example: `main_test.go`, `handlers_test.go`.

## Available Checks

Boy-scout for Go supports the following violation kinds:

1. **funclen** (or `gofunclen`) — Function length violations
2. **filelen** — File length violations
3. **duplication** — Duplicate code pattern violations

## Ignore Comments

To ignore a violation, add the comment `// boy-scout:ignore` on the line with the violation:

```go
func SomeFunction() { // boy-scout:ignore
  // ...
}
```

To ignore a specific violation kind (e.g., just duplication but not funclen), use the kind-scoped directive `// boy-scout:ignore:{kind}`:

```go
func CalculateTax(base float64) float64 { // boy-scout:ignore:duplication
  return base * 0.08
}
```

This excludes the function from duplication comparison but allows other violations (funclen, complexity, etc.) to still be checked and reported.
