# solder

> 🚧 **Under construction — active development.** solder is pre-v0.1. The API
> can change between commits. Try it, open issues, but don't ship it yet.

[![tests](https://github.com/muratmirgun/solder/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/muratmirgun/solder/actions)

**fx ergonomics, compile-time wiring.** solder is dependency injection for Go
that runs at `go generate` time. You get fx-style modules, lifecycle hooks,
named values and groups. The output is plain Go: direct calls, no
reflection, and a missing dependency is a generation error, not a startup
panic.

## Example

```go
//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func NewApp(cfg Config) (*solder.App, error) {
	panic(solder.Build(
		solder.Provide(NewLogger, NewDB),
		solder.Lifecycled(NewServer), // Start/Stop become hooks
		solder.Module("http",
			solder.Provide(NewMux),
			solder.Decorate(WithRequestID), // only inside this module
			solder.Invoke(RegisterRoutes),
		),
	))
}
```

`solder gen` writes `solder_gen.go`:

```go
func NewApp(cfg Config) (*solder.App, error) {
	app := solder.New()
	logger := NewLogger(app)
	db, err := NewDB(cfg)
	if err != nil {
		return nil, err
	}
	...
	return app, nil
}
```

and `main` runs it:

```go
app, err := NewApp(cfg)
if err != nil {
	log.Fatal(err)
}
log.Fatal(app.Run()) // starts hooks, waits for SIGINT/SIGTERM, stops in reverse
```

## Install

```shell
go install github.com/muratmirgun/solder/cmd/solder@latest
```

```shell
solder gen ./...    # write solder_gen.go files
solder check ./...  # CI: fail on errors or out-of-date files
```

## Features (v0.1 target)

| Feature | API | State |
|---|---|---|
| Constructors, errors, injector inputs | `Provide` | ✅ |
| Lifecycle hooks, graceful shutdown | `Lifecycle`, `Lifecycled`, `App.Run` | ✅ |
| Modules, module-scoped decorators | `Module`, `Decorate` | ✅ |
| Named values, value groups | `In` / `Out` + `name:` / `group:` tags | ✅ |
| Interface binding, plain values | `Bind[I, T]()`, `Supply` | ✅ |
| Run code at startup | `Invoke` | ✅ |
| Test overrides, private providers, flattened groups | — | later |
| Benchmarks vs fx, `solder graph` | — | planned |

## Performance

solder only runs at startup and shutdown; it adds nothing to the request
path. The generated code allocates only what your constructors allocate, and
starting plus stopping 8 hooks takes ~25 ns with **0 allocations**
(`go test -bench . .`). Benchmarks against fx are on the way. See
[ADR 0005](docs/adr/0005-performance.md).

## Design

Decisions live in [docs/adr](docs/adr/). solder began from google/wire's code
generation approach (wire is archived) and has been rewritten from scratch
with its own API.

## License

Apache 2.0, see [LICENSE](./LICENSE).
