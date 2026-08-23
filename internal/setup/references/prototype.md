# Prototype (throwaway code as the experiment method)

Use when the question is "does this state model / logic feel right?" or "what should this look like?" — cases where reading code can't resolve it but a full production build is overkill.

## Pick a branch
- **Logic / state model question** → `prototype/LOGIC.md`. One shareable HTML file with free-play buttons plus guided walkthroughs, driving the state machine through the cases that are hard to reason about on paper. A non-developer should be able to drive it.
- **UI / look question** → `prototype/UI.md`. Several structurally different variations on one route, switchable via a URL search param and a floating bottom bar.

## Rules
1. **Throwaway from day one, clearly marked.** Live next to the module/page it's prototyping, but named so a casual reader sees it's disposable.
2. **Trivial to run.** One command (`pnpm <name>`, `python <path>`) or a double-clickable HTML file. No setup.
3. **No persistence by default.** State lives in memory unless persistence is the thing being tested — then use an obviously-named scratch DB/file.
4. **Skip the polish.** No tests, no error handling beyond what's needed to run, no abstractions.
5. **Surface the state.** Print/render the full relevant state after every action or variant switch.
6. **Capture it when done.** Fold the validated decision into real code; commit the prototype itself to a throwaway branch (out of main) and record the verdict + a pointer to that branch on the issue/commit.
