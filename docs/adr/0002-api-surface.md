# 0002 — Injector and marker API

Status: accepted

## Context

Users describe the object graph in an injector file guarded by the
`solderinject` build tag. The generator replaces the injector body with plain
Go. Marker functions only exist to be read by the generator; they panic if
called. The API should feel familiar to people coming from uber-go/fx, which
is the audience solder wants to win.

## Decision

```go
//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func NewApp(cfg Config) (*solder.App, error) {
	panic(solder.Build(
		solder.Provide(NewLogger, NewDB),
		solder.Lifecycled(NewServer),
		solder.Module("http",
			solder.Provide(NewMux),
			solder.Decorate(WithRequestID),
			solder.Invoke(RegisterRoutes),
		),
		solder.Bind[Store, *PostgresStore](),
		solder.Supply(defaultTimeout),
	))
}
```

| Marker | Meaning |
|---|---|
| `Build(opts...)` | marks an injector; its body is `panic(solder.Build(...))` and nothing else |
| `Provide(fns...)` | each function's result is available to the graph; an optional trailing `error` result is propagated |
| `Lifecycled(fns...)` | like `Provide`, and the result's `Start(context.Context) error` / `Stop(context.Context) error` methods are registered as a hook; always constructed, even if nothing depends on it |
| `Module(name, opts...)` | groups options; `Decorate` inside a module only affects that module and its children; the name appears in errors |
| `Supply(values...)` | values used as-is, keyed by their static type; an untyped constant (`Supply(5)`) is an error — write `Supply(time.Duration(5))` |
| `Bind[I, T]()` | when `I` is needed, use the provided `T` (checked at generation time that `T` implements `I`) |
| `Invoke(fns...)` | called during construction, in order; optional `error` result propagated |
| `Decorate(fns...)` | `func(T, deps...) T` (optionally `(T, error)`) wraps `T` for consumers in the same module and its children |

- Functions passed to markers must be package-level functions.
- Injector parameters are inputs to the graph.
- The injector returns `*solder.App` (and optionally `error`). It may instead
  return any provided type `T` (and optionally `error`) when the graph has no
  `Lifecycle` consumer, no `Lifecycled` and no `Invoke`.
- Providers nobody needs are not constructed (except `Lifecycled`).
- An injector file contains only injectors and imports; other declarations go
  in a normal file.

## Rejected

- **wire's `NewSet` / `Value` / `Struct` / `FieldsOf`**: replaced by
  `Provide`/`Module`/`Supply`; a constructor is one line.
- **`Bind(new(I), new(T))`** (wire style): generics give the same information
  without the pointer trick.
- **Cleanup functions as a second result**: replaced by lifecycle hooks
  (ADR 0003), one mechanism instead of two.
- **Detecting `Start`/`Stop` on every provided type**: hidden behaviour;
  `Lifecycled` makes it an explicit opt-in.

## Deferred (not in v0.1)

`Replace` for test overrides, private (module-scoped) providers, optional
dependencies, flattened groups (`group:"x,flatten"`), decorating named or
group values.
