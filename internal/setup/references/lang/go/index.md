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
2. **crap** — CRAP (Change Risk Analysis and Prediction) score violations
3. **filelen** — File length violations
4. **duplication** — Duplicate code pattern violations
5. **instability** — Package dependency instability violations
6. **abstractness** — Package abstraction level violations

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

This excludes the function from duplication comparison but allows other violations (funclen, crap, etc.) to still be checked and reported.

## Note on Test Files

CRAP violations never appear for `_test.go` files by default (no flag needed) — the coverage formula is meaningless for test code by construction, since `go test -coverprofile` never instruments test files. Violations in test files are deferred to the end of the violation list, after every non-test violation.
