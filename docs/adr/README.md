# Architecture decision records

One file per decision. Status is `proposed` until the code that implements it
is merged, then `accepted`. A decision that is replaced keeps its file and
points to the one that replaces it.

| # | Decision | Status |
|---|---|---|
| [0001](0001-rewrite-not-fork.md) | Rewrite the generator instead of maintaining a wire fork | accepted |
| [0002](0002-api-surface.md) | Injector and marker API | accepted |
| [0003](0003-lifecycle.md) | Lifecycle: `App`, `Lifecycle`, `Hook` | accepted |
| [0004](0004-names-and-groups.md) | Named values and value groups via `In`/`Out` structs | accepted |
| [0005](0005-performance.md) | Performance budget: hand-written speed, zero runtime allocations | accepted |
