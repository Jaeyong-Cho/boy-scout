# Refactoring After Green

Only refactor when all tests passing. Never refactor while RED.

## Checklist

- [ ] Extract duplicated logic into one place
- [ ] Deepen modules — move complexity behind simpler interfaces (see `deep-modules.md`)
- [ ] Apply SOLID principles where they arise naturally — don't force them
- [ ] Consider what new code reveals about existing code — does anything nearby need to change?
- [ ] Run all tests after each refactor step

## Rule

Refactoring changes structure, not behavior. If test breaks during refactor, you changed behavior — undo and try again.

## What not to do

- Don't refactor speculatively ("this might be useful later")
- Don't extract abstractions until you have three or more instances of same thing
- Don't redesign while RED — get to green first, always
