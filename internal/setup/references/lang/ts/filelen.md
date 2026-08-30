# Filelen Violations — TypeScript Example

## Problem

A file mixing multiple concerns (data model, business logic, API handlers) cannot be tested or modified independently. Tests must import everything, and changes to one concern force changes in unrelated code.

## Solution

Split into focused files, each with a single responsibility: one for the data model, one for business logic, one for the API layer. Each file now has one cohesive job, tests can import just what they need, and concerns stay loosely coupled.
