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
solder graph ./...  # print the dependency graph as Mermaid
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
| Dependency graph as Mermaid | `solder graph` | ✅ |
| Benchmarks vs fx | [bench/](bench/) | ✅ |

## Performance

Same app, built with solder and with fx: 12 components, 3 with lifecycle
hooks ([bench/](bench/)). One op = build the graph, start it, stop it.

| | solder | fx v1.24 | |
|---|---|---|---|
| time / op | 127 ns | 113 µs | ~890× faster |
| allocs / op | 12 | 2424 | ~200× fewer |
| memory / op | 464 B | 140 KB | ~300× less |
| stripped binary | 1.6 MB | 4.4 MB | 2.7× smaller |

solder's 12 allocations are the 11 constructors' own structs plus the
`App`; solder itself adds nothing else, and nothing at all to the request
path. Measured on an Apple M4 Max, Go 1.27; reproduce with
`cd bench && go test -bench . -benchmem && ./size.sh`. See
[ADR 0005](docs/adr/0005-performance.md).

## Design

Decisions live in [docs/adr](docs/adr/). solder began from google/wire's code
generation approach (wire is archived) and has been rewritten from scratch
with its own API.

## License

Apache 2.0, see [LICENSE](./LICENSE).
