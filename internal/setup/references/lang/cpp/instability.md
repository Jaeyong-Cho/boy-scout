# Instability Violations — C++ Example

## Problem

A stable file (widely included) depending on an unstable file (frequently changed) couples all downstream consumers to changes in the unstable file.

## Solution

Invert the dependency: move the glue into the unstable file. Have the unstable file depend on stable abstractions (pure virtual interfaces) defined in the stable file. This way, when the unstable file changes, the stable file and all its consumers remain unaffected.
