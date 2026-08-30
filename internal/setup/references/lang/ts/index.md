# TypeScript Language Guide

This guide covers how to use boy-scout for TypeScript codebases.

## Running Boy-Scout

Run your test suite once before making any changes to ensure tests are green. Then run:

```bash
boy-scout ts all
```

This command checks all violation kinds in your TypeScript code.

## Available Checks

Boy-scout for TypeScript supports the following violation kinds:

1. **funclen** — Function/method length violations
2. **complexity** — Function/method complexity violations
3. **filelen** — File length violations
4. **cohesion** — Class cohesion violations
5. **linelen** — Line length violations

## Limitations

TypeScript support in boy-scout has the following limitations:

- **Duplication** — Not yet supported for TypeScript

Future versions of boy-scout may expand TypeScript support to include duplicate code detection.
