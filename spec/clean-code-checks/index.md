# Clean code checks

## Business value
Catches clean-code violations — starting with oversized functions — before they land, without a human reviewer having to spot them by eye.

## Completion criteria
Every Story below shipped.

## Overview
The first check implemented is func-length-limit, which flags functions exceeding a configurable line-length limit (default 100). Additional checks (CRAP score, cyclomatic complexity, nesting depth, naming rules) will be added as future Stories in this EPIC.

## Stories
- [func-length-limit](./func-length-limit.md)
