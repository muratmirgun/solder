# 0003 — Lifecycle: `App`, `Lifecycle`, `Hook`

Status: accepted

## Context

Long-running services need to start things (listeners, consumers) after the
graph is built and stop them in reverse order on shutdown. fx does this with
a runtime container. solder builds the graph in generated code, so only a
small runtime is needed to run hooks.

## Decision

```go
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

type Lifecycle interface{ Append(Hook) }

func New() *App                          // generated code calls this
func (a *App) Append(h Hook)             // *App is the Lifecycle passed to providers
func (a *App) Start(ctx context.Context) error
func (a *App) Stop(ctx context.Context) error
func (a *App) Run() error                // Start, wait for SIGINT/SIGTERM, Stop
```

- A provider that needs hooks takes a `solder.Lifecycle` parameter; generated
  code passes the `*App`.
- Hooks are appended in construction order, which is dependency order. Start
  runs them in that order; Stop runs them in reverse.
- If a start hook fails, the hooks that already started are stopped in
  reverse and Start returns both errors joined. The rollback gets its own
  15s deadline, because the start context may be the reason start failed.
- Stop calls only hooks whose start succeeded, keeps going when one fails,
  and returns all errors joined. A second Stop is a no-op.
- Constructors must not start background work; that belongs in `OnStart`.
  So a construction error needs no cleanup.
- `Run` uses 15s start and stop timeouts, fx's default.
- No reflection anywhere in the runtime.

## Rejected

- **Detect `Start`/`Stop` methods automatically**: hidden behaviour, and a
  type may have such methods for other reasons. The explicit opt-in is
  `solder.Lifecycled` (ADR 0002).
- **wire-style cleanup results**: cannot express "start later", and every
  provider in the chain must thread the cleanup through.
- **App/Lifecycle safe for concurrent use**: hooks are appended during
  construction and Start/Stop are called once from one goroutine.

## Consequences

`Run`'s timeouts are fixed for now; configurable timeouts are added when
someone needs them.
