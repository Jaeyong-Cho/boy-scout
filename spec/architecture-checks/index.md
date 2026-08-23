---
type: Spec Epic
title: architecture-checks
description: Flags architectural violations — starting with dependency-direction violations (stable packages depending on unstable ones)
tags: [spec, architecture-checks]
timestamp: 2026-08-23T00:00:00+09:00
---

# Architecture checks

## Business value
Catches architecture violations — starting with packages depending on less-stable ones — before they solidify, without humans manually tracing cross-package import graphs.

## Completion criteria
Every Story below shipped.

## Overview
The first check implemented is instability, which flags packages that are stable (many things depend on them, they depend on little) but depend on packages that are still unstable (churning, high coupling). This is the classic "stable depends on unstable" architectural smell. Additional checks (abstractness, distance-from-main-sequence, deep-module pain-zone refinement) will be added as future Stories in this EPIC.

## Stories
- [instability](./instability.md)
- [abstractness](./abstractness.md)
- [exclude-test-files-from-dependency-graph](./exclude-test-files-from-dependency-graph.md)
- [fix-buildgraph-relative-path-mismatch](./fix-buildgraph-relative-path-mismatch.md)
