# 0005 — Performance budget: hand-written speed, zero runtime allocations

Status: accepted

## Context

Performance is solder's main selling point against fx. fx resolves the graph
with reflection when the app starts; every constructor call goes through
`reflect.Value.Call`. solder resolves the graph when you run `go generate`.

DI only runs at startup and shutdown. It is not in the request path, so
"as fast as Fiber" is about the HTTP framework a solder app chooses, not
about solder. solder's job is to add **nothing** to the request path and as
little as possible to startup.

## Decision

- Generated code is what a careful person would write by hand: direct
  function calls, local variables, no maps, no interfaces, no closures, no
  `reflect`. It allocates only what the user's constructors allocate.
- The runtime (`App`) allocates once per app (the `App` and its hook
  slice). `Start`, `Stop` and successful hook runs allocate nothing; a test
  enforces this with `testing.AllocsPerRun`.
- The generator, not the runtime, does all checking: missing dependencies,
  cycles, duplicate providers and type mismatches are generation errors.
- Every performance claim in the README or a post comes from a benchmark in
  this repository that anyone can re-run (`go test -bench`).

## Consequences

- Features that would need runtime lookups (resolving a type by name at
  runtime, dynamic `Replace`) are out of scope.
- Benchmarks against fx (startup time, allocations, binary size) are part of
  the v0.1 work, not an afterthought.
