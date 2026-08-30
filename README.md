# boy-scout

Static analysis tool that catches code quality violations — oversized functions, high complexity, poor test coverage, and architectural issues — before they land in code review.

## Installation

See [INSTALL.md](INSTALL.md) for setup and build instructions.

## Usage

```bash
boy-scout [flags] [paths...]
```

Available checks:
- **gofunclen**: Flag Go functions exceeding a configurable line limit (default: 50)
- **complexity**: Flag Go functions exceeding a configurable cyclomatic complexity limit (default: 10)
- **filelen**: Flag files exceeding a configurable line limit
- **linelen**: Flag physical lines over a configurable character limit (default: 100), exempting quoted-string overflow
- **duplication**: Flag duplicate code

Run `boy-scout -help` for all available options and flags.

## License

Proprietary.

## Branching

- **main**: release branch only; production code
- **develop**: integration branch; all features merged here
- **feature branches**: Cut from `develop`, merged back via PR with naming scheme:
  - `feat/<topic>` — new features
  - `fix/<topic>` — bug fixes
  - `refactor/<topic>` — refactoring
  - `docs/<topic>` — documentation
  - `chore/<topic>` — maintenance
  - `perf/<topic>` — performance improvements
  - `test/<topic>` — test additions/fixes

Release process: merge `develop` into `main` (after review), tag, and run `make release`.
