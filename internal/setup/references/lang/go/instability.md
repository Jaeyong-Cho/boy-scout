# Instability Violations — Go Example

## Problem

A stable package (widely depended on) importing an unstable package (frequently changed) couples all downstream consumers to changes in the unstable package.

## Solution

Invert the dependency: move the glue into the unstable package. Have the unstable package depend on stable abstractions (interfaces) defined in the stable package. This way, when the unstable package changes, the stable package and all its consumers remain unaffected.
