# TypeScript Language Guide

This guide covers how to use boy-scout for TypeScript codebases.

## Running Boy-Scout

Run your test suite once before making any changes to ensure tests are green. Then run:

```bash
boy-scout ts all
```

This command checks funclen, filelen, and collen. Run `boy-scout ts complexity` separately for complexity violations.

## Available Checks

1. **funclen** — Function length violations
2. **complexity** — Function complexity violations
3. **filelen** — File length violations
4. **collen** — Line length violations

## Limitations

TypeScript support in boy-scout has the following limitations:

- **Duplication** — Not yet supported for TypeScript
