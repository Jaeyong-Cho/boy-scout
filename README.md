# boy-scout

Static analysis tool that catches complexity, function-length, file-length, column-length, and duplication violations before code review.

## Installation

See [INSTALL.md](INSTALL.md) for setup and build instructions.

## Usage

```bash
boy-scout <go|cpp|ts> <check|all> [flags] [paths...]
```

Available checks:
- **funclen** (`gofunclen` in Go): Flag functions exceeding a configurable line limit (default: 50)
- **complexity**: Flag functions exceeding a configurable cyclomatic complexity limit (default: 6)
- **filelen**: Flag files exceeding a configurable line limit (default: 300)
- **collen**: Flag physical lines over a configurable character limit (default: 100), exempting quoted-string overflow
- **duplication**: Flag duplicate code in Go and C++

Go and C++ `all` run all five checks. TypeScript `all` runs funclen, filelen, and collen; run `boy-scout ts complexity` separately. TypeScript duplication is not supported.

Run `boy-scout <lang> <check> -help` for check options. Supporting commands include `setup` and `version`.

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
