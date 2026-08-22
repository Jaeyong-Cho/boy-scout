# Clean code checks

## Business value
Catches clean-code violations — starting with oversized functions — before they land, without a human reviewer having to spot them by eye.

## Completion criteria
Every Story below shipped.

## Overview
The first check implemented is func-length-limit, which flags functions exceeding a configurable line-length limit (default 50). The second check is CRAP score, which flags functions combining high cyclomatic complexity with low test coverage. Additional checks (cyclomatic complexity, nesting depth, naming rules) will be added as future Stories in this EPIC.

## Stories
- [func-length-limit](./func-length-limit.md)
- [crap-score](./crap-score.md)
- [crap-ignores-test-files](./crap-ignores-test-files.md)
- [exclude-files-and-functions](./exclude-files-and-functions.md)
- [selective-checker-ignore](./selective-checker-ignore.md)
