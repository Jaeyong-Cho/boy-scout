# Abstractness Violations — Go Example

## Problem

A concrete package (full of implementation details) that many other packages depend on creates a coupling bottleneck. Changes to implementation force all dependents to recompile, and implementation details are overly visible.

## Solution

Extract the boundary into an abstract interface package (small, interface-focused), and move the concrete implementation into a separate package. Dependents import only the interface, not the concrete type. Changes to implementation don't affect the abstract boundary, so dependents don't need to recompile.
