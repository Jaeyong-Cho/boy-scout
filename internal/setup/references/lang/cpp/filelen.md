# Filelen Violations — C++ Example

## Problem

A file mixing data model, business logic, and API layer cannot be tested or modified independently. Tests must include everything, and changes to one concern (e.g., storage format) force changes in unrelated code.

## Solution

Split into focused headers: one for the data structure, one for business logic, one for persistence, one for the API layer. Each header now has a single cohesive responsibility, tests can include just what they need, and concerns stay loosely coupled.
