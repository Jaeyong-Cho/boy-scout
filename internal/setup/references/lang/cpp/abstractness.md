# Abstractness Violations — C++ Example

## Problem

A concrete class (full of implementation details) that many files depend on creates a coupling bottleneck. Changes to implementation force all dependents to recompile, and implementation details are overly visible.

## Solution

Extract the boundary into an abstract interface header (small, 100% abstract), and move the concrete implementation into a separate header. Dependents include only the interface, not the concrete class. Changes to implementation don't affect the abstract boundary, so dependents don't need to recompile.
