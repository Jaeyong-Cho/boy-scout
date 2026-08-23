# Installation

## Prerequisites

- **Go 1.24+** — [Download Go](https://golang.org/dl/)
- **Git** — for cloning the repository

## Clone the repository

```bash
git clone https://github.com/yourusername/boy-scout.git
cd boy-scout
```

## Build and Install

Use the Makefile for a clean, consistent build:

```bash
make build
```

This compiles the `boy-scout` binary to `./bin/boy-scout`.

For development installation (to `~/.agents/bin/boy-scout`):

```bash
make install
```

## Verify

Run the full test suite to confirm your setup:

```bash
make check
```

This runs `go vet` and all tests. A clean output confirms everything is working.

## Git Hooks

To enable commit hooks (enforces Conventional Commits):

```bash
make install-hooks
```

This sets `git config core.hooksPath .githooks` in your local clone.
