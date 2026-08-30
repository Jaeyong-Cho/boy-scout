# C++ Language Guide

This guide covers how to use boy-scout for C++ codebases.

## Running Boy-Scout

Run your test suite once before making any changes to ensure tests are green. Then run:

```bash
boy-scout cpp funclen
```

## Available Checks

Boy-scout for C++ currently supports:

1. **funclen** — Function length violations
2. **complexity** — Function complexity violations
3. **filelen** — File length violations
4. **cohesion** — Class/struct cohesion violations
5. **duplication** — Duplicate code detection

## Limitations

C++ support in boy-scout is currently limited. The following features are not yet supported:

- **CRAP scoring** — Coverage-based CRAP analysis is not available for C++
- **Ignore comments** — The `boy-scout:ignore` comment syntax is not yet supported for C++

Future versions of boy-scout may expand C++ support to include these features.
