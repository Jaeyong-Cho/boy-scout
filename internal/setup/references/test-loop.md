# Test-Loop

A test-loop is an E2E harness that runs the real system against real inputs and verifies the outputs.

The target is **full situational coverage with the real system** — happy path, edge cases, error conditions, and boundary inputs. If a situation can happen in production, the test-loop should be able to exercise it.

## Recommended Structure

Each scenario gets its own directory:

```
{scenario}/
├── test      # single entry point: clean + run + verify in one command
├── run       # execute the system, write outputs
├── verify    # read outputs, check results
└── data/     # input data, fixtures, seed files
```

The parent location adapts to the project. Common conventions: `tests/e2e/{scenario}/`, `e2e/{scenario}/`, `scripts/test/{scenario}/`. Use whatever fits the existing project layout.

## Test (entry point)

`test` — the easy way to run the whole loop in one command: `run` then `verify`.

This is what you invoke during development. `run` and `verify` exist separately so you can re-verify previous output without re-running.

## Run

`run` — reset to clean state, then execute the system and write all outputs to a result directory:
1. Reset to clean state — delete output dirs, clear caches, reset DB, initialize seed data and default config
2. Execute the system
3. Write outputs: system results, metadata (version, input data, timestamps, config, environment info), debug output and logs

## Verify

`verify` — read the result directory and check outputs:
- Reads metadata written by `run` to know what to verify
- Compares actual outputs against expected per scenario
- Reports: good / unexpected / ambiguous — with root cause from debug output and logs

Run `run` once, run `verify` many times. Compare output dirs across versions to spot regressions.

## Where this fits in the workflow

**Architecture (`/archi`)** — reuse before creating. Check if an existing test-loop scenario already covers the needed behaviors; extend it rather than creating a new scenario. Only create a new scenario when structurally needed.

Design explicitly:
- What is the clean state?
- What does `run` write? (results, metadata: version, input data, config)
- What does `verify` check per scenario?

**After execution** — run the test-loop and use its output as the primary evaluation signal:
- Run `test` for a full cycle, or `run` then `verify` separately
- For every unexpected result: trace root cause through debug output and logs
