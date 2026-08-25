# Funclen Violations — C++ Example

## Problem

Long functions mix distinct concerns — setup, computation, cleanup, error handling — in a single routine, forcing readers to hold too much context in their head.

## Solution

Extract each logical step into its own well-named helper function. The original should read like a table of contents: `FetchOrder()`, `ValidateOrder()`, `ProcessPayment()`, `MarkOrderPaid()`. Name each helper so its purpose is obvious without reading its body. This separates concerns so each function holds one responsibility.
