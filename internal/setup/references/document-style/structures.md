# Structures by Document Type

Read this when writing a bug report, README, design doc, meeting notes, work report, or making an argument/comparison.

## 9. Make Problems Concrete

For problems and bugs, prefer:

```text
Environment
-> Reproduction
-> Expected Result
-> Actual Result
-> Evidence
-> Cause
-> Solution
-> Verification
```

Avoid subjective descriptions such as "It doesn't work correctly." Prefer "When X occurs under Y environment, Z is returned instead of A." Use logs, measurements, and reproduction conditions whenever available.

## 10. Make the Reader's Next Action Obvious

When appropriate, explicitly state: what should be done, who should do it, in what order, under what conditions, how success can be verified.

For procedures, write steps in the order the reader performs them.

For technical instructions:

```text
Prerequisites
-> Step 1
-> Step 2
-> Step 3
-> Expected Result
-> Verification
-> Troubleshooting
```

## 11. Use Evidence-Based Reasoning

For arguments and technical decisions:

```text
Claim
-> Reason
-> Evidence
-> Example
```

For causal explanations:

```text
Observation
-> Cause
-> Evidence
-> Consequence
```

Do not mix cause and result.

For comparisons:

```text
Options
-> Evaluation Criteria
-> Comparison
-> Trade-offs
-> Decision
-> Rationale
```

For decisions, explain not only what was chosen but why, and what trade-offs were accepted.

## 12. Choose the Appropriate Document Structure

### README
```text
What is this? / Why does it exist? / Prerequisites / Installation / Usage / Examples / Configuration / Troubleshooting / Limitations
```

### Bug Report
```text
Problem / Environment / Steps to Reproduce / Expected Result / Actual Result / Evidence / Cause / Fix / Verification
```

### Technical Design / Decision
```text
Summary / Context / Problem / Requirements / Options / Evaluation / Decision / Rationale / Trade-offs / Consequences
```

### Meeting Notes
```text
Purpose / Key Decisions / Discussion / Action Items / Owner / Due Date / Open Questions
```

### Work Report
```text
Result / Completed Work / Evidence / Issues / Remaining Work / Next Action
```

Do not include sections that provide no useful information.
