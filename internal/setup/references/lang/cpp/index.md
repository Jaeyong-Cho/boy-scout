# C++ Language Guide

This guide covers how to use boy-scout for C++ codebases.

## Running Boy-Scout

Run your test suite once before making any changes to ensure tests are green. Then run:

```bash
boy-scout cpp all
```

## Available Checks

1. **funclen** — Function length violations
2. **complexity** — Function complexity violations
3. **filelen** — File length violations
4. **collen** — Line length violations
5. **duplication** — Duplicate code detection

## Limitations

See each check's language-specific reference for supported ignore syntax. Use `--exclude-file` and `--exclude-func` where the check supports those exclusions.
