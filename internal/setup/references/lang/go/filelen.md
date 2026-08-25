# Filelen Violations — Go Example

## Problem

A file mixing data model, business logic, and HTTP API layers cannot be tested or modified independently. Tests must import everything, and changes to one concern (e.g., storage format) force changes in unrelated code.

## Solution

Split into focused files: one for the data model, one for business logic, one for persistence, one for the API layer. Each file now has a single cohesive job, tests can import just what they need, and concerns stay loosely coupled.
