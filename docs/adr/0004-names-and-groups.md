# 0004 — Named values and value groups via `In`/`Out` structs

Status: accepted

## Context

Two values of the same type (a primary and a replica `*sql.DB`) need names.
Many providers contributing to one list (HTTP routes, health checks) need
groups. Types alone cannot express either.

## Decision

Parameter and result structs, as in fx, marked by embedding `solder.In` or
`solder.Out`, with struct tags read by the generator:

```go
type DBs struct {
	solder.Out
	Primary *sql.DB `name:"primary"`
	Replica *sql.DB `name:"replica"`
}

type ServerParams struct {
	solder.In
	DB     *sql.DB `name:"primary"`
	Routes []Route `group:"routes"`
}

type RouteResult struct {
	solder.Out
	Route Route `group:"routes"`
}
```

- The graph key is (type, name). Unnamed is the empty name.
- A group key collects every value tagged with that group into a `[]T`, in
  provider declaration order, so generated code is deterministic.
- Tags are read with `go/types` at generation time; nothing is reflected at
  runtime.

- Groups are not flattened in v0.1: a provider contributes one value per
  tagged field.

## Rejected

- **Generic wrappers (`solder.Named[T, Tag]`)**: Go generics take no string
  constants, so every name would need its own type; it leaks into every
  signature.
- **Name arguments on markers (`Provide(NewDB, Name("primary"))`)**: does not
  cover providers with several results or consumers that need two names.

## Consequences

People who know fx can move their `fx.In`/`fx.Out` structs over mostly by
renaming the import.
