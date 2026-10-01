# Changelog

All notable changes to boy-scout will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.10.0

### Breaking Changes

- Remove cohesion checking for Go, C++, and TypeScript. Retain only complexity,
  function-length, file-length, column-length, and duplication categories.
- Rename `linelen` to `collen` without a compatibility alias. Update existing
  commands and scripts, combined JSON keys, and text-label consumers to `collen`.

### Changed

- Limit bundled setup guidance to retained checks and their language references.
  Re-run `boy-scout setup` to refresh installed guidance; previously installed,
  unreferenced files are not automatically deleted.
- Use report-typed checker configurations and concrete aggregate reports instead
  of report casts and reflection-based result counting.
- Preserve existing detection rules, thresholds, exclusions, and supporting
  `setup`, `all`, and `version` commands. Go retains the `gofunclen` command.
- Preserve TypeScript's current aggregate: `ts all` runs funclen, filelen, and
  collen; run `ts complexity` separately. TypeScript duplication remains unsupported.

## v0.1.0
